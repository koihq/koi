package koiapi

import (
	"context"
	"net/http"

	"github.com/alexedwards/scs/v2"
	"github.com/google/uuid"

	"github.com/koihq/koi/pkg/koiapi/api"
	"github.com/koihq/koi/pkg/koidomain"
)

type userIDContextKey struct{}

func authenticatedUserID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, ok := r.Context().Value(userIDContextKey{}).(uuid.UUID)
	if !ok || id == uuid.Nil {
		api.WriteDomainError(w, r, koidomain.ErrUnauthorized)
		return uuid.Nil, false
	}
	return id, true
}

func Authorizer(sess *scs.SessionManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, err := uuid.Parse(sess.GetString(r.Context(), "user_id"))
			if err != nil || id == uuid.Nil {
				api.WriteDomainError(w, r, koidomain.ErrUnauthorized)
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userIDContextKey{}, id)))
		})
	}
}
