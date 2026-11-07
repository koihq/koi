package middlewares

import (
	"net/http"
	"strings"

	"github.com/koihq/koi/pkg/api/response"
	"github.com/koihq/koi/pkg/domain"
)

const contentSecurityPolicy = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data: https://img.logo.dev; font-src 'self' data:; connect-src 'self'; " +
	"frame-ancestors 'none'; base-uri 'self'"

// SecurityHeaders applies to every response. HSTS needs an HTTPS public address, because a browser
// would otherwise refuse the plain HTTP that a local instance serves.
func SecurityHeaders(publicURL string) func(http.Handler) http.Handler {
	https := strings.HasPrefix(publicURL, "https://")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			headers := w.Header()
			headers.Set("X-Content-Type-Options", "nosniff")
			headers.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			headers.Set("X-Frame-Options", "DENY")

			if https {
				headers.Set("Strict-Transport-Security", "max-age=31536000")
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ContentSecurityPolicy applies to the embedded app. Vite's dev server does not pass through it,
// so a violation only shows on a production build.
func ContentSecurityPolicy(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", contentSecurityPolicy)

		next.ServeHTTP(w, r)
	})
}

// CrossOriginProtection rejects cross-site writes from browsers. SameSite=Lax alone would still
// trust sibling subdomains. Routes authenticated by a bearer token or signature are exempt.
func CrossOriginProtection(trustedOrigins []string, exempt []string) func(http.Handler) http.Handler {
	protection := http.NewCrossOriginProtection()

	for _, origin := range trustedOrigins {
		if err := protection.AddTrustedOrigin(origin); err != nil {
			panic(err)
		}
	}

	for _, pattern := range exempt {
		protection.AddInsecureBypassPattern(pattern)
	}

	protection.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response.WriteDomainError(w, r, domain.ErrCrossOriginRequest)
	}))

	return protection.Handler
}
