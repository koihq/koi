package middlewares

import (
	"context"
	"encoding/json"
	"net/http"

	apiresponse "github.com/koihq/koi/pkg/koiapi/response"
	"github.com/koihq/koi/pkg/koidomain"
)

type bodyKey struct{}

func DecodeBody[T any]() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var v T
			if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
				apiresponse.WriteDomainError(w, r, koidomain.ErrInvalidJSON)

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
