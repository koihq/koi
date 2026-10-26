package response

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/koihq/koi/pkg/koidomain"
)

func MapError(err error) (int, Envelope) {
	var de *koidomain.Error
	if errors.As(err, &de) {
		switch de.Code {
		case koidomain.CodeValidation, koidomain.CodeInvalidJSON:
			return http.StatusBadRequest, envelope(de)
		case koidomain.CodeResourceConflict, koidomain.CodeEmailTaken, koidomain.CodeBrandLimitReached, koidomain.CodePromptLimitReached,
			koidomain.CodeBillingConflict, koidomain.CodeBillingCheckoutPending,
			koidomain.CodeBillingSubscriptionExists, koidomain.CodeBillingPriceMismatch,
			koidomain.CodeGenerationConflict, koidomain.CodeIdempotencyConflict,
			koidomain.CodeInitialAlreadyCompleted, koidomain.CodeRunNotReady:
			return http.StatusConflict, envelope(de)
		case koidomain.CodeInvalidCredentials, koidomain.CodeUnauthorized:
			return http.StatusUnauthorized, envelope(de)
		case koidomain.CodeBillingRequired:
			return http.StatusPaymentRequired, envelope(de)
		case koidomain.CodeNotFound:
			return http.StatusNotFound, envelope(de)
		case koidomain.CodeFeatureUnavailable:
			return http.StatusServiceUnavailable, envelope(de)
		}
	}

	return http.StatusInternalServerError, envelope(koidomain.ErrInternal)
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
