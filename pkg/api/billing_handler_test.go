package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/domain"
)

const webhookSecret = "whsec_test"

// newBillingServer mounts the billing routes as the server does, so responses go through routing
// and the OpenAPI contract check of the cookie client.
func newBillingServer(t *testing.T, querier domain.BillingQuerier, commander domain.BillingCommander) *httptest.Server {
	t.Helper()

	handler := NewBillingHandler(querier, commander, webhookSecret)
	router := chi.NewRouter()
	router.Route("/api/v1/billing", handler.Routes())
	router.Post("/api/v1/webhooks/stripe", handler.Webhook)

	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	return server
}

func billingRequest(t *testing.T, server *httptest.Server, method, path, body, key string) *http.Response {
	t.Helper()

	request, err := http.NewRequest(method, server.URL+path, strings.NewReader(body))
	require.NoError(t, err)
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if key != "" {
		request.Header.Set("Idempotency-Key", key)
	}

	response, err := newCookieClient(t).Do(request)
	require.NoError(t, err)

	return response
}

func TestBillingReadsCatalogQuoteAndSubscription(t *testing.T) {
	querier := NewMockBillingQuerier(t)
	server := newBillingServer(t, querier, NewMockBillingCommander(t))
	quote := domain.BillingQuote{
		Prompts:     25,
		AmountCents: 4500,
		Currency:    "eur",
		Interval:    "month",
	}
	pending := 15
	periodEnd := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)

	querier.EXPECT().Catalog(mock.Anything).Return(domain.BillingCatalog{
		Currency: "eur",
		Interval: "month",
		Options:  []domain.BillingQuote{quote},
	}, nil)
	querier.EXPECT().Quote(mock.Anything, 25).Return(quote, nil)
	querier.EXPECT().Subscription(mock.Anything).Return(domain.BillingStatus{
		Subscription: &domain.BillingSubscription{
			Quantity:         25,
			Status:           "active",
			CurrentPeriodEnd: &periodEnd,
			PendingQuantity:  &pending,
		},
		PromptLimit: 25,
		UsedPrompts: 3,
	}, nil)

	for path, want := range map[string]string{
		"/api/v1/billing/catalog":          `"options":[{"prompts":25,"amountCents":4500`,
		"/api/v1/billing/quote?prompts=25": `"amountCents":4500`,
		"/api/v1/billing/subscription":     `"pendingQuantity":15`,
	} {
		t.Run(path, func(t *testing.T) {
			response := billingRequest(t, server, http.MethodGet, path, "", "")
			body := readBody(t, response)

			require.Equal(t, http.StatusOK, response.StatusCode, body)
			require.Contains(t, body, want)
		})
	}
}

func TestBillingQuoteRejectsAMissingQuantity(t *testing.T) {
	server := newBillingServer(t, NewMockBillingQuerier(t), NewMockBillingCommander(t))

	response := billingRequest(t, server, http.MethodGet, "/api/v1/billing/quote?prompts=many", "", "")
	body := readBody(t, response)

	require.Equal(t, http.StatusBadRequest, response.StatusCode)
	require.Contains(t, body, `"field":"prompts"`)
}

func TestBillingRedirectsToCheckoutAndPortal(t *testing.T) {
	commander := NewMockBillingCommander(t)
	server := newBillingServer(t, NewMockBillingQuerier(t), commander)

	commander.EXPECT().Checkout(mock.Anything, 25).Return("https://checkout.stripe.com/c/test", nil)
	commander.EXPECT().Portal(mock.Anything).Return("https://billing.stripe.com/p/test", nil)

	checkout := billingRequest(t, server, http.MethodPost, "/api/v1/billing/checkout", `{"prompts":25}`, "")
	require.Equal(t, http.StatusOK, checkout.StatusCode)
	require.Contains(t, readBody(t, checkout), `"url":"https://checkout.stripe.com/c/test"`)

	portal := billingRequest(t, server, http.MethodPost, "/api/v1/billing/portal", "", "")
	require.Equal(t, http.StatusOK, portal.StatusCode)
	require.Contains(t, readBody(t, portal), `"url":"https://billing.stripe.com/p/test"`)
}

func TestBillingPortalWithoutStripeCustomerIsAConflict(t *testing.T) {
	commander := NewMockBillingCommander(t)
	server := newBillingServer(t, NewMockBillingQuerier(t), commander)

	commander.EXPECT().Portal(mock.Anything).Return("", domain.ErrBillingConflict)

	response := billingRequest(t, server, http.MethodPost, "/api/v1/billing/portal", "", "")

	require.Equal(t, http.StatusConflict, response.StatusCode)
	require.Contains(t, readBody(t, response), `"code":"billing_conflict"`)
}

