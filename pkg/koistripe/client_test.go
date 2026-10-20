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

func TestStripeVolumePriceMatchesEverySelectablePromptCount(t *testing.T) {
	catalog := koipricing.Current()
	price := Price{
		ID:            "price_test",
		Active:        true,
		Currency:      "eur",
		BillingScheme: "tiered",
		TiersMode:     "volume",
		Metadata:      map[string]string{"pricing_version": catalog.Version},
	}
	price.Recurring.Interval = "month"
	price.Recurring.UsageType = "licensed"

	for quantity := 5; quantity <= 1000; quantity++ {
		amount, _ := catalog.Quote(quantity)
		if len(price.Tiers) > 0 && price.Tiers[len(price.Tiers)-1].FlatAmount == amount {
			continue
		}
		if len(price.Tiers) > 0 {
			previous := quantity - 1
			price.Tiers[len(price.Tiers)-1].UpTo = &previous
		}
		price.Tiers = append(price.Tiers, Tier{FlatAmount: amount})
	}

	if err := price.ValidateCatalog(catalog); err != nil {
		t.Fatal(err)
	}
	if got, err := price.MonthlyAmount(6); err != nil || got != 1000 {
		t.Fatalf("six prompts: %d, %v", got, err)
	}
	for _, quantity := range []int{4, 1001} {
		if _, err := price.MonthlyAmount(quantity); err == nil {
			t.Fatalf("quantity %d was accepted", quantity)
		}
	}
}

func TestGetPriceUsesExpandedTiers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/prices/price_test" || r.URL.Query().Get("expand[0]") != "tiers" {
			t.Errorf("unexpected Stripe request: %s", r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"price_test","active":true,"currency":"eur","billing_scheme":"tiered","tiers_mode":"volume","recurring":{"interval":"month","usage_type":"licensed"},"tiers":[{"up_to":5,"flat_amount":900},{"up_to":null,"flat_amount":1000}]}`))
	}))
	defer server.Close()

	price, err := testClient(server).GetPrice(t.Context(), "price_test")
	if err != nil {
		t.Fatal(err)
	}
	if got, err := price.MonthlyAmount(6); err != nil || got != 1000 {
		t.Fatalf("six prompts: %d, %v", got, err)
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
			"line_items[0][quantity]": "6",
			"subscription_data[metadata][organization_id]": "org_uuid",
			"subscription_data[metadata][price_version]":   "v1",
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

	session, err := testClient(server).CreateCheckout(t.Context(), "cus_org", "org_uuid", "price_v1", "v1", "reservation_uuid", 6, "https://koi.example", false)
	if err != nil || session.URL == "" {
		t.Fatalf("checkout failed: %+v, %v", session, err)
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
			"phases[1][items][0][quantity]": "6",
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
	if err := testClient(server).ScheduleDowngrade(t.Context(), schedule, "price_v1", 25, 6, 200); err != nil {
		t.Fatal(err)
	}
}
