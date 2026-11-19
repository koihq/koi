package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const googleTestClientID = "koi-test.apps.googleusercontent.com"

// fakeGoogle is Google's token endpoint. It answers with an ID token built from claims, after
// checking that the exchange carries the client secret and a PKCE verifier.
type fakeGoogle struct {
	*httptest.Server
	claims map[string]any
}

func newFakeGoogle(t *testing.T) *fakeGoogle {
	t.Helper()

	fake := &fakeGoogle{}
	fake.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		require.Equal(t, "/token", r.URL.Path)
		require.Equal(t, "secret", r.PostForm.Get("client_secret"))
		require.NotEmpty(t, r.PostForm.Get("code_verifier"))

		payload, err := json.Marshal(fake.claims)
		require.NoError(t, err)

		idToken := "eyJhbGciOiJSUzI1NiJ9." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"access_token": "access",
			"token_type":   "Bearer",
			"expires_in":   3600,
			"id_token":     idToken,
		}))
	}))
	t.Cleanup(fake.Close)

	return fake
}

func newGoogleServer(t *testing.T) (integrationServer, *fakeGoogle) {
	t.Helper()

	fake := newFakeGoogle(t)
	server := newIntegrationServer(t, func(options *ServerOptions) {
		options.Google = &GoogleAuth{
			ClientID:     googleTestClientID,
			ClientSecret: "secret",
			PublicURL:    "http://localhost:5173",
			AuthURL:      fake.URL + "/auth",
			TokenURL:     fake.URL + "/token",
		}
	})

	return server, fake
}

func noRedirects(client *http.Client) *http.Client {
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

	return client
}

// startGoogle follows the start redirect and returns the query Koi sent to Google.
func startGoogle(t *testing.T, client *http.Client, server integrationServer, returnTo string) url.Values {
	t.Helper()

	resp := doJSON(t, client, http.MethodGet, server.URL+"/api/v1/auth/google/start?returnTo="+url.QueryEscape(returnTo), "")
	readBody(t, resp)
	require.Equal(t, http.StatusFound, resp.StatusCode)

	location, err := url.Parse(resp.Header.Get("Location"))
	require.NoError(t, err)
	require.Equal(t, "/auth", location.Path)

	return location.Query()
}

// finishGoogle calls the callback and returns where Koi sent the browser.
func finishGoogle(t *testing.T, client *http.Client, server integrationServer, query url.Values) string {
	t.Helper()

	resp := doJSON(t, client, http.MethodGet, server.URL+"/api/v1/auth/google/callback?"+query.Encode(), "")
	readBody(t, resp)
	require.Equal(t, http.StatusFound, resp.StatusCode)

	return resp.Header.Get("Location")
}

func verifiedClaims(email, nonce string) map[string]any {
	return map[string]any{
		"iss":            "https://accounts.google.com",
		"aud":            googleTestClientID,
		"exp":            time.Now().Add(time.Hour).Unix(),
		"nonce":          nonce,
		"email":          email,
		"email_verified": true,
	}
}

func TestGoogleSignInCreatesAnAccountAndReturnsToThePage(t *testing.T) {
	server, fake := newGoogleServer(t)
	client := noRedirects(newCookieClient(t))

	resp := doJSON(t, client, http.MethodGet, server.URL+"/api/v1/auth/providers", "")
	require.JSONEq(t, `{"google":true}`, readBody(t, resp))

	sent := startGoogle(t, client, server, "/oauth/consent")
	require.Equal(t, googleTestClientID, sent.Get("client_id"))
	require.Equal(t, "http://localhost:5173/api/v1/auth/google/callback", sent.Get("redirect_uri"))
	require.Equal(t, "S256", sent.Get("code_challenge_method"))
	require.Contains(t, sent.Get("scope"), "openid")

	email := uuid.NewString() + "@example.com"
	fake.claims = verifiedClaims(email, sent.Get("nonce"))

	location := finishGoogle(t, client, server, url.Values{
		"code":  {"code"},
		"state": {sent.Get("state")},
	})
	require.Equal(t, "/oauth/consent", location)

	resp = doJSON(t, client, http.MethodGet, server.URL+"/api/v1/auth/me", "")
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var me MeRes
	readJSON(t, resp, &me)
	require.Equal(t, email, me.Email)

	// The state was single use: replaying the callback does not sign anyone in again.
	replay := finishGoogle(t, noRedirects(newCookieClient(t)), server, url.Values{
		"code":  {"code"},
		"state": {sent.Get("state")},
	})
	require.Equal(t, "/sign-in?error=google_failed", replay)
}

