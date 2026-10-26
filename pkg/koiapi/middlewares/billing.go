package middlewares

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	apiresponse "github.com/koihq/koi/pkg/koiapi/response"
	"github.com/koihq/koi/pkg/koiauth"
	"github.com/koihq/koi/pkg/koidomain"
)

type ProductAccess interface {
	ProductAccess(context.Context, uuid.UUID) (bool, error)
}

func RequireBilling(access ProductAccess) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identity, ok := koiauth.FromContext(r.Context())
			if !ok || identity.OrganizationID == uuid.Nil {
				apiresponse.WriteDomainError(w, r, koidomain.ErrUnauthorized)
				return
			}

			allowed, err := access.ProductAccess(r.Context(), identity.OrganizationID)
			if err != nil {
				apiresponse.WriteDomainError(w, r, err)
				return
			}
			if !allowed {
				apiresponse.WriteDomainError(w, r, koidomain.ErrBillingRequired)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
