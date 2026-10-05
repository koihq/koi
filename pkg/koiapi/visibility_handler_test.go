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

func newVisibilityTestServer(t *testing.T, q koidomain.VisibilityQuerier, c koidomain.VisibilityCommander) *httptest.Server {
	t.Helper()
	sess := scs.New()
	sess.Cookie.Name = "koi_session"
	r := chi.NewRouter()
	r.Use(sess.LoadAndSave)
	r.Get("/test-session/{id}", func(w http.ResponseWriter, r *http.Request) {
		sess.Put(r.Context(), "user_id", chi.URLParam(r, "id"))
		w.WriteHeader(http.StatusNoContent)
	})
	r.Route("/api/v1/brands/{brandID}/runs", NewVisibilityHandler(q, c, sess).Routes())
	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)
	return ts
}

func TestVisibilityHandlerAuthReadinessAndPublicStatus(t *testing.T) {
	brandID, runID, userID := uuid.New(), uuid.New(), uuid.New()
	q, c := NewMockVisibilityQuerier(t), NewMockVisibilityCommander(t)
	ts := newVisibilityTestServer(t, q, c)
	client := newCookieClient(t)
	base := ts.URL + "/api/v1/brands/" + brandID.String() + "/runs"
	for _, path := range []string{base, base + "/" + runID.String(), base + "/" + runID.String() + "/results"} {
		resp := doJSON(t, client, http.MethodGet, path, "")
		var body api.Envelope
		readJSON(t, resp, &body)
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	}
	resp := doJSON(t, client, http.MethodPost, base, "")
	var body api.Envelope
	readJSON(t, resp, &body)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	brandSession(t, client, ts.URL, userID)
	fields := []koidomain.FieldError{{Field: "prompts", Code: "unconfirmed"}}
	c.EXPECT().Create(identityIs(userID), brandID).Return(nil, &koidomain.Error{Code: koidomain.CodeRunNotReady, Fields: fields}).Once()
	resp = doJSON(t, client, http.MethodPost, base, "")
	readJSON(t, resp, &body)
	require.Equal(t, http.StatusConflict, resp.StatusCode)
	require.Equal(t, koidomain.CodeRunNotReady, body.Error.Code)
	require.Equal(t, fields, body.Error.Fields)
	run := &koidomain.VisibilityRun{
		BaseEntity: koidomain.BaseEntity{ID: runID, CreatedAt: time.Date(2026, 10, 5, 10, 30, 0, 0, time.UTC)},
		BrandID:    brandID, Engines: koidomain.VisibilityEngines(), State: koidomain.VisibilityQueued, TotalResults: 8,
		BrandSnapshot: koidomain.BrandSnapshot{Name: "private snapshot"},
	}
	c.EXPECT().Create(identityIs(userID), brandID).Return(run, nil).Once()
	resp = doJSON(t, client, http.MethodPost, base, "")
	var status map[string]any
	readJSON(t, resp, &status)
	require.Equal(t, http.StatusAccepted, resp.StatusCode)
	require.Equal(t, runID.String(), status["id"])
	require.Equal(t, brandID.String(), status["brand_id"])
	require.Equal(t, base[len(ts.URL):]+"/"+runID.String(), status["status_url"])
	require.Equal(t, float64(8), status["total_results"])
	require.Nil(t, status["started_at"])
	require.Nil(t, status["finished_at"])
	require.Nil(t, status["error_code"])
	require.NotContains(t, status, "brand_snapshot")
	require.NotContains(t, status, "river_job_id")
	require.NotContains(t, status, "attempt")
	q.EXPECT().Get(identityIs(userID), brandID, runID).Return(run, nil).Once()
	resp = doJSON(t, client, http.MethodGet, base+"/"+runID.String(), "")
	var saved map[string]any
	readJSON(t, resp, &saved)
	require.Equal(t, status, saved)
	q.EXPECT().Get(identityIs(userID), brandID, runID).Return(nil, koidomain.ErrNotFound)
	resp = doJSON(t, client, http.MethodGet, base+"/"+runID.String(), "")
	readJSON(t, resp, &body)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestVisibilityHandlerPagesAndPendingResults(t *testing.T) {
	brandID, runID, userID := uuid.New(), uuid.New(), uuid.New()
	q, c := NewMockVisibilityQuerier(t), NewMockVisibilityCommander(t)
	ts := newVisibilityTestServer(t, q, c)
	client := newCookieClient(t)
	brandSession(t, client, ts.URL, userID)
	base := ts.URL + "/api/v1/brands/" + brandID.String() + "/runs"
	q.EXPECT().List(identityIs(userID), brandID, koidomain.PageParams{Page: 1, PageSize: 10}).Return(
		&koidomain.Page[koidomain.VisibilityRun]{Items: []koidomain.VisibilityRun{}, CurrentPage: 1}, nil)
	resp := doJSON(t, client, http.MethodGet, base, "")
	var empty map[string]any
	readJSON(t, resp, &empty)
	require.Equal(t, map[string]any{
		"items": []any{}, "totalItems": float64(0), "totalPages": float64(0),
		"currentPage": float64(1), "hasNext": false, "hasPrev": false,
	}, empty)
	q.EXPECT().ListResults(identityIs(userID), brandID, runID, koidomain.PageParams{Page: 2, PageSize: 40}).Return(
		&koidomain.Page[koidomain.VisibilityResult]{
			Items: []koidomain.VisibilityResult{{
				RunID: runID, PromptID: uuid.New(), PromptText: "Best analytics?",
				Engine: koidomain.EngineChatGPT, Collector: koidomain.CollectorStub, Sources: []string{},
			}},
			CurrentPage: 2, TotalItems: 44, TotalPages: 2, HasPrev: true,
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
