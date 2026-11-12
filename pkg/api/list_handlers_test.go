package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/api/response"
	"github.com/koihq/koi/pkg/domain"
)

func newListsTestServer(
	t *testing.T,
	pq domain.PromptQuerier,
	pc domain.PromptCommander,
	cq domain.CompetitorQuerier,
	cc domain.CompetitorCommander,
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
	r.With(Authorizer(sess)).Route("/api/v1/prompts", NewPromptHandler(pq, pc).Routes())
	r.With(Authorizer(sess)).Route("/api/v1/competitors", NewCompetitorHandler(cq, cc).Routes())
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
		resp := doJSON(t, client, request.method, ts.URL+"/api/v1"+request.path, request.body, brandID)
		var body response.Envelope
		readJSON(t, resp, &body)
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		require.Equal(t, domain.CodeUnauthorized, body.Error.Code)
	}
}

func TestCompetitorPatchCannotConfirmOrChangeSource(t *testing.T) {
	userID, brandID, competitorID := uuid.New(), uuid.New(), uuid.New()
	commander := NewMockCompetitorCommander(t)
	name := "Acme Analytics"
	aliases := []string{"Acme"}
	urls := []string{"https://acme.tools"}
	commander.EXPECT().Update(mock.Anything, competitorID, domain.UpdateCompetitorParams{
		Name: &name, Aliases: &aliases, AdditionalURLs: &urls,
	}).Return(&domain.Competitor{
		BaseEntity: domain.BaseEntity{
			ID: competitorID,
		},
		BrandID: brandID, Name: name, Source: domain.CompetitorSourceObserved,
		Aliases: aliases, AdditionalURLs: urls,
	}, nil)
	ts := newListsTestServer(t, NewMockPromptQuerier(t), NewMockPromptCommander(t), NewMockCompetitorQuerier(t), commander)
	client := newCookieClient(t)
	brandSession(t, client, ts.URL, userID)

	resp := doJSON(t, client, http.MethodPatch, ts.URL+"/api/v1"+"/competitors/"+competitorID.String(),
		`{"name":"Acme Analytics","aliases":["Acme"],"additionalUrls":["https://acme.tools"],"source":"manual","confirmed":true}`, brandID)
	var body CompetitorRes
	readJSON(t, resp, &body)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, domain.CompetitorSourceObserved, body.Source)
	require.False(t, body.Confirmed)
	require.Equal(t, aliases, body.Aliases)
}

func TestPromptHandlerCreateMalformedAndUnowned(t *testing.T) {
	userID, brandID := uuid.New(), uuid.New()
	commander := NewMockPromptCommander(t)
	querier := NewMockPromptQuerier(t)
	querier.EXPECT().ListWithVisibility(mock.Anything, domain.PromptFilter{
		BrandFilter: domain.BrandFilter{
			BrandID: brandID,
		},
		Sort: domain.PromptSortOriginal,
		Days: 30,
	}, domain.PageParams{
		Page:     1,
		PageSize: 10,
	}).Return(nil, domain.ErrNotFound)
	ts := newListsTestServer(t, querier, commander, NewMockCompetitorQuerier(t), NewMockCompetitorCommander(t))
	client := newCookieClient(t)
	brandSession(t, client, ts.URL, userID)

	resp := doJSON(t, client, http.MethodPost, ts.URL+"/api/v1"+"/prompts", `{`, brandID)
	var malformed response.Envelope
	readJSON(t, resp, &malformed)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	require.Equal(t, domain.CodeInvalidJSON, malformed.Error.Code)

	resp = doJSON(t, client, http.MethodGet, ts.URL+"/api/v1"+"/prompts", "", brandID)
	var missing response.Envelope
	readJSON(t, resp, &missing)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	require.Equal(t, domain.CodeNotFound, missing.Error.Code)
}

func TestPromptHandlerConfirmationIsIdempotent(t *testing.T) {
	userID, brandID, promptID := uuid.New(), uuid.New(), uuid.New()
	commander := NewMockPromptCommander(t)

	prompt := &domain.Prompt{
		BaseEntity: domain.BaseEntity{
			ID: promptID,
		},
		BrandID:   brandID,
		Text:      "What is Koi?",
		Source:    domain.PromptSourceDrafted,
		Confirmed: true,
	}

	commander.EXPECT().Confirm(mock.Anything, promptID).Return(prompt, nil).Twice()
	ts := newListsTestServer(t, NewMockPromptQuerier(t), commander, NewMockCompetitorQuerier(t), NewMockCompetitorCommander(t))
	client := newCookieClient(t)
	brandSession(t, client, ts.URL, userID)

	for range 2 {
		resp := doJSON(t, client, http.MethodPost, ts.URL+"/api/v1"+"/prompts/"+promptID.String()+"/confirm", "", brandID)
		var body PromptRes
		readJSON(t, resp, &body)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.True(t, body.Confirmed)
	}
}

