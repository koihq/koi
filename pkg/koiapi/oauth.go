package koiapi

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/google/uuid"
	"github.com/koihq/koi/pkg/koidomain"
)

type oauthHandler struct {
	store  koidomain.CredentialService
	sess   *scs.SessionManager
	origin string
}

func oauthOrigin() string {
	if v := strings.TrimRight(os.Getenv("KOI_MCP_PUBLIC_URL"), "/"); v != "" {
		return v
	}
	if v := strings.TrimRight(os.Getenv("KOI_PUBLIC_URL"), "/"); v != "" {
		return v
	}
	return "http://localhost:8080"
}
func (h *oauthHandler) resourceURL() string { return h.origin + "/mcp" }
func oauthUIOrigin() string {
	if v := strings.TrimRight(os.Getenv("KOI_PUBLIC_URL"), "/"); v != "" {
		return v
	}
	return oauthOrigin()
}
func oauthJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func oauthError(w http.ResponseWriter, status int, code string) {
	oauthJSON(w, status, map[string]string{"error": code})
}
func (h *oauthHandler) serverMetadata(w http.ResponseWriter, r *http.Request) {
	oauthJSON(w, 200, map[string]any{"issuer": h.origin, "authorization_endpoint": h.origin + "/oauth/authorize", "token_endpoint": h.origin + "/oauth/token", "registration_endpoint": h.origin + "/oauth/register", "revocation_endpoint": h.origin + "/oauth/revoke", "response_types_supported": []string{"code"}, "grant_types_supported": []string{"authorization_code", "refresh_token"}, "code_challenge_methods_supported": []string{"S256"}, "token_endpoint_auth_methods_supported": []string{"none"}, "scopes_supported": []string{"koi:read", "koi:write"}, "client_id_metadata_document_supported": true})
}
func validRedirect(raw string) bool {
	u, e := url.Parse(raw)
	if e != nil || u.Host == "" || u.Fragment != "" || u.User != nil {
		return false
	}
	return u.Scheme == "https" || (u.Scheme == "http" && loopbackHost(u.Hostname()))
}

