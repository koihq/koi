package middlewares

import (
	"context"
	"net/http"

	"github.com/alexedwards/scs/v2"
	"github.com/google/uuid"

	"github.com/koihq/koi/pkg/api/response"
	"github.com/koihq/koi/pkg/auth"
	"github.com/koihq/koi/pkg/domain"
)

type IdentityResolver func(context.Context, uuid.UUID) (auth.Identity, error)

func Authenticate(sess *scs.SessionManager, resolve IdentityResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, err := uuid.Parse(sess.GetString(r.Context(), "user_id"))
			if err != nil || id == uuid.Nil {
				response.WriteDomainError(w, r, domain.ErrUnauthorized)
				return
			}

			identity, err := resolve(r.Context(), id)
			if err != nil {
				response.WriteDomainError(w, r, err)
				return
			}

			next.ServeHTTP(w, r.WithContext(auth.WithIdentity(r.Context(), identity)))
		})
	}
}
