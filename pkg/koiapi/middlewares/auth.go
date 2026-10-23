package middlewares

import (
	"context"
	"net/http"

	"github.com/alexedwards/scs/v2"
	"github.com/google/uuid"

	apiresponse "github.com/koihq/koi/pkg/koiapi/response"
	"github.com/koihq/koi/pkg/koiauth"
	"github.com/koihq/koi/pkg/koidomain"
)

type IdentityResolver func(context.Context, uuid.UUID) (koiauth.Identity, error)

func Authenticate(sess *scs.SessionManager, resolve IdentityResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, err := uuid.Parse(sess.GetString(r.Context(), "user_id"))
			if err != nil || id == uuid.Nil {
				apiresponse.WriteDomainError(w, r, koidomain.ErrUnauthorized)
				return
			}

			identity, err := resolve(r.Context(), id)
			if err != nil {
				apiresponse.WriteDomainError(w, r, err)
				return
			}

			next.ServeHTTP(w, r.WithContext(koiauth.WithIdentity(r.Context(), identity)))
		})
	}
}
