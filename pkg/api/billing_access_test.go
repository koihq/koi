package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/api/middlewares"
	"github.com/koihq/koi/pkg/api/response"
	"github.com/koihq/koi/pkg/domain"
)

type testProductAccess struct {
	allowed bool
	err     error
}

type accessOnlyBillingRepo struct {
	domain.BillingQuerier
}

func (accessOnlyBillingRepo) ProductAccess(context.Context, uuid.UUID) (bool, error) {
	return false, nil
}

func TestBrandCreationPrecedesBillingButEditRequiresIt(t *testing.T) {
	access := &testProductAccess{}
	sess := scs.New()
	commander := NewMockBrandCommander(t)
	querier := NewMockBrandQuerier(t)
	brand := &domain.Brand{BaseEntity: domain.BaseEntity{ID: uuid.New()}, Name: "Koi"}
	commander.EXPECT().Create(mock.Anything, mock.Anything).Return(brand, nil)
	r := chi.NewRouter()
	r.Use(sess.LoadAndSave)
	r.Get("/session", func(w http.ResponseWriter, r *http.Request) {
		sess.Put(r.Context(), "user_id", uuid.NewString())
		w.WriteHeader(http.StatusNoContent)
	})
	r.With(Authorizer(sess)).Route("/brands", NewBrandHandler(querier, commander, sess, middlewares.RequireBilling(access)).Routes())
	server := httptest.NewServer(r)
	t.Cleanup(server.Close)
	client := newCookieClient(t)

	session := doJSON(t, client, http.MethodGet, server.URL+"/session", "")
	require.Equal(t, http.StatusNoContent, session.StatusCode)
	require.NoError(t, session.Body.Close())
	created := doJSON(t, client, http.MethodPost, server.URL+"/brands/",
		`{"name":"Koi","domain":"koi.example","country":"IT"}`)
	require.Equal(t, http.StatusCreated, created.StatusCode)
	require.NoError(t, created.Body.Close())

	denied := doJSON(t, client, http.MethodPatch, server.URL+"/brands/"+brand.ID.String(),
		`{"name":"New name"}`)
	var body response.Envelope
	readJSON(t, denied, &body)
	require.Equal(t, http.StatusPaymentRequired, denied.StatusCode)
	require.Equal(t, domain.CodeBillingRequired, body.Error.Code)
}

func TestCloudServerBlocksNestedProductRoutesBeforeBilling(t *testing.T) {
	sess := scs.New()
	cloud := NewServer(sessionStore(t), sess, ServerOptions{
		AIDrafting:     true,
		LiveCollection: true,
		Billing:        &BillingHandler{querier: accessOnlyBillingRepo{}},
	})
	r := chi.NewRouter()
	r.With(sess.LoadAndSave).Get("/session", func(w http.ResponseWriter, r *http.Request) {
		sess.Put(r.Context(), "user_id", uuid.NewString())
		w.WriteHeader(http.StatusNoContent)
	})
	r.Mount("/", cloud)
	server := httptest.NewServer(r)
	t.Cleanup(server.Close)
	client := newCookieClient(t)
	session := doJSON(t, client, http.MethodGet, server.URL+"/session", "")
	require.Equal(t, http.StatusNoContent, session.StatusCode)
	require.NoError(t, session.Body.Close())

	for _, endpoint := range []string{
		"/api/v1" + "/prompts",
		"/api/v1" + "/competitors",
		"/api/v1" + "/generations",
		"/api/v1" + "/runs",
	} {
		res := doJSON(t, client, http.MethodGet, server.URL+endpoint, "")
		var body response.Envelope
		readJSON(t, res, &body)
		require.Equal(t, http.StatusPaymentRequired, res.StatusCode, endpoint)
		require.Equal(t, domain.CodeBillingRequired, body.Error.Code, endpoint)
	}
}

func (a *testProductAccess) ProductAccess(_ context.Context, _ uuid.UUID) (bool, error) {
	return a.allowed, a.err
}

func TestRequireBilling(t *testing.T) {
	access := &testProductAccess{}
	sess := scs.New()
	r := chi.NewRouter()
	r.Use(sess.LoadAndSave)
	r.Get("/session", func(w http.ResponseWriter, r *http.Request) {
		sess.Put(r.Context(), "user_id", uuid.NewString())
		w.WriteHeader(http.StatusNoContent)
	})
	r.With(Authorizer(sess), middlewares.RequireBilling(access)).Get("/product", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	server := httptest.NewServer(r)
	t.Cleanup(server.Close)
	client := newCookieClient(t)

	unauthorized := doJSON(t, client, http.MethodGet, server.URL+"/product", "")
	var unauthorizedBody response.Envelope
	readJSON(t, unauthorized, &unauthorizedBody)
	require.Equal(t, http.StatusUnauthorized, unauthorized.StatusCode)
	require.Equal(t, domain.CodeUnauthorized, unauthorizedBody.Error.Code)

	session := doJSON(t, client, http.MethodGet, server.URL+"/session", "")
	require.Equal(t, http.StatusNoContent, session.StatusCode)
	require.NoError(t, session.Body.Close())

	denied := doJSON(t, client, http.MethodGet, server.URL+"/product", "")
	var deniedBody response.Envelope
	readJSON(t, denied, &deniedBody)
	require.Equal(t, http.StatusPaymentRequired, denied.StatusCode)
	require.Equal(t, domain.CodeBillingRequired, deniedBody.Error.Code)

	access.allowed = true
	allowed := doJSON(t, client, http.MethodGet, server.URL+"/product", "")
	require.Equal(t, http.StatusNoContent, allowed.StatusCode)
	require.NoError(t, allowed.Body.Close())
}
