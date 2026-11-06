package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/api/middlewares"
	"github.com/koihq/koi/pkg/api/response"
	"github.com/koihq/koi/pkg/domain"
)

func newVisibilityTestServer(t *testing.T, q domain.VisibilityQuerier) *httptest.Server {
	t.Helper()

	return newTrackingTestServer(t, q, NewMockTrackingCommander(t))
}

func newTrackingTestServer(t *testing.T, q domain.VisibilityQuerier, tracker domain.TrackingCommander) *httptest.Server {
	t.Helper()
	sess := scs.New()
	sess.Cookie.Name = "koi_session"
	r := chi.NewRouter()
	r.Use(sess.LoadAndSave)
	r.Get("/test-session/{id}", func(w http.ResponseWriter, r *http.Request) {
		sess.Put(r.Context(), "user_id", chi.URLParam(r, "id"))
		w.WriteHeader(http.StatusNoContent)
	})
	handler := NewVisibilityHandler(q, tracker, nil)
	r.With(Authorizer(sess)).Route("/api/v1/runs", handler.Routes())
	r.With(Authorizer(sess), middlewares.DecodeBody[BrandRequestScope]()).Post("/api/v1/tracking/start", handler.StartTracking())
	r.With(Authorizer(sess)).Route("/api/v1/prompt-history", handler.HistoryRoutes())
	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)

	return ts
}

func TestVisibilityHandlerPromptHistory(t *testing.T) {
	brandID, promptID, runID, userID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	q := NewMockVisibilityQuerier(t)
	ts := newVisibilityTestServer(t, q)
	client := newCookieClient(t)
	base := ts.URL + "/api/v1" + "/prompt-history"

	response := doJSON(t, client, http.MethodGet, base, "", brandID)
	require.Equal(t, http.StatusUnauthorized, response.StatusCode)
	response.Body.Close()

	brandSession(t, client, ts.URL, userID)
	removed := doJSON(t, client, http.MethodGet, base+"?pageSize=10", "", brandID)
	require.NotEqual(t, http.StatusOK, removed.StatusCode, "the deleted-prompts list is gone")
	require.NoError(t, removed.Body.Close())

	page := domain.PageParams{Page: 1, PageSize: 10}

	run := domain.VisibilityRun{
		BaseEntity:    domain.BaseEntity{ID: runID},
		BrandID:       brandID,
		Engines:       domain.VisibilityEngines(),
		State:         domain.VisibilitySucceeded,
		BrandSnapshot: domain.BrandSnapshot{Name: "private snapshot"},
	}
	q.EXPECT().ListPromptRuns(identityIs(userID), brandID, promptID, page).Return(
		&domain.Page[domain.PromptHistoryRun]{
			Items: []domain.PromptHistoryRun{{
				Run: run,
				Results: []domain.VisibilityResult{{
					RunID:       runID,
					PromptID:    promptID,
					PromptText:  "Saved question?",
					Engine:      domain.EngineChatGPT,
					Collector:   domain.CollectorCloro,
					Sources:     []string{"https://example.com"},
					Appearances: []domain.CompetitorAppearance{},
				}},
			}},
			TotalItems:  1,
			TotalPages:  1,
			CurrentPage: 1,
		}, nil,
	)
	response = doJSON(t, client, http.MethodGet, base+"/"+promptID.String()+"/runs?pageSize=10", "", brandID)
	var history map[string]any
	readJSON(t, response, &history)
	require.Equal(t, http.StatusOK, response.StatusCode)
	item := history["items"].([]any)[0].(map[string]any)
	publicRun := item["run"].(map[string]any)
	require.Equal(t, runID.String(), publicRun["id"])
	require.NotContains(t, publicRun, "brand_snapshot")
	require.NotContains(t, publicRun, "river_job_id")
	require.Len(t, item["results"], 1)
}