func TestBillingChangeRequiresIdempotencyKeyAndPreservesPaymentResult(t *testing.T) {
	for effective, want := range map[string]int{
		"current":       http.StatusOK,
		"after_payment": http.StatusAccepted,
		"next_cycle":    http.StatusAccepted,
		"trial_end":     http.StatusAccepted,
	} {
		t.Run(effective, func(t *testing.T) {
			commander := NewMockBillingCommander(t)
			server := newBillingServer(t, NewMockBillingQuerier(t), commander)

			missing := billingRequest(t, server, http.MethodPost, "/api/v1/billing/change-allowance", `{"prompts":25}`, "")
			require.Equal(t, http.StatusBadRequest, missing.StatusCode)
			require.NoError(t, missing.Body.Close())

			commander.EXPECT().ChangeAllowance(mock.Anything, 25, "change-key").Return(domain.BillingChange{
				Quantity:   25,
				Effective:  effective,
				PaymentURL: "https://invoice.stripe.com/test",
			}, nil)

			response := billingRequest(t, server, http.MethodPost, "/api/v1/billing/change-allowance", `{"prompts":25}`, "change-key")

			require.Equal(t, want, response.StatusCode)
			require.Contains(t, readBody(t, response), `"paymentUrl":"https://invoice.stripe.com/test"`)
		})
	}
}

func TestBillingStartPlanRequiresIdempotencyKeyAndReportsPayment(t *testing.T) {
	for effective, want := range map[string]int{
		"started":       http.StatusOK,
		"after_payment": http.StatusAccepted,
	} {
		t.Run(effective, func(t *testing.T) {
			commander := NewMockBillingCommander(t)
			server := newBillingServer(t, NewMockBillingQuerier(t), commander)

			missing := billingRequest(t, server, http.MethodPost, "/api/v1/billing/start-plan", "", "")
			require.Equal(t, http.StatusBadRequest, missing.StatusCode)
			require.NoError(t, missing.Body.Close())

			commander.EXPECT().StartPlan(mock.Anything, "start-key").Return(domain.BillingChange{
				Quantity:  25,
				Effective: effective,
			}, nil)

			response := billingRequest(t, server, http.MethodPost, "/api/v1/billing/start-plan", "", "start-key")

			require.Equal(t, want, response.StatusCode)
			require.Contains(t, readBody(t, response), `"effective":"`+effective+`"`)
		})
	}
}

func TestBillingConfirmCheckoutUsesSavedSessionID(t *testing.T) {
	commander := NewMockBillingCommander(t)
	server := newBillingServer(t, NewMockBillingQuerier(t), commander)

	commander.EXPECT().ConfirmCheckout(mock.Anything, "cs_test").Return(domain.BillingStatus{PromptLimit: 5}, nil)

	response := billingRequest(t, server, http.MethodPost, "/api/v1/billing/confirm-checkout", `{"sessionId":"cs_test"}`, "")

	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Contains(t, readBody(t, response), `"promptLimit":5`)
}

func signedWebhook(t *testing.T, server *httptest.Server, payload, secret string) *http.Response {
	t.Helper()

	now := time.Now().Unix()
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = fmt.Fprintf(mac, "%d.%s", now, payload)

	request, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/webhooks/stripe", strings.NewReader(payload))
	require.NoError(t, err)
	request.Header.Set("Stripe-Signature", fmt.Sprintf("t=%d,v1=%s", now, hex.EncodeToString(mac.Sum(nil))))

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)

	return response
}

func TestBillingWebhookProcessesOnlyVerifiedSubscriptionEvents(t *testing.T) {
	subscription := `{"id":"evt_sub","type":"customer.subscription.updated","data":{"object":{"id":"sub_1","customer":"cus_1","metadata":{"organization_id":"org_1"}}}}`
	unrelated := `{"id":"evt_other","type":"invoice.created","data":{"object":{"id":"in_1"}}}`

	t.Run("subscription event", func(t *testing.T) {
		commander := NewMockBillingCommander(t)
		server := newBillingServer(t, NewMockBillingQuerier(t), commander)

		commander.EXPECT().ProcessEvent(mock.Anything, domain.BillingEvent{
			ID:             "evt_sub",
			Type:           "customer.subscription.updated",
			CustomerID:     "cus_1",
			OrganizationID: "org_1",
			SubscriptionID: "sub_1",
		}).Return(nil)

		response := signedWebhook(t, server, subscription, webhookSecret)
		require.NoError(t, response.Body.Close())
		require.Equal(t, http.StatusOK, response.StatusCode)
	})

	t.Run("event without a subscription is acknowledged and ignored", func(t *testing.T) {
		server := newBillingServer(t, NewMockBillingQuerier(t), NewMockBillingCommander(t))

		response := signedWebhook(t, server, unrelated, webhookSecret)
		require.NoError(t, response.Body.Close())
		require.Equal(t, http.StatusOK, response.StatusCode)
	})

	t.Run("wrong signature", func(t *testing.T) {
		server := newBillingServer(t, NewMockBillingQuerier(t), NewMockBillingCommander(t))

		response := signedWebhook(t, server, subscription, "whsec_other")
		require.NoError(t, response.Body.Close())
		require.Equal(t, http.StatusBadRequest, response.StatusCode)
	})

	t.Run("processing failure lets Stripe retry", func(t *testing.T) {
		commander := NewMockBillingCommander(t)
		server := newBillingServer(t, NewMockBillingQuerier(t), commander)

		commander.EXPECT().ProcessEvent(mock.Anything, mock.Anything).Return(errors.New("database down"))

		response := signedWebhook(t, server, subscription, webhookSecret)
		require.NoError(t, response.Body.Close())
		require.Equal(t, http.StatusInternalServerError, response.StatusCode)
	})
}
