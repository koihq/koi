package koiapi

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/google/uuid"
	"github.com/koihq/koi/pkg/koidomain"
	"github.com/stretchr/testify/require"
)

func TestOAuthOriginSeparatesMCPFromBrowserInDev(t *testing.T) {
	t.Setenv("KOI_PUBLIC_URL", "http://localhost:5173")
	t.Setenv("KOI_MCP_PUBLIC_URL", "http://localhost:8080")
	require.Equal(t, "http://localhost:8080", oauthOrigin())

	t.Setenv("KOI_MCP_PUBLIC_URL", "")
	require.Equal(t, "http://localhost:5173", oauthOrigin())
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
	clients     map[string]*koidomain.OAuthClient
	codes       map[string]*koidomain.OAuthCode
	credentials map[string]*koidomain.Credential
}

func newMemoryOAuthStore() *memoryOAuthStore {
	return &memoryOAuthStore{clients: map[string]*koidomain.OAuthClient{}, codes: map[string]*koidomain.OAuthCode{}, credentials: map[string]*koidomain.Credential{}}
}
func (s *memoryOAuthStore) CreateOAuthClient(_ context.Context, c *koidomain.OAuthClient) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[c.ID] = c
	return nil
}
func (s *memoryOAuthStore) OAuthClient(_ context.Context, id string) (*koidomain.OAuthClient, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.clients[id], nil
}
func (s *memoryOAuthStore) CreateOAuthCode(_ context.Context, c *koidomain.OAuthCode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.codes[c.Hash] = c
	return nil
}
func (s *memoryOAuthStore) ConsumeOAuthCode(_ context.Context, raw string) (*koidomain.OAuthCode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := koidomain.HashCredential(raw)
	c := s.codes[key]
	delete(s.codes, key)
	return c, nil
}
func (s *memoryOAuthStore) CreateCredential(_ context.Context, c *koidomain.Credential) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.credentials[c.TokenHash] = c
	return nil
}
func (s *memoryOAuthStore) CredentialByToken(_ context.Context, raw string) (*koidomain.Credential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.credentials[koidomain.HashCredential(raw)], nil
}
func (s *memoryOAuthStore) ConsumeRefreshToken(_ context.Context, raw string) (*koidomain.Credential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := koidomain.HashCredential(raw)
	c := s.credentials[key]
	delete(s.credentials, key)
	return c, nil
}
func (s *memoryOAuthStore) RevokeToken(_ context.Context, raw string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.credentials, koidomain.HashCredential(raw))
	return nil
}
func (s *memoryOAuthStore) ListCredentials(context.Context, uuid.UUID, koidomain.PageParams) (*koidomain.Page[koidomain.Credential], error) {
	return nil, nil
}
func (s *memoryOAuthStore) RevokeCredential(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return false, nil
}

func (s *memoryOAuthStore) List(context.Context, koidomain.PageParams) (*koidomain.Page[koidomain.Credential], error) {
	return nil, nil
}
func (s *memoryOAuthStore) Revoke(context.Context, uuid.UUID) error { return nil }
func (s *memoryOAuthStore) Resolve(ctx context.Context, raw string) (*koidomain.Credential, error) {
	return s.CredentialByToken(ctx, raw)
}
func (s *memoryOAuthStore) RegisterClient(ctx context.Context, c *koidomain.OAuthClient) error {
	return s.CreateOAuthClient(ctx, c)
}
func (s *memoryOAuthStore) Client(ctx context.Context, id string) (*koidomain.OAuthClient, error) {
	return s.OAuthClient(ctx, id)
}
func (s *memoryOAuthStore) CreateCode(ctx context.Context, c *koidomain.OAuthCode) error {
	return s.CreateOAuthCode(ctx, c)
}
func (s *memoryOAuthStore) ConsumeCode(ctx context.Context, raw string) (*koidomain.OAuthCode, error) {
	return s.ConsumeOAuthCode(ctx, raw)
}
func (s *memoryOAuthStore) IssueTokens(ctx context.Context, userID uuid.UUID, clientID, name, scope, audience string, connectionID uuid.UUID) (koidomain.IssuedTokens, error) {
	access, _ := koidomain.RandomCredential()
	refresh, _ := koidomain.RandomCredential()
	s.CreateCredential(ctx, &koidomain.Credential{UserID: userID, ClientID: clientID, Audience: audience, Scope: scope, Kind: "oauth_access", TokenHash: koidomain.HashCredential(access)})
	s.CreateCredential(ctx, &koidomain.Credential{UserID: userID, ClientID: clientID, Audience: audience, Scope: scope, Kind: "oauth_refresh", TokenHash: koidomain.HashCredential(refresh)})
	return koidomain.IssuedTokens{AccessToken: access, RefreshToken: refresh, Scope: scope, ExpiresIn: 3600}, nil
}
func (s *memoryOAuthStore) RefreshTokens(context.Context, string, string, string) (koidomain.IssuedTokens, error) {
	return koidomain.IssuedTokens{}, nil
}

func TestOAuthAuthorizationCodePKCEAndRevocation(t *testing.T) {
	store := newMemoryOAuthStore()
	sess := scs.New()
	h := &oauthHandler{store: store, sess: sess, origin: "http://localhost:8080"}
	store.clients["client-1"] = &koidomain.OAuthClient{ID: "client-1", Name: "Test client", RedirectURIs: []string{"https://client.example/cb"}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /oauth/authorize", h.authorize)
	mux.HandleFunc("POST /oauth/authorize", h.authorizePost)
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
	verifier := strings.Repeat("v", 43)
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	q := url.Values{"client_id": {"client-1"}, "redirect_uri": {"https://client.example/cb"}, "response_type": {"code"}, "code_challenge_method": {"S256"}, "code_challenge": {challenge}, "state": {"opaque"}, "scope": {"koi:read"}}
	response, err = client.Get(server.URL + "/oauth/authorize?" + q.Encode())
	require.NoError(t, err)
	body, _ := io.ReadAll(response.Body)
	response.Body.Close()
	require.Equal(t, 200, response.StatusCode)
	csrf := regexp.MustCompile(`name="csrf" value="([^"]+)"`).FindSubmatch(body)
	require.Len(t, csrf, 2)
	response, err = client.PostForm(server.URL+"/oauth/authorize", url.Values{"csrf": {string(csrf[1])}, "action": {"approve"}})
	require.NoError(t, err)
	response.Body.Close()
	require.Equal(t, 303, response.StatusCode)
	redirect, err := url.Parse(response.Header.Get("Location"))
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
	revoke, err := client.PostForm(server.URL+"/oauth/revoke", url.Values{"token": {tokens.AccessToken}})
	require.NoError(t, err)
	revoke.Body.Close()
	stored, err = store.CredentialByToken(context.Background(), tokens.AccessToken)
	require.NoError(t, err)
	require.Nil(t, stored)
}