func loopbackHost(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func redirectAllowed(registered []string, requested string) bool {
	if slices.Contains(registered, requested) {
		return true
	}

	requestURI, err := url.Parse(requested)
	if err != nil || requestURI.Scheme != "http" || !loopbackHost(requestURI.Hostname()) || !validRedirect(requested) {
		return false
	}

	for _, raw := range registered {
		registeredURI, err := url.Parse(raw)
		if err != nil {
			continue
		}
		if registeredURI.Scheme == requestURI.Scheme && registeredURI.Hostname() == requestURI.Hostname() &&
			registeredURI.EscapedPath() == requestURI.EscapedPath() && registeredURI.RawQuery == requestURI.RawQuery &&
			registeredURI.ForceQuery == requestURI.ForceQuery {
			return true
		}
	}
	return false
}
func (h *oauthHandler) register(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name            string   `json:"client_name"`
		RedirectURIs    []string `json:"redirect_uris"`
		TokenAuthMethod string   `json:"token_endpoint_auth_method"`
	}
	if e := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&in); e != nil || len(in.RedirectURIs) == 0 || len(in.RedirectURIs) > 10 || len(in.Name) > 200 || (in.TokenAuthMethod != "" && in.TokenAuthMethod != "none") {
		oauthError(w, 400, "invalid_client_metadata")
		return
	}
	for _, uri := range in.RedirectURIs {
		if !validRedirect(uri) {
			oauthError(w, 400, "invalid_redirect_uri")
			return
		}
	}
	id, e := koidomain.RandomCredential()
	if e != nil {
		oauthError(w, 500, "server_error")
		return
	}
	c := &koidomain.OAuthClient{ID: id, Name: in.Name, RedirectURIs: in.RedirectURIs}
	if e := h.store.RegisterClient(r.Context(), c); e != nil {
		oauthError(w, 500, "server_error")
		return
	}
	oauthJSON(w, 201, map[string]any{"client_id": id, "client_name": in.Name, "redirect_uris": in.RedirectURIs, "token_endpoint_auth_method": "none"})
}
func publicIP(ip net.IP) bool {
	return ip != nil && !ip.IsLoopback() && !ip.IsPrivate() && !ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast() && !ip.IsMulticast() && !ip.IsUnspecified()
}
func fetchCIMD(ctx context.Context, id string) (*koidomain.OAuthClient, error) {
	u, e := url.Parse(id)
	if e != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Fragment != "" {
		return nil, fmt.Errorf("invalid client id")
	}
	transport := &http.Transport{DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, e := net.SplitHostPort(address)
		if e != nil {
			return nil, e
		}
		ips, e := net.DefaultResolver.LookupIP(ctx, "ip", host)
		if e != nil {
			return nil, e
		}
		for _, ip := range ips {
			if publicIP(ip) {
				return (&net.Dialer{Timeout: 3 * time.Second}).DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
			}
		}
		return nil, fmt.Errorf("private address")
	}}
	client := &http.Client{Timeout: 5 * time.Second, Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	req, e := http.NewRequestWithContext(ctx, "GET", id, nil)
	if e != nil {
		return nil, e
	}
	res, e := client.Do(req)
	if e != nil {
		return nil, e
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("client metadata status %d", res.StatusCode)
	}
	var doc struct {
		ClientID     string   `json:"client_id"`
		Name         string   `json:"client_name"`
		RedirectURIs []string `json:"redirect_uris"`
	}
	if e := json.NewDecoder(io.LimitReader(res.Body, 16<<10)).Decode(&doc); e != nil {
		return nil, e
	}
	if doc.ClientID != id || len(doc.RedirectURIs) == 0 {
		return nil, fmt.Errorf("invalid client metadata")
	}
	for _, uri := range doc.RedirectURIs {
		if !validRedirect(uri) {
			return nil, fmt.Errorf("invalid redirect")
		}
	}
	return &koidomain.OAuthClient{ID: id, Name: doc.Name, RedirectURIs: doc.RedirectURIs}, nil
}
func (h *oauthHandler) client(ctx context.Context, id string) (*koidomain.OAuthClient, error) {
	if strings.HasPrefix(id, "https://") {
		return fetchCIMD(ctx, id)
	}
	return h.store.Client(ctx, id)
}
func validChallenge(s string) bool {
	if len(s) < 43 || len(s) > 128 {
		return false
	}
	for _, c := range s {
		if c > 127 || !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}
func (h *oauthHandler) authorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	id := q.Get("client_id")
	client, e := h.client(r.Context(), id)
	redirect := q.Get("redirect_uri")
	if e != nil || client == nil || !redirectAllowed(client.RedirectURIs, redirect) || q.Get("response_type") != "code" || q.Get("code_challenge_method") != "S256" || !validChallenge(q.Get("code_challenge")) || q.Get("resource") != "" && q.Get("resource") != h.resourceURL() {
		oauthError(w, 400, "invalid_request")
		return
	}
	scope := q.Get("scope")
	if scope == "" {
		scope = "koi:read koi:write"
	}
	for _, item := range strings.Fields(scope) {
		if item != "koi:read" && item != "koi:write" {
			oauthError(w, 400, "invalid_scope")
			return
		}
	}
	csrf, e := koidomain.RandomCredential()
	if e != nil {
		oauthError(w, 500, "server_error")
		return
	}
	h.sess.Put(r.Context(), "oauth_csrf", csrf)
	h.sess.Put(r.Context(), "oauth_query", r.URL.RawQuery)
	http.Redirect(w, r, oauthUIOrigin()+"/oauth/consent", http.StatusSeeOther)
}

func (h *oauthHandler) consent(w http.ResponseWriter, r *http.Request) {
	raw := h.sess.GetString(r.Context(), "oauth_query")
	if raw == "" {
		oauthError(w, 400, "invalid_request")
		return
	}
	q, err := url.ParseQuery(raw)
	if err != nil {
		oauthError(w, 400, "invalid_request")
		return
	}
	client, err := h.client(r.Context(), q.Get("client_id"))
	if err != nil || client == nil || !redirectAllowed(client.RedirectURIs, q.Get("redirect_uri")) {
		oauthError(w, 400, "invalid_request")
		return
	}
	scope := q.Get("scope")
	if scope == "" {
		scope = "koi:read koi:write"
	}
	_, signedIn := h.sess.Get(r.Context(), "user_id").(string)
	oauthJSON(w, 200, map[string]any{"clientName": client.Name, "scopes": strings.Fields(scope), "signedIn": signedIn, "csrf": h.sess.GetString(r.Context(), "oauth_csrf"), "redirectHost": redirectHost(q.Get("redirect_uri"))})
}

func redirectHost(raw string) string {
	u, _ := url.Parse(raw)
	if u == nil {
		return ""
	}
	return u.Host
}

