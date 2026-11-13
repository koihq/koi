package stripe

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v87"

	"github.com/koihq/koi/pkg/domain"
	"github.com/koihq/koi/pkg/pricing"
)

type checkoutResponse struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	Customer  string `json:"customer"`
	Reference string `json:"client_reference_id"`
	ExpiresAt int64  `json:"expires_at"`
}

func testClient(server *httptest.Server) *Client {
	return newClient("sk_test_unused", &stripe.BackendConfig{
		URL:               stripe.String(server.URL),
		HTTPClient:        server.Client(),
		MaxNetworkRetries: stripe.Int64(0),
	})
}

func TestStripePriceRequiresInclusiveFlatOnlyTiers(t *testing.T) {
	catalog := pricing.Current()
	price := Price{
		ID:            "price_test",
		Active:        true,
		Currency:      "eur",
		BillingScheme: "tiered",
		TaxBehavior:   "inclusive",
		TiersMode:     "volume",
	}
	price.Recurring.Interval = "month"
	price.Recurring.IntervalCount = 1
	price.Recurring.UsageType = "licensed"
	for index, option := range catalog.Options {
		tier := Tier{FlatAmount: int64(option.Euros) * 100}
		if index < len(catalog.Options)-1 {
			upper := option.Prompts
			tier.UpTo = &upper
		}
		price.Tiers = append(price.Tiers, tier)
	}
	if err := price.ValidateCatalog(catalog); err != nil {
		t.Fatal(err)
	}
	price.TaxBehavior = "exclusive"
	if err := price.ValidateCatalog(catalog); err == nil {
		t.Fatal("tax-exclusive Stripe price was accepted")
	}
	price.TaxBehavior = "inclusive"
	zero := int64(0)
	price.Tiers[0].UnitAmount = &zero
	if err := price.ValidateCatalog(catalog); err == nil {
		t.Fatal("explicit zero unit amount was accepted")
	}
	price.Tiers[0].UnitAmount = nil
	decimal := "0.5"
	price.Tiers[0].UnitAmountDecimal = &decimal
	if err := price.ValidateCatalog(catalog); err == nil {
		t.Fatal("decimal unit amount was accepted")
	}
	price.Tiers[0].UnitAmountDecimal = nil
	price.Tiers[0].FlatAmount++
	if err := price.ValidateCatalog(catalog); err == nil {
		t.Fatal("wrong tier amount was accepted")
	}
	price.Tiers[0].FlatAmount--
}

