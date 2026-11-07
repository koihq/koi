package api

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/api/response"
	"github.com/koihq/koi/pkg/domain"
)

func securityServer(t *testing.T, options ServerOptions) http.Handler {
	t.Helper()

	store := NewMockStore(t)
	store.EXPECT().CredentialRepo().Return(nil)

	return NewServer(store, scs.New(), options)
}

func post(handler http.Handler, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	for key, value := range headers {
		request.Header.Set(key, value)
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	return recorder
}

func errorCode(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()

	var body response.Envelope
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))

	return body.Error.Code
}

func TestRequestBodyOverOneMebibyteIsRejected(t *testing.T) {
	handler := securityServer(t, ServerOptions{})
	body := `{"email":"` + strings.Repeat("a", 2<<20) + `"}`

	recorder := post(handler, "/api/v1/auth/login", body, nil)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Equal(t, domain.CodeInvalidJSON, errorCode(t, recorder))
}

func TestCrossSiteWritesAreRejected(t *testing.T) {
	handler := securityServer(t, ServerOptions{})
	crossSite := map[string]string{"Sec-Fetch-Site": "cross-site"}

	for _, path := range []string{"/api/v1/auth/login", "/api/v1/onboarding/events", "/api/v1/oauth/consent"} {
		recorder := post(handler, path, `{}`, crossSite)

		require.Equal(t, http.StatusForbidden, recorder.Code, path)
		require.Equal(t, domain.CodeCrossOriginRequest, errorCode(t, recorder), path)
	}

	// These do not use the session cookie, so they are not rejected for being cross-site.
	for _, path := range []string{"/mcp", "/oauth/token", "/oauth/register", "/oauth/revoke", "/api/v1/webhooks/stripe"} {
		require.NotEqual(t, http.StatusForbidden, post(handler, path, `{}`, crossSite).Code, path)
	}

	require.Equal(t, http.StatusBadRequest, post(handler, "/api/v1/auth/login", `broken`, map[string]string{
		"Sec-Fetch-Site": "same-origin",
	}).Code)
}

func TestSecurityHeaders(t *testing.T) {
	read := func(handler http.Handler, path string) http.Header {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))

		return recorder.Header()
	}

	local := read(securityServer(t, ServerOptions{}), "/prompts")
	require.Equal(t, "nosniff", local.Get("X-Content-Type-Options"))
	require.Equal(t, "strict-origin-when-cross-origin", local.Get("Referrer-Policy"))
	require.Equal(t, "DENY", local.Get("X-Frame-Options"))
	require.Contains(t, local.Get("Content-Security-Policy"), "frame-ancestors 'none'")
	require.Contains(t, local.Get("Content-Security-Policy"), "https://img.logo.dev")
	require.Contains(t, local.Get("Content-Security-Policy"), "font-src 'self' data:", "Vite inlines small fonts")
	require.Empty(t, local.Get("Strict-Transport-Security"))

	public := read(securityServer(t, ServerOptions{PublicURL: "https://app.example"}), "/prompts")
	require.NotEmpty(t, public.Get("Strict-Transport-Security"))
}

func TestDevOriginOnlyOnLocalInstances(t *testing.T) {
	preflight := func(handler http.Handler) string {
		request := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
		request.Header.Set("Origin", devUIOrigin)
		request.Header.Set("Access-Control-Request-Method", http.MethodPost)

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)

		return recorder.Header().Get("Access-Control-Allow-Origin")
	}

	require.Equal(t, devUIOrigin, preflight(securityServer(t, ServerOptions{})))
	require.Equal(t, devUIOrigin, preflight(securityServer(t, ServerOptions{PublicURL: "http://localhost:8080"})))
	require.Empty(t, preflight(securityServer(t, ServerOptions{PublicURL: "https://app.example"})))
}