func (h *oauthHandler) authorizePost(w http.ResponseWriter, r *http.Request) {
	var body struct {
		CSRF   string `json:"csrf"`
		Action string `json:"action"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&body); err != nil || body.CSRF == "" || subtle.ConstantTimeCompare([]byte(body.CSRF), []byte(h.sess.GetString(r.Context(), "oauth_csrf"))) != 1 || (body.Action != "approve" && body.Action != "deny") {
		oauthError(w, 400, "invalid_request")
		return
	}
	raw := h.sess.GetString(r.Context(), "oauth_query")
	q, e := url.ParseQuery(raw)
	if e != nil {
		oauthError(w, 400, "invalid_request")
		return
	}
	client, e := h.client(r.Context(), q.Get("client_id"))
	if e != nil || client == nil || !redirectAllowed(client.RedirectURIs, q.Get("redirect_uri")) {
		oauthError(w, 400, "invalid_request")
		return
	}
	id, e := uuid.Parse(h.sess.GetString(r.Context(), "user_id"))
	if e != nil || id == uuid.Nil {
		oauthError(w, 401, "unauthorized")
		return
	}
	redirect, _ := url.Parse(q.Get("redirect_uri"))
	params := redirect.Query()
	if q.Get("state") != "" {
		params.Set("state", q.Get("state"))
	}
	if body.Action != "approve" {
		params.Set("error", "access_denied")
	} else {
		code, e := koidomain.RandomCredential()
		if e != nil {
			oauthError(w, 500, "server_error")
			return
		}
		scope := q.Get("scope")
		if scope == "" {
			scope = "koi:read koi:write"
		}
		record := &koidomain.OAuthCode{Hash: koidomain.HashCredential(code), UserID: id, ClientID: client.ID, ClientName: client.Name, RedirectURI: q.Get("redirect_uri"), Challenge: q.Get("code_challenge"), Scope: scope, ExpiresAt: time.Now().Add(5 * time.Minute)}
		if e := h.store.CreateCode(r.Context(), record); e != nil {
			oauthError(w, 500, "server_error")
			return
		}
		params.Set("code", code)
	}
	h.sess.Remove(r.Context(), "oauth_csrf")
	h.sess.Remove(r.Context(), "oauth_query")
	redirect.RawQuery = params.Encode()
	oauthJSON(w, 200, map[string]string{"redirectUrl": redirect.String()})
}
func (h *oauthHandler) issueTokens(w http.ResponseWriter, r *http.Request, userID uuid.UUID, clientID, name, scope string, connectionID uuid.UUID) {
	tokens, err := h.store.IssueTokens(r.Context(), userID, clientID, name, scope, h.resourceURL(), connectionID)
	if err != nil {
		oauthError(w, 500, "server_error")
		return
	}
	oauthJSON(w, 200, map[string]any{"access_token": tokens.AccessToken, "token_type": "Bearer", "expires_in": tokens.ExpiresIn, "refresh_token": tokens.RefreshToken, "scope": tokens.Scope})
}
func (h *oauthHandler) token(w http.ResponseWriter, r *http.Request) {
	if e := r.ParseForm(); e != nil {
		oauthError(w, 400, "invalid_request")
		return
	}
	clientID := r.Form.Get("client_id")
	client, e := h.client(r.Context(), clientID)
	if e != nil || client == nil {
		oauthError(w, 401, "invalid_client")
		return
	}
	switch r.Form.Get("grant_type") {
	case "authorization_code":
		code, e := h.store.ConsumeCode(r.Context(), r.Form.Get("code"))
		if e != nil {
			oauthError(w, 500, "server_error")
			return
		}
		if code == nil || time.Now().After(code.ExpiresAt) || code.ClientID != clientID || code.RedirectURI != r.Form.Get("redirect_uri") {
			oauthError(w, 400, "invalid_grant")
			return
		}
		verifier := r.Form.Get("code_verifier")
		hash := sha256.Sum256([]byte(verifier))
		challenge := base64.RawURLEncoding.EncodeToString(hash[:])
		if !validChallenge(verifier) || subtle.ConstantTimeCompare([]byte(challenge), []byte(code.Challenge)) != 1 {
			oauthError(w, 400, "invalid_grant")
			return
		}
		h.issueTokens(w, r, code.UserID, clientID, code.ClientName, code.Scope, uuid.Nil)
	case "refresh_token":
		tokens, err := h.store.RefreshTokens(r.Context(), r.Form.Get("refresh_token"), clientID, h.resourceURL())
		if err != nil {
			oauthError(w, 400, "invalid_grant")
			return
		}
		oauthJSON(w, 200, map[string]any{"access_token": tokens.AccessToken, "token_type": "Bearer", "expires_in": tokens.ExpiresIn, "refresh_token": tokens.RefreshToken, "scope": tokens.Scope})
	default:
		oauthError(w, 400, "unsupported_grant_type")
	}
}
func (h *oauthHandler) revoke(w http.ResponseWriter, r *http.Request) {
	if e := r.ParseForm(); e != nil {
		oauthError(w, 400, "invalid_request")
		return
	}
	if e := h.store.RevokeToken(r.Context(), r.Form.Get("token")); e != nil {
		oauthError(w, 500, "server_error")
		return
	}
	w.WriteHeader(http.StatusOK)
}