func TestGetPricePreservesAbsentUnitAmount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/prices/price_test" || r.URL.Query().Get("expand[0]") != "tiers" {
			t.Errorf("unexpected Stripe request: %s", r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"price_test","active":true,"currency":"eur","billing_scheme":"tiered","tiers_mode":"volume","tax_behavior":"inclusive","recurring":{"interval":"month","interval_count":1,"usage_type":"licensed"},"tiers":[{"up_to":5,"flat_amount":900,"unit_amount":null},{"up_to":null,"flat_amount":1500,"unit_amount":null}]}`))
	}))
	defer server.Close()

	price, err := testClient(server).GetPrice(t.Context(), "price_test")
	if err != nil {
		t.Fatal(err)
	}
	if len(price.Tiers) != 2 || price.Tiers[0].UnitAmount != nil || price.Tiers[1].UnitAmount != nil {
		t.Fatalf("unit amounts should be absent: %+v", price.Tiers)
	}
}

func TestHostedCheckoutKeepsOrganizationAndCustomerIdentifiers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		for key, want := range map[string]string{
			"mode":                      "subscription",
			"ui_mode":                   "hosted_page",
			"customer":                  "cus_org",
			"client_reference_id":       "org_uuid",
			"line_items[0][price]":      "price_v1",
			"line_items[0][quantity]":   "10",
			"managed_payments[enabled]": "true",
			"subscription_data[metadata][organization_id]": "org_uuid",
			"success_url": "https://koi.example/settings/billing?checkout=success&session_id={CHECKOUT_SESSION_ID}",
		} {
			if got := r.Form.Get(key); got != want {
				t.Errorf("%s = %q, want %q", key, got, want)
			}
		}
		if got := r.Header.Get("Idempotency-Key"); got != "koi-checkout-reservation_uuid" {
			t.Errorf("idempotency key = %q", got)
		}
		for _, key := range []string{
			"automatic_tax[enabled]",
			"tax_id_collection[enabled]",
			"payment_method_types[0]",
			"customer_update[name]",
			"customer_update[address]",
		} {
			if r.Form.Has(key) {
				t.Errorf("Managed Payments checkout must omit %s", key)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(checkoutResponse{
			ID:        "cs_123",
			URL:       "https://checkout.stripe.com/test",
			Customer:  "cus_org",
			Reference: "org_uuid",
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
		})
	}))
	defer server.Close()

	session, err := testClient(server).CreateCheckout(t.Context(), domain.BillingCheckoutInput{CustomerID: "cus_org", OrganizationID: "org_uuid", PriceID: "price_v1", ReservationID: "reservation_uuid", Quantity: 10, PublicURL: "https://koi.example"})
	if err != nil || session.URL == "" {
		t.Fatalf("checkout failed: %+v, %v", session, err)
	}
}

func TestHostedCheckoutTrialRequiresPaymentMethod(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if got := r.Form.Get("subscription_data[trial_period_days]"); got != "7" {
			t.Errorf("trial days = %q, want 7", got)
		}
		if got := r.Form.Get("payment_method_collection"); got != "always" {
			t.Errorf("payment method collection = %q, want always", got)
		}
		if got := r.Form.Get("managed_payments[enabled]"); got != "true" {
			t.Errorf("managed payments = %q, want true", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(checkoutResponse{
			ID:        "cs_trial",
			URL:       "https://checkout.stripe.com/test",
			Customer:  "cus_org",
			Reference: "org_uuid",
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
		})
	}))
	defer server.Close()

	_, err := testClient(server).CreateCheckout(t.Context(), domain.BillingCheckoutInput{CustomerID: "cus_org", OrganizationID: "org_uuid", PriceID: "price_v1", ReservationID: "reservation_uuid", Quantity: 15, PublicURL: "https://koi.example", Trial: true})
	if err != nil {
		t.Fatal(err)
	}
}

func TestEndTrialKeepsTrialUntilPaymentSucceeds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		for key, want := range map[string]string{
			"trial_end":          "now",
			"payment_behavior":   "pending_if_incomplete",
			"proration_behavior": "none",
		} {
			if got := r.Form.Get(key); got != want {
				t.Errorf("%s = %q, want %q", key, got, want)
			}
		}
		if got := r.Header.Get("Idempotency-Key"); got != "start-key" {
			t.Errorf("idempotency key = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"sub_123","customer":"cus_org","status":"trialing","trial_end":900,"pending_update":{"expires_at":500,"trial_end":100},"latest_invoice":{"id":"in_1","hosted_invoice_url":"https://invoice.stripe.com/pay"},"items":{"data":[{"id":"si_123","quantity":25,"current_period_end":200,"price":{"id":"price_v1"}}]}}`))
	}))
	defer server.Close()

	result, err := testClient(server).EndTrial(t.Context(), domain.BillingSnapshot{ID: "sub_123"}, "start-key")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "trialing" || result.PendingUpgradeQuantity == nil || *result.PendingUpgradeQuantity != 25 {
		t.Fatalf("pending plan start was not mapped: %+v", result)
	}
	if result.PaymentURL != "https://invoice.stripe.com/pay" || result.PendingUpgradeExpiresAt == nil || result.PendingUpgradeExpiresAt.Unix() != 500 {
		t.Fatalf("payment recovery was not mapped: %+v", result)
	}
}

