package api

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/google/uuid"
	"github.com/koihq/koi/pkg/domain"
	"github.com/stretchr/testify/require"
)

func TestOAuthOriginSeparatesMCPFromBrowserInDev(t *testing.T) {
	t.Setenv("KOI_PUBLIC_URL", "http://localhost:5173")
	t.Setenv("KOI_MCP_PUBLIC_URL", "http://localhost:8080")
	require.Equal(t, "http://localhost:8080", oauthOrigin())

	t.Setenv("KOI_MCP_PUBLIC_URL", "")
	require.Equal(t, "http://localhost:5173", oauthOrigin())
}

func TestOAuthConsentUsesTheSessionHostname(t *testing.T) {
	t.Setenv("KOI_PUBLIC_URL", "http://localhost:5173")
	require.Equal(t, "http://localhost:5173", oauthUIOrigin("http://localhost:8080"))
	t.Setenv("KOI_PUBLIC_URL", "https://app.example.com")
	require.Equal(t, "https://mcp.example.com", oauthUIOrigin("https://mcp.example.com"))
}

func TestPublicMCPURLRequiresConfigurationForRemoteRequests(t *testing.T) {
	t.Setenv("KOI_PUBLIC_URL", "")
	t.Setenv("KOI_MCP_PUBLIC_URL", "")
	local := httptest.NewRequest(http.MethodGet, "http://localhost:8080/api/v1/capabilities", nil)
	remote := httptest.NewRequest(http.MethodGet, "https://koi.example.com/api/v1/capabilities", nil)
	require.Equal(t, "http://localhost:8080/mcp", publicMCPURL(local))
	require.Empty(t, publicMCPURL(remote))

	recorder := httptest.NewRecorder()
	requireConfiguredRemoteMCP(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(recorder, remote)
	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)

	t.Setenv("KOI_MCP_PUBLIC_URL", "https://mcp.example.com")
	require.Equal(t, "https://mcp.example.com/mcp", publicMCPURL(remote))
}

func TestPKCEVerifierAlphabet(t *testing.T) {
	require.True(t, validVerifier(strings.Repeat("v", 41)+".~"))
	require.False(t, validChallenge(strings.Repeat("v", 41)+".~"))
	require.False(t, validVerifier(strings.Repeat("v", 42)+"!"))
}

func TestOAuthLoopbackRedirectAllowsOnlyPortChange(t *testing.T) {
	registered := []string{"http://127.0.0.1/callback/client", "https://app.example/callback"}

	require.True(t, redirectAllowed(registered, "http://127.0.0.1:53102/callback/client"))
	require.True(t, redirectAllowed(registered, "https://app.example/callback"))
	require.False(t, redirectAllowed(registered, "http://localhost:53102/callback/client"))
	require.False(t, redirectAllowed(registered, "http://127.0.0.1:53102/callback/other"))
	require.False(t, redirectAllowed(registered, "http://127.0.0.1:53102/callback/client?next=evil"))
	require.False(t, redirectAllowed(registered, "https://app.example:53102/callback"))
}

type memoryOAuthStore struct {
	mu          sync.Mutex
	clients     map[string]*domain.OAuthClient
	codes       map[string]*domain.OAuthCode
	credentials map[string]*domain.Credential
}

func newMemoryOAuthStore() *memoryOAuthStore {
	return &memoryOAuthStore{clients: map[string]*domain.OAuthClient{}, codes: map[string]*domain.OAuthCode{}, credentials: map[string]*domain.Credential{}}
}
func (s *memoryOAuthStore) CreateOAuthClient(_ context.Context, c *domain.OAuthClient) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[c.ID] = c
	return nil
}
func (s *memoryOAuthStore) OAuthClient(_ context.Context, id string) (*domain.OAuthClient, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.clients[id], nil
}
func (s *memoryOAuthStore) CreateOAuthCode(_ context.Context, c *domain.OAuthCode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.codes[c.Hash] = c
	return nil
}
func (s *memoryOAuthStore) ConsumeOAuthCode(_ context.Context, raw string) (*domain.OAuthCode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := domain.HashCredential(raw)
	c := s.codes[key]
	delete(s.codes, key)
	return c, nil
}
func (s *memoryOAuthStore) CreateCredential(_ context.Context, c *domain.Credential) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.credentials[c.TokenHash] = c
	return nil
}
func (s *memoryOAuthStore) CredentialByToken(_ context.Context, raw string) (*domain.Credential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.credentials[domain.HashCredential(raw)], nil
}
func (s *memoryOAuthStore) ConsumeRefreshToken(_ context.Context, raw string) (*domain.Credential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := domain.HashCredential(raw)
	c := s.credentials[key]
	delete(s.credentials, key)
	return c, nil
}
func (s *memoryOAuthStore) RevokeToken(_ context.Context, raw string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.credentials, domain.HashCredential(raw))
	return nil
}
func (s *memoryOAuthStore) ListCredentials(context.Context, uuid.UUID, domain.PageParams) (*domain.Page[domain.Credential], error) {
	return nil, nil
}
func (s *memoryOAuthStore) RevokeCredential(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return false, nil
}

