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
		{http.MethodPatch, "/competitors/" + childID.String(), `{}`},
		{http.MethodGet, "/competitors/search?q=acme", ""},
	}

	for _, request := range paths {
		resp := doJSON(t, client, request.method, ts.URL+"/api/v1/brands/"+brandID.String()+request.path, request.body)
		var body api.Envelope
		readJSON(t, resp, &body)
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		require.Equal(t, koidomain.CodeUnauthorized, body.Error.Code)
	}
}

func TestCompetitorPatchCannotConfirmOrChangeSource(t *testing.T) {
	userID, brandID, competitorID := uuid.New(), uuid.New(), uuid.New()
	commander := NewMockCompetitorCommander(t)
	name := "Acme Analytics"
	aliases := []string{"Acme"}
	urls := []string{"https://acme.tools"}
	commander.EXPECT().Patch(mock.Anything, brandID, competitorID, koidomain.CompetitorUpdate{
		Name: &name, Aliases: &aliases, AdditionalURLs: &urls,
	}).Return(&koidomain.Competitor{
		BaseEntity: koidomain.BaseEntity{
			ID: competitorID,
		},
		BrandID: brandID, Name: name, Source: koidomain.CompetitorSourceObserved,
		Aliases: aliases, AdditionalURLs: urls,
	}, nil)
	ts := newListsTestServer(t, NewMockPromptQuerier(t), NewMockPromptCommander(t), NewMockCompetitorQuerier(t), commander)
	client := newCookieClient(t)
	brandSession(t, client, ts.URL, userID)

	resp := doJSON(t, client, http.MethodPatch, ts.URL+"/api/v1/brands/"+brandID.String()+"/competitors/"+competitorID.String(),
		`{"name":"Acme Analytics","aliases":["Acme"],"additional_urls":["https://acme.tools"],"source":"manual","confirmed":true}`)
	var body koidomain.Competitor
	readJSON(t, resp, &body)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, koidomain.CompetitorSourceObserved, body.Source)
	require.False(t, body.Confirmed)
	require.Equal(t, aliases, body.Aliases)
}

func TestPromptHandlerCreateMalformedAndUnowned(t *testing.T) {
	userID, brandID := uuid.New(), uuid.New()
	commander := NewMockPromptCommander(t)
	querier := NewMockPromptQuerier(t)
	querier.EXPECT().List(mock.Anything, brandID,
		koidomain.PageParams{Page: 1, PageSize: 10}, "", (*bool)(nil),
	).Return(nil, koidomain.ErrNotFound)
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
		BaseEntity: koidomain.BaseEntity{
			ID: promptID,
		},
		BrandID:   brandID,
		Text:      "What is Koi?",
		Source:    koidomain.PromptSourceDrafted,
		Confirmed: true,
	}

	commander.EXPECT().Confirm(mock.Anything, brandID, promptID).Return(prompt, nil).Twice()
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
	commander.EXPECT().Create(mock.Anything, brandID, koidomain.CompetitorParams{
		Name:   "Acme",
		Domain: "acme.com",
	}).Return(&koidomain.Competitor{
		BrandID:   brandID,
		Name:      "Acme",
		Domain:    "acme.com",
		Source:    koidomain.CompetitorSourceManual,
		Confirmed: true,
	}, nil)
	ts := newListsTestServer(t, NewMockPromptQuerier(t), NewMockPromptCommander(t), NewMockCompetitorQuerier(t), commander)
	client := newCookieClient(t)
	brandSession(t, client, ts.URL, userID)

	resp := doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/brands/"+brandID.String()+"/competitors",
		`{"name":"Acme","domain":"acme.com","source":"observed","confirmed":false}`)

	var body koidomain.Competitor
	readJSON(t, resp, &body)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	require.Equal(t, koidomain.CompetitorSourceManual, body.Source)
	require.True(t, body.Confirmed)
}

