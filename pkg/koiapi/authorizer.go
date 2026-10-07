package koiapi

import (
	"net/http"

	"github.com/alexedwards/scs/v2"
	"github.com/google/uuid"

	"github.com/koihq/koi/pkg/koiapi/api"
	"github.com/koihq/koi/pkg/koiauth"
	"github.com/koihq/koi/pkg/koidomain"
)

func authenticatedUserID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	identity, ok := koiauth.FromContext(r.Context())
	if !ok {
		api.WriteDomainError(w, r, koidomain.ErrUnauthorized)

		return uuid.Nil, false
	}

	return identity.UserID, true
}

func Authorizer(sess *scs.SessionManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, err := uuid.Parse(sess.GetString(r.Context(), "user_id"))
			if err != nil || id == uuid.Nil {
				api.WriteDomainError(w, r, koidomain.ErrUnauthorized)

				return
			}

			ctx := koiauth.WithIdentity(r.Context(), koiauth.Identity{UserID: id})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