func TestUpgradeRequestsPaidPendingProration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		for key, want := range map[string]string{
			"items[0][id]":       "si_123",
			"items[0][quantity]": "25",
			"payment_behavior":   "pending_if_incomplete",
			"proration_behavior": "always_invoice",
		} {
			if got := r.Form.Get(key); got != want {
				t.Errorf("%s = %q, want %q", key, got, want)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"sub_123","customer":"cus_org","status":"active","items":{"data":[{"id":"si_123","quantity":25,"current_period_end":200,"price":{"id":"price_v1"}}]}}`))
	}))
	defer server.Close()

	if _, err := testClient(server).Upgrade(t.Context(), domain.BillingSnapshot{ID: "sub_123", ItemID: "si_123"}, 25, "upgrade-key"); err != nil {
		t.Fatal(err)
	}
}

func TestPhaseMappingPreservesTrialAndBillingSettings(t *testing.T) {
	phase := &stripe.SubscriptionSchedulePhase{
		TrialEnd:             200,
		Metadata:             map[string]string{"organization_id": "org_1"},
		DefaultPaymentMethod: &stripe.PaymentMethod{ID: "pm_1"},
		AutomaticTax:         &stripe.SubscriptionAutomaticTax{Enabled: true},
		Items: []*stripe.SubscriptionSchedulePhaseItem{
			{Price: &stripe.Price{ID: "price_1"}, Quantity: 25},
		},
	}
	mapped, err := phaseParams(phase)
	if err != nil {
		t.Fatal(err)
	}
	if stripe.Int64Value(mapped.TrialEnd) != 200 || stripe.StringValue(mapped.DefaultPaymentMethod) != "pm_1" || !stripe.BoolValue(mapped.AutomaticTax.Enabled) || mapped.Metadata["organization_id"] != "org_1" {
		t.Fatalf("lost phase settings: %+v", mapped)
	}
	phase.ApplicationFeePercent = 5
	if _, err := phaseParams(phase); err == nil {
		t.Fatal("unsupported fee would be silently dropped")
	}
}

func TestTrialChangePreservesEndAndDoesNotInvoice(t *testing.T) {
	end := time.Now().Add(24 * time.Hour)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("proration_behavior") != "none" || r.Form.Get("trial_end") != fmt.Sprint(end.Unix()) || r.Header.Get("Idempotency-Key") != "trial-key" {
			t.Errorf("invalid trial update: %v", r.Form)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"id":"sub_1","customer":"cus_1","status":"trialing","trial_end":%d,"items":{"data":[{"id":"si_1","quantity":10,"price":{"id":"price_1"}}]}}`, end.Unix())
	}))
	defer server.Close()
	result, err := testClient(server).ChangeTrial(t.Context(), domain.BillingSnapshot{ID: "sub_1", ItemID: "si_1", TrialEnd: &end}, 10, "trial-key")
	if err != nil {
		t.Fatal(err)
	}
	if result.TrialEnd == nil || result.TrialEnd.Unix() != end.Unix() {
		t.Fatal("trial end changed")
	}
}

func TestSubscriptionPreservesPendingPayment(t *testing.T) {
	var subscription stripe.Subscription
	err := json.Unmarshal([]byte(`{"id":"sub_1","customer":"cus_1","status":"active","items":{"data":[{"id":"si_1","quantity":10,"price":{"id":"price_1"}}]},"pending_update":{"expires_at":200,"subscription_items":[{"id":"si_1","quantity":25}]},"latest_invoice":{"id":"in_1","hosted_invoice_url":"https://invoice.stripe.com/test"}}`), &subscription)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := subscriptionFromStripe(&subscription)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.PendingUpgradeQuantity == nil || *snapshot.PendingUpgradeQuantity != 25 || snapshot.PaymentURL == "" || snapshot.PendingUpgradeExpiresAt.Unix() != 200 {
		t.Fatalf("lost payment details: %+v", snapshot)
	}
}

type stripeCall struct {
	method string
	path   string
	form   url.Values
	key    string
}

