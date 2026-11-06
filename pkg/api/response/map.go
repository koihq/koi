package response

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/koihq/koi/pkg/domain"
)

func MapError(err error) (int, Envelope) {
	var de *domain.Error
	if errors.As(err, &de) {
		switch de.Code {
		case domain.CodeValidation, domain.CodeInvalidJSON, domain.CodePasswordResetInvalid:
			return http.StatusBadRequest, envelope(de)
		case domain.CodeResourceConflict, domain.CodeEmailTaken, domain.CodeBrandLimitReached, domain.CodePromptLimitReached,
			domain.CodeBillingConflict, domain.CodeBillingCheckoutPending,
			domain.CodeBillingSubscriptionExists, domain.CodeBillingPriceMismatch,
			domain.CodeIdempotencyConflict,
			domain.CodeInitialAlreadyCompleted, domain.CodeRunNotReady:
			return http.StatusConflict, envelope(de)
		case domain.CodeInvalidCredentials, domain.CodeUnauthorized:
			return http.StatusUnauthorized, envelope(de)
		case domain.CodeBillingRequired:
			return http.StatusPaymentRequired, envelope(de)
		case domain.CodeNotFound:
			return http.StatusNotFound, envelope(de)
		case domain.CodeFeatureUnavailable:
			return http.StatusServiceUnavailable, envelope(de)
		}
	}

	return http.StatusInternalServerError, envelope(domain.ErrInternal)
}

func WriteDomainError(w http.ResponseWriter, r *http.Request, err error) {
	status, env := MapError(err)
	if status == http.StatusInternalServerError {
		slog.ErrorContext(r.Context(), "internal error",
			"err", err,
			"request_id", middleware.GetReqID(r.Context()),
		)
	}

	JSON(w, r, status, env)
}
