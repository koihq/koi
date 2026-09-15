package koiapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/koidomain"
)

type fakeAuth struct {
	me        *koidomain.Me
	signupErr error
	loginErr  error
	meErr     error
}

func (f *fakeAuth) Signup(context.Context, koidomain.SignupParams) (*koidomain.Me, error) {
	if f.signupErr != nil {
		return nil, f.signupErr
	}
	return f.me, nil
}

func (f *fakeAuth) Login(context.Context, string, string) (*koidomain.Me, error) {
	if f.loginErr != nil {
		return nil, f.loginErr
	}
	return f.me, nil
}

func (f *fakeAuth) Me(context.Context, uuid.UUID) (*koidomain.Me, error) {
	if f.meErr != nil {
		return nil, f.meErr
	}
	return f.me, nil
}

func sampleMe() *koidomain.Me {
	return &koidomain.Me{
		User: koidomain.User{
			BaseEntity: koidomain.BaseEntity{ID: uuid.MustParse("11111111-1111-1111-1111-111111111111")},
			Email:      "you@example.com",
		},
		Organization: koidomain.Organization{
			BaseEntity: koidomain.BaseEntity{ID: uuid.MustParse("22222222-2222-2222-2222-222222222222")},
			Name:       "you",
		},
		Role: koidomain.RoleOwner,
	}
}

func newAuthTestServer(t *testing.T, fake *fakeAuth) *httptest.Server {
	t.Helper()
	sess := scs.New()
	sess.Cookie.Name = "koi_session"
	r := chi.NewRouter()
	r.Use(sess.LoadAndSave)
	r.Route("/api/v1/auth", NewAuthHandler(fake, fake, sess).Routes())
	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)
	return ts
}

func newCookieClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	return &http.Client{Jar: jar}
}

func doJSON(t *testing.T, client *http.Client, method, url, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	require.NoError(t, err)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	require.NoError(t, err)
	return resp
}

func readJSON(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer resp.Body.Close()
	require.NoError(t, json.NewDecoder(resp.Body).Decode(v))
}

func TestAuthHandler_SignupSetsCookie(t *testing.T) {
	ts := newAuthTestServer(t, &fakeAuth{me: sampleMe()})
	client := newCookieClient(t)

	resp := doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/auth/signup",
		`{"email":"you@example.com","password":"password1"}`)
	var body MeRes
	readJSON(t, resp, &body)

	require.Equal(t, http.StatusCreated, resp.StatusCode)
	require.Equal(t, "you@example.com", body.Email)
	require.Equal(t, "you", body.Organization.Name)
	require.Equal(t, "owner", body.Organization.Role)
	found := false
	for _, c := range resp.Cookies() {
		if c.Name == "koi_session" && c.Value != "" {
			found = true
			break
		}
	}
	require.True(t, found, "expected Set-Cookie koi_session")
}

func TestAuthHandler_LoginInvalidCredentials(t *testing.T) {
	ts := newAuthTestServer(t, &fakeAuth{loginErr: koidomain.ErrInvalidCredentials})
	client := newCookieClient(t)

	resp := doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/auth/login",
		`{"email":"you@example.com","password":"wrongpass"}`)
	var body map[string]string
	readJSON(t, resp, &body)

	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	require.Equal(t, "invalid credentials", body["error"])
}

func TestAuthHandler_MeUnauthorized(t *testing.T) {
	ts := newAuthTestServer(t, &fakeAuth{me: sampleMe()})
	resp, err := http.Get(ts.URL + "/api/v1/auth/me")
	require.NoError(t, err)
	var body map[string]string
	readJSON(t, resp, &body)

	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	require.Equal(t, "unauthorized", body["error"])
}

func TestAuthHandler_SessionFlow(t *testing.T) {
	ts := newAuthTestServer(t, &fakeAuth{me: sampleMe()})
	client := newCookieClient(t)
	creds := `{"email":"you@example.com","password":"password1"}`

	resp := doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/auth/signup", creds)
	var created MeRes
	readJSON(t, resp, &created)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	resp, err := client.Get(ts.URL + "/api/v1/auth/me")
	require.NoError(t, err)
	var me MeRes
	readJSON(t, resp, &me)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, created.Email, me.Email)
	require.Equal(t, created.ID, me.ID)

	resp = doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/auth/logout", "")
	resp.Body.Close()
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	resp, err = client.Get(ts.URL + "/api/v1/auth/me")
	require.NoError(t, err)
	var after map[string]string
	readJSON(t, resp, &after)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