func TestCompetitorHandlerManualRequestCannotChangeProvenance(t *testing.T) {
	userID, brandID := uuid.New(), uuid.New()
	commander := NewMockCompetitorCommander(t)
	commander.EXPECT().Create(mock.Anything, brandID, domain.CreateCompetitorParams{
		Name:   "Acme",
		Domain: "acme.com",
	}).Return(&domain.Competitor{
		BrandID:        brandID,
		Name:           "Acme",
		Domain:         "acme.com",
		Aliases:        []string{},
		AdditionalURLs: []string{},
		Source:         domain.CompetitorSourceManual,
		Confirmed:      true,
	}, nil)
	ts := newListsTestServer(t, NewMockPromptQuerier(t), NewMockPromptCommander(t), NewMockCompetitorQuerier(t), commander)
	client := newCookieClient(t)
	brandSession(t, client, ts.URL, userID)

	resp := doJSON(t, client, http.MethodPost, ts.URL+"/api/v1"+"/competitors",
		`{"name":"Acme","domain":"acme.com","source":"observed","confirmed":false}`, brandID)

	var body CompetitorRes
	readJSON(t, resp, &body)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	require.Equal(t, domain.CompetitorSourceManual, body.Source)
	require.True(t, body.Confirmed)
}

func TestPromptHandlerPaginatesAndFilters(t *testing.T) {
	userID, brandID := uuid.New(), uuid.New()
	querier := NewMockPromptQuerier(t)
	querier.EXPECT().ListWithVisibility(mock.Anything, mock.MatchedBy(func(filter domain.PromptFilter) bool {
		return filter.BrandID == brandID &&
			filter.Query == "best tools" && filter.Confirmed != nil && !*filter.Confirmed &&
			filter.Sort == domain.PromptSortOriginal && filter.Days == 30
	}), domain.PageParams{
		Page:     2,
		PageSize: 5,
	}).Return(&domain.Page[domain.Prompt]{
		Items:       []domain.Prompt{},
		TotalItems:  6,
		TotalPages:  2,
		CurrentPage: 2,
		HasPrev:     true,
	}, nil)
	ts := newListsTestServer(t, querier, NewMockPromptCommander(t), NewMockCompetitorQuerier(t), NewMockCompetitorCommander(t))
	client := newCookieClient(t)
	brandSession(t, client, ts.URL, userID)

	resp := doJSON(t, client, http.MethodGet, ts.URL+"/api/v1"+"/prompts?page=2&pageSize=5&q=best+tools&confirmed=false", "", brandID)
	var page domain.Page[domain.Prompt]
	readJSON(t, resp, &page)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NotNil(t, page.Items)
	require.EqualValues(t, 6, page.TotalItems)
	require.EqualValues(t, 2, page.TotalPages)
	require.Equal(t, 2, page.CurrentPage)
	require.True(t, page.HasPrev)
	require.False(t, page.HasNext)
}

func TestPromptHandlerFiltersStoppedAndReportsResults(t *testing.T) {
	userID, brandID, promptID := uuid.New(), uuid.New(), uuid.New()
	querier := NewMockPromptQuerier(t)
	querier.EXPECT().ListWithVisibility(mock.Anything, mock.MatchedBy(func(filter domain.PromptFilter) bool {
		return filter.BrandID == brandID && filter.Inactive != nil && *filter.Inactive && filter.Confirmed == nil
	}), mock.Anything).Return(&domain.Page[domain.Prompt]{
		Items: []domain.Prompt{{
			BaseEntity: domain.BaseEntity{ID: promptID},
			BrandID:    brandID,
			Text:       "Stopped?",
			Source:     domain.PromptSourceManual,
			Confirmed:  true,
			Inactive:   true,
			HasResults: true,
		}},
		TotalItems:  1,
		TotalPages:  1,
		CurrentPage: 1,
	}, nil)
	ts := newListsTestServer(t, querier, NewMockPromptCommander(t), NewMockCompetitorQuerier(t), NewMockCompetitorCommander(t))
	client := newCookieClient(t)
	brandSession(t, client, ts.URL, userID)

	resp := doJSON(t, client, http.MethodGet, ts.URL+"/api/v1/prompts?inactive=true", "", brandID)
	var page struct {
		Items []PromptRes `json:"items"`
	}
	readJSON(t, resp, &page)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Len(t, page.Items, 1)
	require.True(t, page.Items[0].Inactive)
	require.True(t, page.Items[0].HasResults)
}

