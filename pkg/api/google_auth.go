package api

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/koihq/koi/pkg/api/response"
	"github.com/koihq/koi/pkg/domain"
)

const (
	googleAuthURL  = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenURL = "https://oauth2.googleapis.com/token"

	googleCallbackPath = "/api/v1/auth/google/callback"

	googleStateKey    = "google_state"
	googleNonceKey    = "google_nonce"
	googleVerifierKey = "google_verifier"
	googleReturnToKey = "google_return_to"
)

// GoogleAuth turns on "Continue with Google". Nil leaves email and password as the only sign-in.
type GoogleAuth struct {
	ClientID     string
	ClientSecret string
	// PublicURL is the browser origin Google sends the user back to.
	PublicURL string
	// AuthURL and TokenURL default to Google's; tests point them at a fake provider.
	AuthURL  string
	TokenURL string
}

func (g *GoogleAuth) oauthConfig() *oauth2.Config {
	if g == nil {
		return nil
	}

	authURL := g.AuthURL
	if authURL == "" {
		authURL = googleAuthURL
	}

	tokenURL := g.TokenURL
	if tokenURL == "" {
		tokenURL = googleTokenURL
	}

	return &oauth2.Config{
		ClientID:     g.ClientID,
		ClientSecret: g.ClientSecret,
		RedirectURL:  g.PublicURL + googleCallbackPath,
		Scopes:       []string{"openid", "email"},
		Endpoint: oauth2.Endpoint{
			AuthURL:   authURL,
			TokenURL:  tokenURL,
			AuthStyle: oauth2.AuthStyleInParams,
		},
	}
}

type AuthProvidersRes struct {
	Google bool `json:"google"`
}

func (h *AuthHandler) Providers(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, r, http.StatusOK, AuthProvidersRes{Google: h.google != nil})
}

// GoogleStart sends the browser to Google, keeping what the callback must check in the session.
func (h *AuthHandler) GoogleStart(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	state := rand.Text()
	nonce := rand.Text()
	verifier := oauth2.GenerateVerifier()

	h.sess.Put(ctx, googleStateKey, state)
	h.sess.Put(ctx, googleNonceKey, nonce)
	h.sess.Put(ctx, googleVerifierKey, verifier)
	h.sess.Put(ctx, googleReturnToKey, safeReturnTo(r.URL.Query().Get("returnTo")))

	target := h.google.AuthCodeURL(
		state,
		oauth2.S256ChallengeOption(verifier),
		oauth2.SetAuthURLParam("nonce", nonce),
		oauth2.SetAuthURLParam("prompt", "select_account"),
	)

	http.Redirect(w, r, target, http.StatusFound)
}

// GoogleCallback signs the user in and always answers with a redirect, so the browser never
// lands on a JSON error. Failures go back to the sign-in page with a code the UI explains.
func (h *AuthHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Each value is single use, whatever the outcome.
	state := h.sess.PopString(ctx, googleStateKey)
	nonce := h.sess.PopString(ctx, googleNonceKey)
	verifier := h.sess.PopString(ctx, googleVerifierKey)
	returnTo := h.sess.PopString(ctx, googleReturnToKey)

	email, code := h.googleEmail(ctx, r.URL.Query(), state, nonce, verifier)
	if code != "" {
		redirectSignInError(w, r, code)

		return
	}

	me, err := h.commander.SignInWithGoogle(ctx, email)
	if errors.Is(err, domain.ErrSignupClosed) {
		redirectSignInError(w, r, "signup_closed")

		return
	}

	if err != nil {
		slog.Error("google sign-in", "err", err)
		redirectSignInError(w, r, "google_failed")

		return
	}

	if err := h.saveUser(r, me); err != nil {
		slog.Error("google sign-in session", "user_id", me.User.ID, "err", err)
		redirectSignInError(w, r, "google_failed")

		return
	}

	http.Redirect(w, r, safeReturnTo(returnTo), http.StatusFound)
}

// googleEmail returns the verified email of the callback, or the error code to show instead.
func (h *AuthHandler) googleEmail(ctx context.Context, query url.Values, state, nonce, verifier string) (string, string) {
	if reason := query.Get("error"); reason != "" {
		if reason == "access_denied" {
			return "", "google_cancelled"
		}

		slog.Warn("google sign-in refused", "reason", reason)

		return "", "google_failed"
	}

	if state == "" || subtle.ConstantTimeCompare([]byte(state), []byte(query.Get("state"))) != 1 {
		slog.Warn("google sign-in state mismatch")

		return "", "google_failed"
	}

	exchangeCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	token, err := h.google.Exchange(exchangeCtx, query.Get("code"), oauth2.VerifierOption(verifier))
	if err != nil {
		slog.Error("google token exchange", "err", err)

		return "", "google_failed"
	}

	rawIDToken, _ := token.Extra("id_token").(string)

	claims, err := parseGoogleIDToken(rawIDToken, h.google.ClientID, nonce, time.Now())
	if err != nil {
		slog.Error("google id token", "err", err)

		return "", "google_failed"
	}

	if !claims.EmailVerified {
		return "", "google_unverified"
	}

	return claims.Email, ""
}

type googleClaims struct {
	Issuer        string `json:"iss"`
	Audience      string `json:"aud"`
	Expiry        int64  `json:"exp"`
	Nonce         string `json:"nonce"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

// parseGoogleIDToken reads the ID token returned by the token endpoint. It came straight from
// Google over TLS, so the issuer is already authenticated and the signature need not be checked
// (OpenID Connect Core 3.1.3.7); the claims that bind it to this client and request still are.
func parseGoogleIDToken(raw, clientID, nonce string, now time.Time) (googleClaims, error) {
	var claims googleClaims

	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return claims, errors.New("id token is not a JWT")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return claims, fmt.Errorf("decode id token: %w", err)
	}

	if err := json.Unmarshal(payload, &claims); err != nil {
		return claims, fmt.Errorf("parse id token: %w", err)
	}

	if claims.Issuer != "https://accounts.google.com" && claims.Issuer != "accounts.google.com" {
		return claims, fmt.Errorf("unexpected issuer %q", claims.Issuer)
	}

	if claims.Audience != clientID {
		return claims, errors.New("id token is for another client")
	}

	if now.Unix() >= claims.Expiry {
		return claims, errors.New("id token expired")
	}

	if nonce == "" || subtle.ConstantTimeCompare([]byte(claims.Nonce), []byte(nonce)) != 1 {
		return claims, errors.New("id token nonce mismatch")
	}

	return claims, nil
}

// safeReturnTo keeps the redirect on this origin: a path, never another host.
func safeReturnTo(raw string) string {
	if !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") || strings.ContainsAny(raw, "\\\r\n") {
		return "/"
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "" || parsed.Host != "" {
		return "/"
	}

	return raw
}

func redirectSignInError(w http.ResponseWriter, r *http.Request, code string) {
	http.Redirect(w, r, "/sign-in?error="+url.QueryEscape(code), http.StatusFound)
}
