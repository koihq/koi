package koiapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/koiapi/api"
	"github.com/koihq/koi/pkg/koidomain"
)

func newVisibilityTestServer(t *testing.T, q koidomain.VisibilityQuerier) *httptest.Server {
	t.Helper()
	sess := scs.New()
	sess.Cookie.Name = "koi_session"
	r := chi.NewRouter()
	r.Use(sess.LoadAndSave)
	r.Get("/test-session/{id}", func(w http.ResponseWriter, r *http.Request) {
		sess.Put(r.Context(), "user_id", chi.URLParam(r, "id"))
		w.WriteHeader(http.StatusNoContent)
	})
	handler := NewVisibilityHandler(q, sess)
	r.Route("/api/v1/brands/{brandID}/runs", handler.Routes())
	r.With(Authorizer(sess)).Get("/api/v1/brands/{brandID}/prompt-history", handler.ListHistoricalPrompts)
	r.With(Authorizer(sess)).Get("/api/v1/brands/{brandID}/prompt-history/{promptID}/runs", handler.ListPromptRuns)
	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)

	return ts
}

func TestVisibilityHandlerPromptHistory(t *testing.T) {
	brandID, promptID, runID, userID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	q := NewMockVisibilityQuerier(t)
	ts := newVisibilityTestServer(t, q)
	client := newCookieClient(t)
	base := ts.URL + "/api/v1/brands/" + brandID.String() + "/prompt-history"

	response := doJSON(t, client, http.MethodGet, base, "")
	require.Equal(t, http.StatusUnauthorized, response.StatusCode)
	response.Body.Close()

	brandSession(t, client, ts.URL, userID)
	page := koidomain.PageParams{Page: 1, PageSize: 10}
	q.EXPECT().ListHistoricalPrompts(identityIs(userID), brandID, koidomain.PromptListParams{
		Page: page,
		Sort: koidomain.PromptSortOriginal,
		Days: 30,
	}).Return(
		&koidomain.Page[koidomain.HistoricalPrompt]{
			Items: []koidomain.HistoricalPrompt{{
				PromptID:   promptID,
				PromptText: "Saved question?",
				LastSeenAt: time.Date(2026, 10, 5, 10, 30, 0, 0, time.UTC),
			}},
			TotalItems:  1,
			TotalPages:  1,
			CurrentPage: 1,
		}, nil,
	)
	response = doJSON(t, client, http.MethodGet, base+"?pageSize=10", "")
	var snapshots map[string]any
	readJSON(t, response, &snapshots)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.EqualValues(t, 1, snapshots["totalItems"])
	require.Equal(t, promptID.String(), snapshots["items"].([]any)[0].(map[string]any)["prompt_id"])

	run := koidomain.VisibilityRun{
		BaseEntity:    koidomain.BaseEntity{ID: runID},
		BrandID:       brandID,
		Engines:       koidomain.VisibilityEngines(),
		State:         koidomain.VisibilitySucceeded,
		BrandSnapshot: koidomain.BrandSnapshot{Name: "private snapshot"},
	}
	q.EXPECT().ListPromptRuns(identityIs(userID), brandID, promptID, page).Return(
		&koidomain.Page[koidomain.PromptHistoryRun]{
			Items: []koidomain.PromptHistoryRun{{
				Run: run,
				Results: []koidomain.VisibilityResult{{
					RunID:       runID,
					PromptID:    promptID,
					PromptText:  "Saved question?",
					Engine:      koidomain.EngineChatGPT,
					Collector:   koidomain.CollectorCloro,
					Sources:     []string{"https://example.com"},
					Appearances: []koidomain.CompetitorAppearance{},
				}},
			}},
			TotalItems:  1,
			TotalPages:  1,
			CurrentPage: 1,
		}, nil,
	)
	response = doJSON(t, client, http.MethodGet, base+"/"+promptID.String()+"/runs?pageSize=10", "")
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
	base := ts.URL + "/api/v1/brands/" + brandID.String() + "/runs"

	for _, path := range []string{base, base + "/" + runID.String(), base + "/" + runID.String() + "/results"} {
		resp := doJSON(t, client, http.MethodGet, path, "")
		var body api.Envelope
		readJSON(t, resp, &body)
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	}

	brandSession(t, client, ts.URL, userID)
	run := &koidomain.VisibilityRun{
		BaseEntity: koidomain.BaseEntity{
			ID:        runID,
			CreatedAt: time.Date(2026, 10, 5, 10, 30, 0, 0, time.UTC),
		},
		BrandID:      brandID,
		Engines:      koidomain.VisibilityEngines(),
		State:        koidomain.VisibilityQueued,
		TotalResults: 8,
		BrandSnapshot: koidomain.BrandSnapshot{
			Name: "private snapshot",
		},
	}

	q.EXPECT().Get(identityIs(userID), brandID, runID).Return(run, nil).Once()
	resp := doJSON(t, client, http.MethodGet, base+"/"+runID.String(), "")
	var saved map[string]any
	readJSON(t, resp, &saved)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, runID.String(), saved["id"])
	require.Equal(t, float64(8), saved["total_results"])
	require.NotContains(t, saved, "brand_snapshot")
	require.NotContains(t, saved, "river_job_id")
	require.NotContains(t, saved, "attempt")
	q.EXPECT().Get(identityIs(userID), brandID, runID).Return(nil, koidomain.ErrNotFound)
	resp = doJSON(t, client, http.MethodGet, base+"/"+runID.String(), "")
	var body api.Envelope
	readJSON(t, resp, &body)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestVisibilityHandlerPagesAndPendingResults(t *testing.T) {
	brandID, runID, userID := uuid.New(), uuid.New(), uuid.New()
	q := NewMockVisibilityQuerier(t)
	ts := newVisibilityTestServer(t, q)
	client := newCookieClient(t)
	brandSession(t, client, ts.URL, userID)
	base := ts.URL + "/api/v1/brands/" + brandID.String() + "/runs"
	q.EXPECT().List(identityIs(userID), brandID, koidomain.PageParams{Page: 1, PageSize: 10}).Return(
		&koidomain.Page[koidomain.VisibilityRun]{
			Items:       []koidomain.VisibilityRun{},
			CurrentPage: 1,
		}, nil)
	resp := doJSON(t, client, http.MethodGet, base, "")
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
	q.EXPECT().ListResults(identityIs(userID), brandID, runID, koidomain.PageParams{Page: 2, PageSize: 40}).Return(
		&koidomain.Page[koidomain.VisibilityResult]{
			Items: []koidomain.VisibilityResult{
				{
					RunID:      runID,
					PromptID:   uuid.New(),
					PromptText: "Best analytics?",
					Engine:     koidomain.EngineChatGPT,
					Collector:  koidomain.CollectorCloro,
					Sources:    []string{},
				},
			},
			CurrentPage: 2,
			TotalItems:  44,
			TotalPages:  2,
			HasPrev:     true,
		}, nil)
	resp = doJSON(t, client, http.MethodGet, base+"/"+runID.String()+"/results?page=2&pageSize=40", "")
	var page koidomain.Page[map[string]any]
	readJSON(t, resp, &page)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.EqualValues(t, 44, page.TotalItems)
	require.Nil(t, page.Items[0]["outcome"])
	require.Nil(t, page.Items[0]["fetched_at"])
	require.Equal(t, []any{}, page.Items[0]["sources"])

	for _, path := range []string{base, base + "/" + runID.String() + "/results"} {
		for _, query := range []string{"page=0", "pageSize=101", "page=bad", "page=1&page=2"} {
			resp = doJSON(t, client, http.MethodGet, path+"?"+query, "")
			var body api.Envelope
			readJSON(t, resp, &body)
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		}
	}

	resp = doJSON(t, client, http.MethodGet, base+"/bad-id/results", "")
	var missing api.Envelope
	readJSON(t, resp, &missing)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}