// fakeStripe answers each "METHOD /path" with a status and JSON body and records every call.
func fakeStripe(t *testing.T, routes map[string]func() (int, string)) (*Client, *[]stripeCall) {
	t.Helper()

	calls := &[]stripeCall{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		*calls = append(*calls, stripeCall{
			method: r.Method,
			path:   r.URL.Path,
			form:   r.PostForm,
			key:    r.Header.Get("Idempotency-Key"),
		})

		route, ok := routes[r.Method+" "+r.URL.Path]
		if !ok {
			t.Errorf("unexpected Stripe call %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		status, body := route()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return testClient(server), calls
}

func ok(body string) func() (int, string) {
	return func() (int, string) { return http.StatusOK, body }
}

func TestCreateCustomerIsIdempotentPerOrganization(t *testing.T) {
	client, calls := fakeStripe(t, map[string]func() (int, string){
		"POST /v1/customers": ok(`{"id":"cus_1"}`),
	})

	id, err := client.CreateCustomer(t.Context(), "org_1")

	require.NoError(t, err)
	require.Equal(t, "cus_1", id)
	require.Equal(t, "org_1", (*calls)[0].form.Get("metadata[organization_id]"))
	require.Equal(t, "koi-customer-org_1", (*calls)[0].key)
}

func TestCreateCustomerRejectsAnEmptyCustomer(t *testing.T) {
	client, _ := fakeStripe(t, map[string]func() (int, string){
		"POST /v1/customers": ok(`{}`),
	})

	_, err := client.CreateCustomer(t.Context(), "org_1")

	require.ErrorContains(t, err, "empty customer")
}

func TestCreatePortalReturnsToBillingSettings(t *testing.T) {
	client, calls := fakeStripe(t, map[string]func() (int, string){
		"POST /v1/billing_portal/sessions": ok(`{"id":"bps_1","url":"https://billing.stripe.com/p/1"}`),
	})

	got, err := client.CreatePortal(t.Context(), "cus_1", "https://app.koi.example")

	require.NoError(t, err)
	require.Equal(t, "https://billing.stripe.com/p/1", got)
	require.Equal(t, "cus_1", (*calls)[0].form.Get("customer"))
	require.Equal(t, "https://app.koi.example/settings/billing", (*calls)[0].form.Get("return_url"))
}

func TestRetrieveCheckoutRejectsADifferentSession(t *testing.T) {
	client, _ := fakeStripe(t, map[string]func() (int, string){
		"GET /v1/checkout/sessions/cs_1": ok(`{"id":"cs_other","customer":"cus_1","status":"open"}`),
	})

	_, err := client.RetrieveCheckout(t.Context(), "cs_1")

	require.ErrorContains(t, err, "unexpected checkout")
}

func TestExpireCheckoutReadsTheSessionWhenPaymentWonTheRace(t *testing.T) {
	completed := `{"id":"cs_1","customer":"cus_1","status":"complete","subscription":"sub_1","client_reference_id":"org_1"}`

	t.Run("already completed", func(t *testing.T) {
		client, calls := fakeStripe(t, map[string]func() (int, string){
			"POST /v1/checkout/sessions/cs_1/expire": func() (int, string) {
				return http.StatusBadRequest, `{"error":{"type":"invalid_request_error","message":"Only open sessions can be expired."}}`
			},
			"GET /v1/checkout/sessions/cs_1": ok(completed),
		})

		got, err := client.ExpireCheckout(t.Context(), "cs_1")

		require.NoError(t, err)
		require.Equal(t, "complete", got.Status)
		require.Equal(t, "sub_1", got.SubscriptionID)
		require.Equal(t, "koi-expire-cs_1", (*calls)[0].key)
	})

	t.Run("other failures are returned", func(t *testing.T) {
		client, calls := fakeStripe(t, map[string]func() (int, string){
			"POST /v1/checkout/sessions/cs_1/expire": func() (int, string) {
				return http.StatusUnauthorized, `{"error":{"type":"invalid_request_error","message":"Invalid API key"}}`
			},
		})

		_, err := client.ExpireCheckout(t.Context(), "cs_1")

		require.Error(t, err)
		require.Len(t, *calls, 1)
	})
}

const scheduledSubscription = `{"id":"sub_1","customer":"cus_1","status":"active","schedule":"sub_sched_1","items":{"data":[{"id":"si_1","quantity":25,"current_period_end":2000,"price":{"id":"price_1"}}]}}`

func schedule(nextQuantity int) string {
	return fmt.Sprintf(`{"id":"sub_sched_1","subscription":"sub_1","current_phase":{"start_date":1000,"end_date":2000},"phases":[`+
		`{"start_date":1000,"end_date":2000,"items":[{"price":{"id":"price_1"},"quantity":25}]},`+
		`{"start_date":2000,"items":[{"price":{"id":"price_1"},"quantity":%d}]}]}`, nextQuantity)
}

func TestGetSubscriptionReadsTheScheduledQuantity(t *testing.T) {
	client, _ := fakeStripe(t, map[string]func() (int, string){
		"GET /v1/subscriptions/sub_1":                ok(scheduledSubscription),
		"GET /v1/subscription_schedules/sub_sched_1": ok(schedule(15)),
	})

	got, err := client.GetSubscription(t.Context(), "sub_1")

	require.NoError(t, err)
	require.Equal(t, 25, got.Quantity)
	require.NotNil(t, got.PendingQuantity)
	require.Equal(t, 15, *got.PendingQuantity)
}

func TestGetSubscriptionRejectsAScheduleOfAnotherSubscription(t *testing.T) {
	client, _ := fakeStripe(t, map[string]func() (int, string){
		"GET /v1/subscriptions/sub_1": ok(scheduledSubscription),
		"GET /v1/subscription_schedules/sub_sched_1": ok(
			`{"id":"sub_sched_1","subscription":"sub_other","current_phase":{"start_date":1000,"end_date":2000},"phases":[]}`,
		),
	})

	_, err := client.GetSubscription(t.Context(), "sub_1")

	require.ErrorContains(t, err, "unexpected subscription schedule")
}

func TestScheduleDowngradeSplitsTheCurrentPhaseAtPeriodEnd(t *testing.T) {
	periodEnd := time.Unix(2000, 0)
	next := 25
	client, calls := fakeStripe(t, map[string]func() (int, string){
		"POST /v1/subscription_schedules": ok(`{"id":"sub_sched_1"}`),
		"GET /v1/subscription_schedules/sub_sched_1": func() (int, string) {
			return http.StatusOK, schedule(next)
		},
		"POST /v1/subscription_schedules/sub_sched_1": func() (int, string) {
			next = 15
			return http.StatusOK, schedule(next)
		},
		"GET /v1/subscriptions/sub_1": ok(scheduledSubscription),
	})

	got, err := client.ScheduleDowngrade(t.Context(), domain.BillingSnapshot{
		ID:               "sub_1",
		PriceID:          "price_1",
		Quantity:         25,
		CurrentPeriodEnd: &periodEnd,
	}, 15, "downgrade")

	require.NoError(t, err)
	require.NotNil(t, got.PendingQuantity)
	require.Equal(t, 15, *got.PendingQuantity)

	create, update := (*calls)[0], (*calls)[2]
	require.Equal(t, "sub_1", create.form.Get("from_subscription"))
	require.Equal(t, "downgrade-create-schedule", create.key)
	require.Equal(t, "downgrade-update-schedule", update.key)
	for field, want := range map[string]string{
		"end_behavior":                  "release",
		"proration_behavior":            "none",
		"phases[0][start_date]":         "1000",
		"phases[0][end_date]":           "2000",
		"phases[0][items][0][quantity]": "25",
		"phases[1][start_date]":         "2000",
		"phases[1][items][0][quantity]": "15",
		"phases[1][proration_behavior]": "none",
		"phases[1][duration][interval]": "month",
	} {
		require.Equal(t, want, update.form.Get(field), field)
	}
}

func TestScheduleDowngradeRefusesAScheduleThatDiffersFromTheSubscription(t *testing.T) {
	periodEnd := time.Unix(2000, 0)
	client, calls := fakeStripe(t, map[string]func() (int, string){
		"GET /v1/subscription_schedules/sub_sched_1": ok(schedule(25)),
	})
	scheduleID := "sub_sched_1"

	_, err := client.ScheduleDowngrade(t.Context(), domain.BillingSnapshot{
		ID:               "sub_1",
		PriceID:          "price_other",
		Quantity:         25,
		CurrentPeriodEnd: &periodEnd,
		ScheduleID:       &scheduleID,
	}, 15, "downgrade")

	require.ErrorContains(t, err, "differs from subscription")
	require.Len(t, *calls, 1, "nothing is updated")
}

func TestReleaseScheduleLeavesUnscheduledSubscriptionsAlone(t *testing.T) {
	client, calls := fakeStripe(t, map[string]func() (int, string){})

	current := domain.BillingSnapshot{ID: "sub_1"}
	got, err := client.ReleaseSchedule(t.Context(), current, "release")

	require.NoError(t, err)
	require.Equal(t, current, got)
	require.Empty(t, *calls)
}

func TestReleaseScheduleReleasesAndRereads(t *testing.T) {
	client, calls := fakeStripe(t, map[string]func() (int, string){
		"POST /v1/subscription_schedules/sub_sched_1/release": ok(`{"id":"sub_sched_1"}`),
		"GET /v1/subscriptions/sub_1": ok(
			`{"id":"sub_1","customer":"cus_1","status":"active","items":{"data":[{"id":"si_1","quantity":25,"price":{"id":"price_1"}}]}}`,
		),
	})
	scheduleID := "sub_sched_1"

	got, err := client.ReleaseSchedule(t.Context(), domain.BillingSnapshot{ID: "sub_1", ScheduleID: &scheduleID}, "release")

	require.NoError(t, err)
	require.Nil(t, got.ScheduleID)
	require.Equal(t, "release", (*calls)[0].key)
}
