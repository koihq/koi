package middlewares

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/httprate"

	"github.com/koihq/koi/pkg/api/response"
	"github.com/koihq/koi/pkg/domain"
)

const (
	// Each password hash holds 19 MiB, and the garbage collector frees it late. A load test reached
	// 447 MB with 8 at once, too close to the container's 512 MiB limit.
	maxConcurrentHashes = 4
	hashSlotWait        = 5 * time.Second
)

// RateLimiter counts in memory, which suits a single server; counts restart with each deploy.
// A nil RateLimiter allows everything, for handlers tested without one.
type RateLimiter struct {
	clientIPHeader string
	hashSlots      chan struct{}
	hashWait       time.Duration
}

// NewRateLimiter trusts clientIPHeader for the client's address. Set it only behind a proxy that
// overwrites the header; otherwise anyone could send a new value to escape the limit.
func NewRateLimiter(clientIPHeader string) *RateLimiter {
	return &RateLimiter{
		clientIPHeader: clientIPHeader,
		hashSlots:      make(chan struct{}, maxConcurrentHashes),
		hashWait:       hashSlotWait,
	}
}

func (l *RateLimiter) clientIP(r *http.Request) (string, error) {
	if l.clientIPHeader != "" {
		if ip := strings.TrimSpace(r.Header.Get(l.clientIPHeader)); ip != "" {
			return ip, nil
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr, nil
	}

	return host, nil
}

// PerIP limits one route. Every call keeps its own count.
func (l *RateLimiter) PerIP(requests int, window time.Duration) func(http.Handler) http.Handler {
	if l == nil {
		return passThrough
	}

	return httprate.LimitBy(requests, window, l.clientIP, httprate.WithLimitHandler(writeRateLimited))
}

// PerBodyEmail limits one route by address. It runs after DecodeBody, so the key is the normalized
// email. The limit passes with time and never locks the account.
func PerBodyEmail[T any](l *RateLimiter, requests int, window time.Duration, email func(T) string) func(http.Handler) http.Handler {
	if l == nil {
		return passThrough
	}

	limiter := httprate.NewRateLimiter(requests, window, httprate.WithLimitHandler(writeRateLimited))

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := strings.ToLower(strings.TrimSpace(email(Body[T](r))))
			if key != "" && limiter.RespondOnLimit(w, r, key) {
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// HashSlot limits password hashing across routes, so a burst cannot exhaust the server's memory.
func (l *RateLimiter) HashSlot(next http.Handler) http.Handler {
	if l == nil {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wait := time.NewTimer(l.hashWait)
		defer wait.Stop()

		select {
		case l.hashSlots <- struct{}{}:
			defer func() { <-l.hashSlots }()

			next.ServeHTTP(w, r)
		case <-wait.C:
			w.Header().Set("Retry-After", strconv.Itoa(int(hashSlotWait.Seconds())))
			writeRateLimited(w, r)
		case <-r.Context().Done():
		}
	})
}

func writeRateLimited(w http.ResponseWriter, r *http.Request) {
	response.WriteDomainError(w, r, domain.ErrRateLimited)
}

func passThrough(next http.Handler) http.Handler {
	return next
}
