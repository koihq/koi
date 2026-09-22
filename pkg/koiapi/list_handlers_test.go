package koiapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/koiapi/api"
	"github.com/koihq/koi/pkg/koidomain"
)

func newListsTestServer(
	t *testing.T,
	pq koidomain.PromptQuerier,
	pc koidomain.PromptCommander,
	cq koidomain.CompetitorQuerier,
	cc koidomain.CompetitorCommander,
) *httptest.Server {
	t.Helper()
	sess := scs.New()
	sess.Cookie.Name = "koi_session"
	r := chi.NewRouter()
	r.Use(sess.LoadAndSave)
	r.Get("/test-session/{id}", func(w http.ResponseWriter, r *http.Request) {
		sess.Put(r.Context(), "user_id", chi.URLParam(r, "id"))
		w.WriteHeader(http.StatusNoContent)
	})
	r.Route("/api/v1/brands/{brandID}/prompts", NewPromptHandler(pq, pc, sess).Routes())
	r.Route("/api/v1/brands/{brandID}/competitors", NewCompetitorHandler(cq, cc, sess).Routes())
	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)
	return ts
}

func TestListHandlersRequireSession(t *testing.T) {
	ts := newListsTestServer(t,
		NewMockPromptQuerier(t), NewMockPromptCommander(t),
		NewMockCompetitorQuerier(t), NewMockCompetitorCommander(t),
	)
	client := newCookieClient(t)
	brandID, childID := uuid.New(), uuid.New()
	paths := []struct{ method, path, body string }{
		{http.MethodGet, "/prompts", ""},
		{http.MethodPost, "/prompts", `{}`},
		{http.MethodPatch, "/prompts/" + childID.String(), `{}`},
		{http.MethodDelete, "/prompts/" + childID.String(), ""},
		{http.MethodPost, "/prompts/" + childID.String() + "/confirm", ""},
		{http.MethodGet, "/competitors", ""},
	}
	for _, request := range paths {
		resp := doJSON(t, client, request.method, ts.URL+"/api/v1/brands/"+brandID.String()+request.path, request.body)
		var body api.Envelope
		readJSON(t, resp, &body)
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		require.Equal(t, koidomain.CodeUnauthorized, body.Error.Code)
	}
}

func TestPromptHandlerCreateMalformedAndUnowned(t *testing.T) {
	userID, brandID := uuid.New(), uuid.New()
	commander := NewMockPromptCommander(t)
	querier := NewMockPromptQuerier(t)
	querier.EXPECT().List(mock.Anything, userID, brandID).Return(nil, koidomain.ErrNotFound)
	ts := newListsTestServer(t, querier, commander, NewMockCompetitorQuerier(t), NewMockCompetitorCommander(t))
	client := newCookieClient(t)
	brandSession(t, client, ts.URL, userID)

	resp := doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/brands/"+brandID.String()+"/prompts", `{`)
	var malformed api.Envelope
	readJSON(t, resp, &malformed)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	require.Equal(t, koidomain.CodeInvalidJSON, malformed.Error.Code)

	resp = doJSON(t, client, http.MethodGet, ts.URL+"/api/v1/brands/"+brandID.String()+"/prompts", "")
	var missing api.Envelope
	readJSON(t, resp, &missing)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	require.Equal(t, koidomain.CodeNotFound, missing.Error.Code)
}

func TestPromptHandlerConfirmationIsIdempotent(t *testing.T) {
	userID, brandID, promptID := uuid.New(), uuid.New(), uuid.New()
	commander := NewMockPromptCommander(t)
	prompt := &koidomain.Prompt{
		BaseEntity: koidomain.BaseEntity{ID: promptID}, BrandID: brandID,
		Text: "What is Koi?", Source: koidomain.PromptSourceDrafted, Confirmed: true,
	}
	commander.EXPECT().Confirm(mock.Anything, userID, brandID, promptID).Return(prompt, nil).Twice()
	ts := newListsTestServer(t, NewMockPromptQuerier(t), commander, NewMockCompetitorQuerier(t), NewMockCompetitorCommander(t))
	client := newCookieClient(t)
	brandSession(t, client, ts.URL, userID)

	for range 2 {
		resp := doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/brands/"+brandID.String()+"/prompts/"+promptID.String()+"/confirm", "")
		var body koidomain.Prompt
		readJSON(t, resp, &body)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.True(t, body.Confirmed)
	}
}

func TestCompetitorHandlerManualRequestCannotChangeProvenance(t *testing.T) {
	userID, brandID := uuid.New(), uuid.New()
	commander := NewMockCompetitorCommander(t)
	commander.EXPECT().Create(mock.Anything, userID, brandID, koidomain.CompetitorParams{
		Name: "Acme", Domain: "acme.com", Aliases: []string{"Acme AI"},
	}).Return(&koidomain.Competitor{
		BrandID: brandID, Name: "Acme", Domain: "acme.com", Aliases: []string{"Acme AI"},
		Source: koidomain.CompetitorSourceManual, Confirmed: true,
	}, nil)
	ts := newListsTestServer(t, NewMockPromptQuerier(t), NewMockPromptCommander(t), NewMockCompetitorQuerier(t), commander)
	client := newCookieClient(t)
	brandSession(t, client, ts.URL, userID)

	resp := doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/brands/"+brandID.String()+"/competitors",
		`{"name":"Acme","domain":"acme.com","aliases":["Acme AI"],"source":"observed","confirmed":false}`)
	var body koidomain.Competitor
	readJSON(t, resp, &body)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	require.Equal(t, koidomain.CompetitorSourceManual, body.Source)
	require.True(t, body.Confirmed)
}