func TestGoogleSignInJoinsTheAccountWithTheSameEmail(t *testing.T) {
	server, fake := newGoogleServer(t)
	_, account := server.signup(t)
	client := noRedirects(newCookieClient(t))

	sent := startGoogle(t, client, server, "//evil.example/steal")
	fake.claims = verifiedClaims(strings.ToUpper(account.Email), sent.Get("nonce"))

	location := finishGoogle(t, client, server, url.Values{
		"code":  {"code"},
		"state": {sent.Get("state")},
	})
	require.Equal(t, "/", location, "returnTo never leaves the origin")

	resp := doJSON(t, client, http.MethodGet, server.URL+"/api/v1/auth/me", "")

	var me MeRes
	readJSON(t, resp, &me)
	require.Equal(t, account.ID, me.ID)
	require.Equal(t, account.Organization.ID, me.Organization.ID)
}

func TestGoogleSignInFailuresReturnToSignIn(t *testing.T) {
	for name, tc := range map[string]struct {
		claims func(nonce string) map[string]any
		query  func(state string) url.Values
		want   string
	}{
		"cancelled on Google": {
			query: func(state string) url.Values {
				return url.Values{"error": {"access_denied"}, "state": {state}}
			},
			want: "google_cancelled",
		},
		"state from another session": {
			query: func(string) url.Values {
				return url.Values{"code": {"code"}, "state": {"forged"}}
			},
			want: "google_failed",
		},
		"email not verified": {
			claims: func(nonce string) map[string]any {
				claims := verifiedClaims("unverified@example.com", nonce)
				claims["email_verified"] = false

				return claims
			},
			want: "google_unverified",
		},
		"token for another client": {
			claims: func(nonce string) map[string]any {
				claims := verifiedClaims("other@example.com", nonce)
				claims["aud"] = "someone-else"

				return claims
			},
			want: "google_failed",
		},
		"token from another request": {
			claims: func(string) map[string]any {
				return verifiedClaims("replayed@example.com", "old-nonce")
			},
			want: "google_failed",
		},
	} {
		t.Run(name, func(t *testing.T) {
			server, fake := newGoogleServer(t)
			client := noRedirects(newCookieClient(t))
			sent := startGoogle(t, client, server, "/")

			if tc.claims != nil {
				fake.claims = tc.claims(sent.Get("nonce"))
			}

			query := url.Values{"code": {"code"}, "state": {sent.Get("state")}}
			if tc.query != nil {
				query = tc.query(sent.Get("state"))
			}

			location := finishGoogle(t, client, server, query)
			require.Equal(t, "/sign-in?error="+tc.want, location)

			resp := doJSON(t, client, http.MethodGet, server.URL+"/api/v1/auth/me", "")
			readBody(t, resp)
			require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		})
	}
}

func TestGoogleSignInIsOffWithoutAClient(t *testing.T) {
	server := newIntegrationServer(t)
	client := noRedirects(newCookieClient(t))

	resp := doJSON(t, client, http.MethodGet, server.URL+"/api/v1/auth/providers", "")
	require.JSONEq(t, `{"google":false}`, readBody(t, resp))

	for _, path := range []string{"/api/v1/auth/google/start", "/api/v1/auth/google/callback"} {
		resp := doJSON(t, client, http.MethodGet, server.URL+path, "")
		readBody(t, resp)
		require.Equal(t, http.StatusNotFound, resp.StatusCode, path)
	}
}
