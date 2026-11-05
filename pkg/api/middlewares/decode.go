package middlewares

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/koihq/koi/pkg/api/response"
	"github.com/koihq/koi/pkg/domain"
)

type bodyKey struct{}

func DecodeBody[T any]() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var v T
			if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
				response.WriteDomainError(w, r, domain.ErrInvalidJSON)

				return
			}

			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), bodyKey{}, v)))
		})
	}
}

func Body[T any](r *http.Request) T {
	v, _ := r.Context().Value(bodyKey{}).(T)

	return v
}
