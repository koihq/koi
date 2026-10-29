package koiapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/koiapi/middlewares"
	"github.com/koihq/koi/pkg/koidomain"
)

func TestBillingChangeRequiresIdempotencyKeyAndPreservesPaymentResult(t *testing.T) {
	for _, effective := range []string{"current", "after_payment", "next_cycle", "trial_end"} {
		t.Run(effective, func(t *testing.T) {
			commander := NewMockBillingCommander(t)
			handler := NewBillingHandler(NewMockBillingQuerier(t), commander, "secret")
			router := chi.NewRouter()
			router.With(middlewares.DecodeBody[allowanceRequest]()).Post("/change", handler.ChangeAllowance)

			missing := httptest.NewRecorder()
			router.ServeHTTP(missing, httptest.NewRequest(http.MethodPost, "/change", strings.NewReader(`{"prompts":25}`)))
			require.Equal(t, http.StatusBadRequest, missing.Code)

			commander.EXPECT().ChangeAllowance(mock.Anything, 25, "change-key").Return(koidomain.BillingChange{Quantity: 25, Effective: effective, PaymentURL: "https://invoice.stripe.com/test"}, nil)
			request := httptest.NewRequest(http.MethodPost, "/change", strings.NewReader(`{"prompts":25}`))
			request.Header.Set("Idempotency-Key", "change-key")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			want := http.StatusAccepted
			if effective == "current" {
				want = http.StatusOK
			}
			require.Equal(t, want, response.Code)
			require.Contains(t, response.Body.String(), `"paymentUrl":"https://invoice.stripe.com/test"`)
		})
	}
}

func TestBillingConfirmCheckoutUsesSavedSessionID(t *testing.T) {
	commander := NewMockBillingCommander(t)
	handler := NewBillingHandler(NewMockBillingQuerier(t), commander, "secret")
	router := chi.NewRouter()
	router.With(middlewares.DecodeBody[confirmCheckoutRequest]()).Post("/confirm", handler.ConfirmCheckout)
	commander.EXPECT().ConfirmCheckout(mock.Anything, "cs_test").Return(koidomain.BillingStatus{PromptLimit: 5}, nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/confirm", strings.NewReader(`{"sessionId":"cs_test"}`)))
	require.Equal(t, http.StatusOK, response.Code)
	require.Contains(t, response.Body.String(), `"promptLimit":5`)
}