func (s *memoryOAuthStore) List(context.Context, domain.PageParams) (*domain.Page[domain.Credential], error) {
	return nil, nil
}
func (s *memoryOAuthStore) Revoke(context.Context, uuid.UUID) error { return nil }
func (s *memoryOAuthStore) Resolve(ctx context.Context, raw string) (*domain.Credential, error) {
	return s.CredentialByToken(ctx, raw)
}
func (s *memoryOAuthStore) RegisterClient(ctx context.Context, c *domain.OAuthClient) error {
	return s.CreateOAuthClient(ctx, c)
}
func (s *memoryOAuthStore) Client(ctx context.Context, id string) (*domain.OAuthClient, error) {
	return s.OAuthClient(ctx, id)
}
func (s *memoryOAuthStore) CreateCode(ctx context.Context, c *domain.OAuthCode) error {
	return s.CreateOAuthCode(ctx, c)
}
func (s *memoryOAuthStore) ConsumeCode(ctx context.Context, raw string) (*domain.OAuthCode, error) {
	return s.ConsumeOAuthCode(ctx, raw)
}
func (s *memoryOAuthStore) IssueTokens(ctx context.Context, userID uuid.UUID, clientID, name, scope, audience string, connectionID uuid.UUID) (domain.IssuedTokens, error) {
	access, _ := domain.RandomCredential()
	refresh, _ := domain.RandomCredential()
	s.CreateCredential(ctx, &domain.Credential{UserID: userID, ClientID: clientID, Audience: audience, Scope: scope, Kind: "oauth_access", TokenHash: domain.HashCredential(access)})
	s.CreateCredential(ctx, &domain.Credential{UserID: userID, ClientID: clientID, Audience: audience, Scope: scope, Kind: "oauth_refresh", TokenHash: domain.HashCredential(refresh)})
	return domain.IssuedTokens{AccessToken: access, RefreshToken: refresh, Scope: scope, ExpiresIn: 3600}, nil
}
func (s *memoryOAuthStore) RefreshTokens(context.Context, string, string, string) (domain.IssuedTokens, error) {
	return domain.IssuedTokens{}, nil
}

