package koiapi

import (
	"net/http"

	"github.com/alexedwards/scs/v2"

	"github.com/koihq/koi/pkg/koiapi/api"
)

func Authorizer(sess *scs.SessionManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if sess.GetString(r.Context(), "user_id") == "" {
				api.Error(w, r, http.StatusUnauthorized, "unauthorized")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