func TestPromptHandlerDeleteWithResultsIsAConflict(t *testing.T) {
	userID, promptID := uuid.New(), uuid.New()
	commander := NewMockPromptCommander(t)
	commander.EXPECT().Delete(mock.Anything, promptID).Return(domain.ErrResourceConflict)
	ts := newListsTestServer(t, NewMockPromptQuerier(t), commander, NewMockCompetitorQuerier(t), NewMockCompetitorCommander(t))
	client := newCookieClient(t)
	brandSession(t, client, ts.URL, userID)

	resp := doJSON(t, client, http.MethodDelete, ts.URL+"/api/v1/prompts/"+promptID.String(), "")
	var body response.Envelope
	readJSON(t, resp, &body)
	require.Equal(t, http.StatusConflict, resp.StatusCode)
	require.Equal(t, domain.CodeResourceConflict, body.Error.Code)
}

func TestPromptHandlerRejectsInvalidListFilters(t *testing.T) {
	userID, brandID := uuid.New(), uuid.New()
	ts := newListsTestServer(t, NewMockPromptQuerier(t), NewMockPromptCommander(t), NewMockCompetitorQuerier(t), NewMockCompetitorCommander(t))
	client := newCookieClient(t)
	brandSession(t, client, ts.URL, userID)

	for _, query := range []string{
		"page=0", "pageSize=101", "confirmed=maybe", "confirmed=true&confirmed=false", "q=one&q=two",
		"inactive=maybe", "inactive=true&inactive=false",
	} {
		resp := doJSON(t, client, http.MethodGet, ts.URL+"/api/v1"+"/prompts?"+query, "", brandID)
		var body response.Envelope
		readJSON(t, resp, &body)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode, query)
		require.Equal(t, domain.CodeValidation, body.Error.Code)
	}
}

func TestPromptHandlerGetsOne(t *testing.T) {
	userID, brandID, promptID := uuid.New(), uuid.New(), uuid.New()
	querier := NewMockPromptQuerier(t)
	commander := NewMockPromptCommander(t)
	prompt := &domain.Prompt{
		BaseEntity: domain.BaseEntity{
			ID: promptID,
		},
		BrandID: brandID,
		Source:  domain.PromptSourceManual,
	}
	querier.EXPECT().Get(mock.Anything, promptID).Return(prompt, nil)
	ts := newListsTestServer(t, querier, commander, NewMockCompetitorQuerier(t), NewMockCompetitorCommander(t))
	client := newCookieClient(t)
	brandSession(t, client, ts.URL, userID)

	resp := doJSON(t, client, http.MethodGet, ts.URL+"/api/v1"+"/prompts/"+promptID.String(), "", brandID)
	var body PromptRes
	readJSON(t, resp, &body)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, promptID, body.ID)
}

func TestCompetitorHandlerPaginates(t *testing.T) {
	userID, brandID := uuid.New(), uuid.New()
	querier := NewMockCompetitorQuerier(t)
	commander := NewMockCompetitorCommander(t)
	querier.EXPECT().List(mock.Anything, mock.MatchedBy(func(filter domain.CompetitorFilter) bool {
		return filter.BrandID == brandID && filter.Query == "acme" && filter.Confirmed != nil && *filter.Confirmed
	}), domain.PageParams{
		Page:     1,
		PageSize: 1,
	}).Return(&domain.Page[domain.Competitor]{
		Items:       []domain.Competitor{},
		CurrentPage: 1,
	}, nil)
	ts := newListsTestServer(t, NewMockPromptQuerier(t), NewMockPromptCommander(t), querier, commander)
	client := newCookieClient(t)
	brandSession(t, client, ts.URL, userID)

	resp := doJSON(t, client, http.MethodGet, ts.URL+"/api/v1"+"/competitors?pageSize=1&q=acme&confirmed=true", "", brandID)
	var page domain.Page[domain.Competitor]
	readJSON(t, resp, &page)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NotNil(t, page.Items)
}

func TestCompetitorSearchReturnsProviderMatchesOrSaysItIsOff(t *testing.T) {
	userID, brandID := uuid.New(), uuid.New()
	querier := NewMockCompetitorQuerier(t)
	ts := newListsTestServer(t, NewMockPromptQuerier(t), NewMockPromptCommander(t), querier, NewMockCompetitorCommander(t))
	client := newCookieClient(t)
	brandSession(t, client, ts.URL, userID)
	search := ts.URL + "/api/v1/competitors/search?q="

	querier.EXPECT().Search(identityIs(userID), brandID, "stri").Return([]domain.CompanySearchResult{
		{Name: "Stripe", Domain: "stripe.com"},
	}, nil).Once()
	resp := doJSON(t, client, http.MethodGet, search+"stri", "", brandID)
	var results []domain.CompanySearchResult
	readJSON(t, resp, &results)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, []domain.CompanySearchResult{{Name: "Stripe", Domain: "stripe.com"}}, results)

	// Without the optional Logo.dev key, manual competitors still work and search says it is off.
	querier.EXPECT().Search(identityIs(userID), brandID, "acme").Return(nil, domain.ErrFeatureUnavailable).Once()
	resp = doJSON(t, client, http.MethodGet, search+"acme", "", brandID)
	require.Contains(t, readBody(t, resp), `"code":"feature_unavailable"`)
	require.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}
