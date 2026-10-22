package koistripe

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stripe/stripe-go/v87"

	"github.com/koihq/koi/pkg/koipricing"
)

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
	price.Tiers[0].FlatAmount++
	if err := price.ValidateCatalog(catalog); err == nil {
		t.Fatal("wrong tier amount was accepted")
	}
	price.Tiers[0].FlatAmount--
	if amount, err := price.MonthlyAmount(5); err != nil || amount != 900 {
		t.Fatalf("five prompts: %d, %v", amount, err)
	}
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
		_ = json.NewEncoder(w).Encode(CheckoutSession{
			ID:        "cs_123",
			URL:       "https://checkout.stripe.com/test",
			Customer:  "cus_org",
			Reference: "org_uuid",
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
		})
	}))
	defer server.Close()

	session, err := testClient(server).CreateCheckout(t.Context(), "cus_org", "org_uuid", "price_v1", "reservation_uuid", 10, "https://koi.example", false)
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
		_ = json.NewEncoder(w).Encode(CheckoutSession{
			ID:        "cs_trial",
			URL:       "https://checkout.stripe.com/test",
			Customer:  "cus_org",
			Reference: "org_uuid",
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
		})
	}))
	defer server.Close()

	_, err := testClient(server).CreateCheckout(t.Context(), "cus_org", "org_uuid", "price_v1", "reservation_uuid", 15, "https://koi.example", true)
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

	if _, err := testClient(server).Upgrade(t.Context(), "sub_123", "si_123", 25); err != nil {
		t.Fatal(err)
	}
}

func TestDowngradeSchedulesNextPeriodWithoutProration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		for key, want := range map[string]string{
			"proration_behavior":            "none",
			"end_behavior":                  "release",
			"phases[0][start_date]":         "100",
			"phases[0][end_date]":           "200",
			"phases[0][items][0][quantity]": "25",
			"phases[1][start_date]":         "200",
			"phases[1][items][0][price]":    "price_v1",
			"phases[1][items][0][quantity]": "5",
		} {
			if got := r.Form.Get(key); got != want {
				t.Errorf("%s = %q, want %q", key, got, want)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"sub_sched_123"}`))
	}))
	defer server.Close()

	schedule := Schedule{ID: "sub_sched_123"}
	schedule.Phases = append(schedule.Phases, struct {
		StartDate int64
		EndDate   int64
	}{StartDate: 100, EndDate: 200})
	if err := testClient(server).ScheduleDowngrade(t.Context(), schedule, "price_v1", 25, 5, 200); err != nil {
		t.Fatal(err)
	}
}
