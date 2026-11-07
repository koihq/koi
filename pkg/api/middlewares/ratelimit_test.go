package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHashSlotRejectsWhenAllSlotsAreBusy(t *testing.T) {
	limiter := NewRateLimiter("")
	limiter.hashSlots = make(chan struct{}, 1)
	limiter.hashWait = 10 * time.Millisecond

	release := make(chan struct{})
	entered := make(chan struct{})

	handler := limiter.HashSlot(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(entered)
		<-release
		w.WriteHeader(http.StatusNoContent)
	}))

	first := httptest.NewRecorder()
	done := make(chan struct{})

	go func() {
		handler.ServeHTTP(first, httptest.NewRequest(http.MethodPost, "/", nil))
		close(done)
	}()
	<-entered

	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest(http.MethodPost, "/", nil))
	require.Equal(t, http.StatusTooManyRequests, second.Code)
	require.NotEmpty(t, second.Header().Get("Retry-After"))

	close(release)
	<-done
	require.Equal(t, http.StatusNoContent, first.Code)
}

func TestNilRateLimiterAllowsEverything(t *testing.T) {
	var limiter *RateLimiter

	handler := limiter.PerIP(1, time.Hour)(limiter.HashSlot(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})))

	for range 3 {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/", nil))
		require.Equal(t, http.StatusNoContent, response.Code)
	}
}
