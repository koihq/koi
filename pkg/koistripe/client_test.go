package koistripe

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stripe/stripe-go/v87"

	"github.com/koihq/koi/pkg/koidomain"
	"github.com/koihq/koi/pkg/koipricing"
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
	catalog := koipricing.Current()
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
			"mode":                    "subscription",
			"ui_mode":                 "hosted_page",
			"customer":                "cus_org",
			"client_reference_id":     "org_uuid",
			"line_items[0][price]":    "price_v1",
			"line_items[0][quantity]": "10",
			"subscription_data[metadata][organization_id]": "org_uuid",
		} {
			if got := r.Form.Get(key); got != want {
				t.Errorf("%s = %q, want %q", key, got, want)
			}
		}
		if got := r.Header.Get("Idempotency-Key"); got != "koi-checkout-reservation_uuid" {
			t.Errorf("idempotency key = %q", got)
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

	session, err := testClient(server).CreateCheckout(t.Context(), koidomain.BillingCheckoutInput{CustomerID: "cus_org", OrganizationID: "org_uuid", PriceID: "price_v1", ReservationID: "reservation_uuid", Quantity: 10, PublicURL: "https://koi.example"})
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

	_, err := testClient(server).CreateCheckout(t.Context(), koidomain.BillingCheckoutInput{CustomerID: "cus_org", OrganizationID: "org_uuid", PriceID: "price_v1", ReservationID: "reservation_uuid", Quantity: 15, PublicURL: "https://koi.example", Trial: true})
	if err != nil {
		t.Fatal(err)
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

	if _, err := testClient(server).Upgrade(t.Context(), koidomain.BillingSnapshot{ID: "sub_123", ItemID: "si_123"}, 25, "upgrade-key"); err != nil {
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
	result, err := testClient(server).ChangeTrial(t.Context(), koidomain.BillingSnapshot{ID: "sub_1", ItemID: "si_1", TrialEnd: &end}, 10, "trial-key")
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