func TestRateLimitPerIP(t *testing.T) {
	handler := securityServer(t, ServerOptions{})

	for range 20 {
		require.Equal(t, http.StatusBadRequest, post(handler, "/api/v1/auth/login", `broken`, nil).Code)
	}

	limited := post(handler, "/api/v1/auth/login", `broken`, nil)
	require.Equal(t, http.StatusTooManyRequests, limited.Code)
	require.Equal(t, domain.CodeRateLimited, errorCode(t, limited))
	require.NotEmpty(t, limited.Header().Get("Retry-After"))

	// Each route keeps its own count.
	require.Equal(t, http.StatusBadRequest, post(handler, "/api/v1/auth/signup", `broken`, nil).Code)
}

func TestClientIPHeaderIsTrustedOnlyWhenConfigured(t *testing.T) {
	exhaust := func(handler http.Handler) {
		for range 20 {
			post(handler, "/api/v1/auth/login", `broken`, map[string]string{"CF-Connecting-IP": "203.0.113.1"})
		}
	}

	otherClient := map[string]string{"CF-Connecting-IP": "203.0.113.2"}

	configured := securityServer(t, ServerOptions{ClientIPHeader: "CF-Connecting-IP"})
	exhaust(configured)
	require.Equal(t, http.StatusBadRequest, post(configured, "/api/v1/auth/login", `broken`, otherClient).Code)

	unconfigured := securityServer(t, ServerOptions{})
	exhaust(unconfigured)
	require.Equal(t, http.StatusTooManyRequests, post(unconfigured, "/api/v1/auth/login", `broken`, otherClient).Code)
}

func TestRateLimitPerEmail(t *testing.T) {
	store := NewMockStore(t)
	store.EXPECT().CredentialRepo().Return(nil)
	store.EXPECT().EnqueuePasswordResetEmail(mock.Anything, mock.Anything).Return(nil)

	handler := NewServer(store, scs.New(), ServerOptions{})
	forgot := func(email string) *httptest.ResponseRecorder {
		return post(handler, "/api/v1/auth/password/forgot", `{"email":"`+email+`"}`, nil)
	}

	for range 5 {
		require.Equal(t, http.StatusAccepted, forgot("you@example.com").Code)
	}

	// Case and spacing do not make a new address.
	limited := forgot(" YOU@example.com ")
	require.Equal(t, http.StatusTooManyRequests, limited.Code)
	require.Equal(t, domain.CodeRateLimited, errorCode(t, limited))

	require.Equal(t, http.StatusAccepted, forgot("other@example.com").Code)
}

func TestBrandLookupIsRateLimited(t *testing.T) {
	sess := scs.New()
	sess.Cookie.Name = "koi_session"

	app := NewServer(sessionStore(t), sess, ServerOptions{})
	testSession := sess.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess.Put(r.Context(), "user_id", uuid.NewString())
		w.WriteHeader(http.StatusNoContent)
	}))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/test-session" {
			testSession.ServeHTTP(w, r)

			return
		}

		app.ServeHTTP(w, r)
	}))
	defer ts.Close()

	client := newCookieClient(t)

	session := doJSON(t, client, http.MethodGet, ts.URL+"/test-session", "")
	require.NoError(t, session.Body.Close())

	lookup := ts.URL + "/api/v1/brands/lookup?domain=acme.com"

	for range 30 {
		res := doJSON(t, client, http.MethodGet, lookup, "")
		require.Equal(t, http.StatusOK, res.StatusCode)
		require.NoError(t, res.Body.Close())
	}

	limited := doJSON(t, client, http.MethodGet, lookup, "")
	defer limited.Body.Close()
	require.Equal(t, http.StatusTooManyRequests, limited.StatusCode)
}

func TestPublicIPRejectsCarrierGradeNAT(t *testing.T) {
	require.False(t, publicIP(net.ParseIP("100.64.0.1")))
	require.False(t, publicIP(net.ParseIP("100.127.255.254")))
	require.True(t, publicIP(net.ParseIP("100.63.255.255")))
	require.True(t, publicIP(net.ParseIP("100.128.0.1")))
}
