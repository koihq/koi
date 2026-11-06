package middlewares

import (
	"net/http"

	"github.com/koihq/koi/pkg/api/response"
	"github.com/koihq/koi/pkg/domain"
)

func RequireBilling(access domain.ProductAccessChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := domain.RequireProductAccess(r.Context(), access); err != nil {
				response.WriteDomainError(w, r, err)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
