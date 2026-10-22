package koiapi

import (
	"context"
	"net/http"

	"github.com/alexedwards/scs/v2"
	"github.com/google/uuid"

	"github.com/koihq/koi/pkg/koiapi/api"
	"github.com/koihq/koi/pkg/koidomain"
)

type ProductAccess interface {
	ProductAccess(context.Context, uuid.UUID) (bool, error)
}

func RequireBilling(sess *scs.SessionManager, access ProductAccess) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, err := uuid.Parse(sess.GetString(r.Context(), "user_id"))
			if err != nil || userID == uuid.Nil {
				api.WriteDomainError(w, r, koidomain.ErrUnauthorized)
				return
			}

			allowed, err := access.ProductAccess(r.Context(), userID)
			if err != nil {
				api.WriteDomainError(w, r, err)
				return
			}
			if !allowed {
				api.WriteDomainError(w, r, koidomain.ErrBillingRequired)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