func TestPromptHandlerPaginatesAndFilters(t *testing.T) {
	userID, brandID := uuid.New(), uuid.New()
	querier := NewMockPromptQuerier(t)
	querier.EXPECT().List(mock.Anything, brandID,
		koidomain.PageParams{Page: 2, PageSize: 5}, "best tools", mock.MatchedBy(func(confirmed *bool) bool {
			return confirmed != nil && !*confirmed
		}),
	).Return(&koidomain.Page[koidomain.Prompt]{
		Items:       []koidomain.Prompt{},
		TotalItems:  6,
		TotalPages:  2,
		CurrentPage: 2,
		HasPrev:     true,
	}, nil)
	ts := newListsTestServer(t, querier, NewMockPromptCommander(t), NewMockCompetitorQuerier(t), NewMockCompetitorCommander(t))
	client := newCookieClient(t)
	brandSession(t, client, ts.URL, userID)

	resp := doJSON(t, client, http.MethodGet, ts.URL+"/api/v1/brands/"+brandID.String()+"/prompts?page=2&pageSize=5&q=best+tools&confirmed=false", "")
	var page koidomain.Page[koidomain.Prompt]
	readJSON(t, resp, &page)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NotNil(t, page.Items)
	require.EqualValues(t, 6, page.TotalItems)
	require.EqualValues(t, 2, page.TotalPages)
	require.Equal(t, 2, page.CurrentPage)
	require.True(t, page.HasPrev)
	require.False(t, page.HasNext)
}

func TestPromptHandlerRejectsInvalidListFilters(t *testing.T) {
	userID, brandID := uuid.New(), uuid.New()
	ts := newListsTestServer(t, NewMockPromptQuerier(t), NewMockPromptCommander(t), NewMockCompetitorQuerier(t), NewMockCompetitorCommander(t))
	client := newCookieClient(t)
	brandSession(t, client, ts.URL, userID)

	for _, query := range []string{
		"page=0", "pageSize=101", "confirmed=maybe", "confirmed=true&confirmed=false", "q=one&q=two",
	} {
		resp := doJSON(t, client, http.MethodGet, ts.URL+"/api/v1/brands/"+brandID.String()+"/prompts?"+query, "")
		var body api.Envelope
		readJSON(t, resp, &body)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode, query)
		require.Equal(t, koidomain.CodeValidation, body.Error.Code)
	}
}

func TestPromptHandlerGetsOneAndConfirmsAllWithoutBody(t *testing.T) {
	userID, brandID, promptID := uuid.New(), uuid.New(), uuid.New()
	querier := NewMockPromptQuerier(t)
	commander := NewMockPromptCommander(t)
	prompt := &koidomain.Prompt{
		BaseEntity: koidomain.BaseEntity{
			ID: promptID,
		},
		BrandID: brandID,
	}
	querier.EXPECT().Get(mock.Anything, brandID, promptID).Return(prompt, nil)
	commander.EXPECT().ConfirmAll(mock.Anything, brandID).Return(nil)
	ts := newListsTestServer(t, querier, commander, NewMockCompetitorQuerier(t), NewMockCompetitorCommander(t))
	client := newCookieClient(t)
	brandSession(t, client, ts.URL, userID)

	resp := doJSON(t, client, http.MethodGet, ts.URL+"/api/v1/brands/"+brandID.String()+"/prompts/"+promptID.String(), "")
	var body koidomain.Prompt
	readJSON(t, resp, &body)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, promptID, body.ID)

	resp = doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/brands/"+brandID.String()+"/prompts/confirm", "")
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	require.Zero(t, resp.ContentLength)
	resp.Body.Close()
}

func TestCompetitorHandlerPaginatesAndConfirmsAllWithoutBody(t *testing.T) {
	userID, brandID := uuid.New(), uuid.New()
	querier := NewMockCompetitorQuerier(t)
	commander := NewMockCompetitorCommander(t)
	querier.EXPECT().List(mock.Anything, brandID,
		koidomain.PageParams{Page: 1, PageSize: 1}, "acme", mock.MatchedBy(func(confirmed *bool) bool {
			return confirmed != nil && *confirmed
		}),
	).Return(&koidomain.Page[koidomain.Competitor]{
		Items:       []koidomain.Competitor{},
		CurrentPage: 1,
	}, nil)
	commander.EXPECT().ConfirmAll(mock.Anything, brandID).Return(nil)
	ts := newListsTestServer(t, NewMockPromptQuerier(t), NewMockPromptCommander(t), querier, commander)
	client := newCookieClient(t)
	brandSession(t, client, ts.URL, userID)

	resp := doJSON(t, client, http.MethodGet, ts.URL+"/api/v1/brands/"+brandID.String()+"/competitors?pageSize=1&q=acme&confirmed=true", "")
	var page koidomain.Page[koidomain.Competitor]
	readJSON(t, resp, &page)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NotNil(t, page.Items)

	resp = doJSON(t, client, http.MethodPost, ts.URL+"/api/v1/brands/"+brandID.String()+"/competitors/confirm", "")
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	require.Zero(t, resp.ContentLength)
	resp.Body.Close()
}
