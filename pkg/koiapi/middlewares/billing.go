package middlewares

import (
	"net/http"

	apiresponse "github.com/koihq/koi/pkg/koiapi/response"
	"github.com/koihq/koi/pkg/koidomain"
)

func RequireBilling(access koidomain.ProductAccessChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := koidomain.RequireProductAccess(r.Context(), access); err != nil {
				apiresponse.WriteDomainError(w, r, err)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