func TestOAuthAuthorizationCodePKCEAndRevocation(t *testing.T) {
	store := newMemoryOAuthStore()
	sess := scs.New()
	h := &oauthHandler{store: store, sess: sess, origin: "http://localhost:8080"}
	store.clients["client-1"] = &domain.OAuthClient{ID: "client-1", Name: "Test client", RedirectURIs: []string{"https://client.example/cb"}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /oauth/authorize", h.authorize)
	mux.HandleFunc("GET /api/v1/oauth/consent", h.consent)
	mux.HandleFunc("POST /api/v1/oauth/consent", h.authorizePost)
	mux.HandleFunc("POST /oauth/token", h.token)
	mux.HandleFunc("POST /oauth/revoke", h.revoke)
	mux.HandleFunc("GET /test-session", func(w http.ResponseWriter, r *http.Request) {
		sess.Put(r.Context(), "user_id", uuid.NewString())
		w.WriteHeader(204)
	})
	server := httptest.NewServer(sess.LoadAndSave(mux))
	defer server.Close()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Get(server.URL + "/test-session")
	require.NoError(t, err)
	response.Body.Close()
	verifier := strings.Repeat("v", 41) + ".~"
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	q := url.Values{"client_id": {"client-1"}, "redirect_uri": {"https://client.example/cb"}, "response_type": {"code"}, "code_challenge_method": {"S256"}, "code_challenge": {challenge}, "state": {"opaque"}, "scope": {"koi:read"}}
	response, err = client.Get(server.URL + "/oauth/authorize?" + q.Encode())
	require.NoError(t, err)
	response.Body.Close()
	require.Equal(t, 303, response.StatusCode)
	require.Equal(t, "http://localhost:8080/oauth/consent", response.Header.Get("Location"))
	response, err = client.Get(server.URL + "/api/v1/oauth/consent")
	require.NoError(t, err)
	var consent struct {
		CSRF       string `json:"csrf"`
		ClientName string `json:"clientName"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&consent))
	response.Body.Close()
	require.Equal(t, "Test client", consent.ClientName)
	badPayload, _ := json.Marshal(map[string]string{"csrf": "wrong", "action": "approve"})
	bad, err := client.Post(server.URL+"/api/v1/oauth/consent", "application/json", strings.NewReader(string(badPayload)))
	require.NoError(t, err)
	bad.Body.Close()
	require.Equal(t, 400, bad.StatusCode)
	payload, _ := json.Marshal(map[string]string{"csrf": consent.CSRF, "action": "approve"})
	response, err = client.Post(server.URL+"/api/v1/oauth/consent", "application/json", strings.NewReader(string(payload)))
	require.NoError(t, err)
	var decision struct {
		RedirectURL string `json:"redirectUrl"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&decision))
	response.Body.Close()
	require.Equal(t, 200, response.StatusCode)
	redirect, err := url.Parse(decision.RedirectURL)
	require.NoError(t, err)
	code := redirect.Query().Get("code")
	require.NotEmpty(t, code)
	require.Equal(t, "opaque", redirect.Query().Get("state"))
	form := url.Values{"grant_type": {"authorization_code"}, "client_id": {"client-1"}, "redirect_uri": {"https://client.example/cb"}, "code": {code}, "code_verifier": {verifier}}
	response, err = client.PostForm(server.URL+"/oauth/token", form)
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, 200, response.StatusCode)
	var tokens struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		Scope        string `json:"scope"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&tokens))
	require.Equal(t, "koi:read", tokens.Scope)
	stored, err := store.CredentialByToken(context.Background(), tokens.AccessToken)
	require.NoError(t, err)
	require.Equal(t, "http://localhost:8080/mcp", stored.Audience)
	replay, err := client.PostForm(server.URL+"/oauth/token", form)
	require.NoError(t, err)
	replay.Body.Close()
	require.Equal(t, 400, replay.StatusCode)
	response, err = client.Get(server.URL + "/oauth/authorize?" + q.Encode())
	require.NoError(t, err)
	response.Body.Close()
	require.Equal(t, 303, response.StatusCode)
	response, err = client.Get(server.URL + "/api/v1/oauth/consent")
	require.NoError(t, err)
	require.NoError(t, json.NewDecoder(response.Body).Decode(&consent))
	response.Body.Close()
	denyPayload, _ := json.Marshal(map[string]string{"csrf": consent.CSRF, "action": "deny"})
	response, err = client.Post(server.URL+"/api/v1/oauth/consent", "application/json", strings.NewReader(string(denyPayload)))
	require.NoError(t, err)
	require.NoError(t, json.NewDecoder(response.Body).Decode(&decision))
	response.Body.Close()
	deniedURL, err := url.Parse(decision.RedirectURL)
	require.NoError(t, err)
	require.Equal(t, "access_denied", deniedURL.Query().Get("error"))
	require.Equal(t, "opaque", deniedURL.Query().Get("state"))
	revoke, err := client.PostForm(server.URL+"/oauth/revoke", url.Values{"token": {tokens.AccessToken}})
	require.NoError(t, err)
	revoke.Body.Close()
	stored, err = store.CredentialByToken(context.Background(), tokens.AccessToken)
	require.NoError(t, err)
	require.Nil(t, stored)
}

// TestOAuthDynamicClientFlowPersistsInPostgres runs the flow an MCP client takes, through the real
// routes and store: discovery, dynamic registration, consent, code exchange and code replay.
func TestOAuthDynamicClientFlowPersistsInPostgres(t *testing.T) {
	server := newIntegrationServer(t)
	browser, _ := server.signup(t)
	noRedirect := &http.Client{
		Jar:       browser.Jar,
		Transport: browser.Transport,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	metadata, err := browser.Get(server.URL + "/.well-known/oauth-authorization-server")
	require.NoError(t, err)
	var discovered struct {
		RegistrationEndpoint string   `json:"registration_endpoint"`
		Methods              []string `json:"code_challenge_methods_supported"`
	}
	require.NoError(t, json.NewDecoder(metadata.Body).Decode(&discovered))
	require.NoError(t, metadata.Body.Close())
	require.Equal(t, http.StatusOK, metadata.StatusCode)
	require.True(t, strings.HasSuffix(discovered.RegistrationEndpoint, "/oauth/register"))
	require.Equal(t, []string{"S256"}, discovered.Methods)

	for name, body := range map[string]string{
		"no redirect":      `{"client_name":"Agent","redirect_uris":[]}`,
		"plain http":       `{"client_name":"Agent","redirect_uris":["http://agent.example/cb"]}`,
		"client secret":    `{"client_name":"Agent","redirect_uris":["https://agent.example/cb"],"token_endpoint_auth_method":"client_secret_basic"}`,
		"malformed object": `{"client_name":`,
	} {
		t.Run(name, func(t *testing.T) {
			rejected, err := http.Post(server.URL+"/oauth/register", "application/json", strings.NewReader(body))
			require.NoError(t, err)
			require.NoError(t, rejected.Body.Close())
			require.Equal(t, http.StatusBadRequest, rejected.StatusCode)
		})
	}

	registered, err := http.Post(server.URL+"/oauth/register", "application/json",
		strings.NewReader(`{"client_name":"Agent","redirect_uris":["http://127.0.0.1/callback"]}`))
	require.NoError(t, err)
	var client struct {
		ID string `json:"client_id"`
	}
	require.NoError(t, json.NewDecoder(registered.Body).Decode(&client))
	require.NoError(t, registered.Body.Close())
	require.Equal(t, http.StatusCreated, registered.StatusCode)

	verifier := strings.Repeat("v", 43)
	sum := sha256.Sum256([]byte(verifier))
	authorize := url.Values{
		"client_id":             {client.ID},
		"redirect_uri":          {"http://127.0.0.1:53102/callback"},
		"response_type":         {"code"},
		"code_challenge_method": {"S256"},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(sum[:])},
		"state":                 {"opaque"},
		"scope":                 {"koi:read"},
	}
	redirect, err := noRedirect.Get(server.URL + "/oauth/authorize?" + authorize.Encode())
	require.NoError(t, err)
	require.NoError(t, redirect.Body.Close())
	require.Equal(t, http.StatusSeeOther, redirect.StatusCode)

	consent, err := browser.Get(server.URL + "/api/v1/oauth/consent")
	require.NoError(t, err)
	var pending struct {
		CSRF       string `json:"csrf"`
		ClientName string `json:"clientName"`
	}
	require.NoError(t, json.NewDecoder(consent.Body).Decode(&pending))
	require.NoError(t, consent.Body.Close())
	require.Equal(t, "Agent", pending.ClientName)

	approved := doJSON(t, browser, http.MethodPost, server.URL+"/api/v1/oauth/consent",
		`{"csrf":"`+pending.CSRF+`","action":"approve"}`)
	var decision struct {
		RedirectURL string `json:"redirectUrl"`
	}
	readJSON(t, approved, &decision)
	callback, err := url.Parse(decision.RedirectURL)
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1:53102", callback.Host)
	require.Equal(t, "opaque", callback.Query().Get("state"))

	exchange := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {client.ID},
		"redirect_uri":  {"http://127.0.0.1:53102/callback"},
		"code":          {callback.Query().Get("code")},
		"code_verifier": {verifier},
	}
	tokens, err := http.PostForm(server.URL+"/oauth/token", exchange)
	require.NoError(t, err)
	var issued struct {
		AccessToken string `json:"access_token"`
		Scope       string `json:"scope"`
	}
	require.NoError(t, json.NewDecoder(tokens.Body).Decode(&issued))
	require.NoError(t, tokens.Body.Close())
	require.Equal(t, http.StatusOK, tokens.StatusCode)
	require.NotEmpty(t, issued.AccessToken)
	require.Equal(t, "koi:read", issued.Scope)

	// A code works once, even when it is stored in Postgres.
	replay, err := http.PostForm(server.URL+"/oauth/token", exchange)
	require.NoError(t, err)
	require.NoError(t, replay.Body.Close())
	require.Equal(t, http.StatusBadRequest, replay.StatusCode)
}