func TestVisibilityHandlerAuthReadinessAndPublicStatus(t *testing.T) {
	brandID, runID, userID := uuid.New(), uuid.New(), uuid.New()
	q := NewMockVisibilityQuerier(t)
	ts := newVisibilityTestServer(t, q)
	client := newCookieClient(t)
	base := ts.URL + "/api/v1" + "/runs"

	for _, path := range []string{base, base + "/" + runID.String(), base + "/" + runID.String() + "/results"} {
		resp := doJSON(t, client, http.MethodGet, path, "", brandID)
		var body response.Envelope
		readJSON(t, resp, &body)
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	}

	brandSession(t, client, ts.URL, userID)
	run := &domain.VisibilityRun{
		BaseEntity: domain.BaseEntity{
			ID:        runID,
			CreatedAt: time.Date(2026, 10, 5, 10, 30, 0, 0, time.UTC),
		},
		BrandID:      brandID,
		Engines:      domain.VisibilityEngines(),
		State:        domain.VisibilityQueued,
		TotalResults: 8,
		BrandSnapshot: domain.BrandSnapshot{
			Name: "private snapshot",
		},
	}

	q.EXPECT().Get(identityIs(userID), runID).Return(run, nil).Once()
	resp := doJSON(t, client, http.MethodGet, base+"/"+runID.String(), "", brandID)
	var saved map[string]any
	readJSON(t, resp, &saved)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, runID.String(), saved["id"])
	require.Equal(t, float64(8), saved["totalResults"])
	require.NotContains(t, saved, "brand_snapshot")
	require.NotContains(t, saved, "river_job_id")
	require.NotContains(t, saved, "attempt")
	q.EXPECT().Get(identityIs(userID), runID).Return(nil, domain.ErrNotFound)
	resp = doJSON(t, client, http.MethodGet, base+"/"+runID.String(), "", brandID)
	var body response.Envelope
	readJSON(t, resp, &body)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestVisibilityHandlerPagesAndPendingResults(t *testing.T) {
	brandID, runID, userID := uuid.New(), uuid.New(), uuid.New()
	q := NewMockVisibilityQuerier(t)
	ts := newVisibilityTestServer(t, q)
	client := newCookieClient(t)
	brandSession(t, client, ts.URL, userID)
	base := ts.URL + "/api/v1" + "/runs"
	q.EXPECT().List(identityIs(userID), domain.BrandFilter{BrandID: brandID}, domain.PageParams{Page: 1, PageSize: 10}).Return(
		&domain.Page[domain.VisibilityRun]{
			Items:       []domain.VisibilityRun{},
			CurrentPage: 1,
		}, nil)
	resp := doJSON(t, client, http.MethodGet, base, "", brandID)
	var empty map[string]any
	readJSON(t, resp, &empty)
	require.Equal(t, map[string]any{
		"items":       []any{},
		"totalItems":  float64(0),
		"totalPages":  float64(0),
		"currentPage": float64(1),
		"hasNext":     false,
		"hasPrev":     false,
	}, empty)
	q.EXPECT().ListResults(identityIs(userID), runID, domain.PageParams{Page: 2, PageSize: 40}).Return(
		&domain.Page[domain.VisibilityResult]{
			Items: []domain.VisibilityResult{
				{
					RunID:      runID,
					PromptID:   uuid.New(),
					PromptText: "Best analytics?",
					Engine:     domain.EngineChatGPT,
					Collector:  domain.CollectorCloro,
					Sources:    []string{},
				},
			},
			CurrentPage: 2,
			TotalItems:  44,
			TotalPages:  2,
			HasPrev:     true,
		}, nil)
	resp = doJSON(t, client, http.MethodGet, base+"/"+runID.String()+"/results?page=2&pageSize=40", "", brandID)
	var page domain.Page[map[string]any]
	readJSON(t, resp, &page)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.EqualValues(t, 44, page.TotalItems)
	require.Nil(t, page.Items[0]["outcome"])
	require.Nil(t, page.Items[0]["fetchedAt"])
	require.Equal(t, []any{}, page.Items[0]["sources"])

	for _, path := range []string{base, base + "/" + runID.String() + "/results"} {
		for _, query := range []string{"page=0", "pageSize=101", "page=bad", "page=1&page=2"} {
			resp = doJSON(t, client, http.MethodGet, path+"?"+query, "", brandID)
			var body response.Envelope
			readJSON(t, resp, &body)
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		}
	}

	resp = doJSON(t, client, http.MethodGet, base+"/bad-id/results", "", brandID)
	var missing response.Envelope
	readJSON(t, resp, &missing)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestVisibilityHandlerStartTracking(t *testing.T) {
	brandID, userID := uuid.New(), uuid.New()
	tracker := NewMockTrackingCommander(t)
	ts := newTrackingTestServer(t, NewMockVisibilityQuerier(t), tracker)
	client := newCookieClient(t)
	base := ts.URL + "/api/v1/tracking/start"

	body := `{"brandId":"` + brandID.String() + `"}`

	resp := doJSON(t, client, http.MethodPost, base, body)
	var unauthorized response.Envelope
	readJSON(t, resp, &unauthorized)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	brandSession(t, client, ts.URL, userID)

	resp = doJSON(t, client, http.MethodPost, base, `{}`)
	var missing response.Envelope
	readJSON(t, resp, &missing)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	require.Equal(t, domain.CodeValidation, missing.Error.Code)

	tracker.EXPECT().Start(identityIs(userID), brandID).Return(nil).Once()
	resp = doJSON(t, client, http.MethodPost, base, body)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()

	tracker.EXPECT().Start(identityIs(userID), brandID).Return(domain.Validation(domain.FieldError{Field: "prompts", Code: "empty"})).Once()
	resp = doJSON(t, client, http.MethodPost, base, body)
	var invalid response.Envelope
	readJSON(t, resp, &invalid)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	require.Equal(t, domain.CodeValidation, invalid.Error.Code)
}
