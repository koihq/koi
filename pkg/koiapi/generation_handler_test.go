package koiapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	apiresponse "github.com/koihq/koi/pkg/koiapi/response"
	"github.com/koihq/koi/pkg/koidomain"
)

func newGenerationTestServer(t *testing.T, q koidomain.GenerationQuerier, c koidomain.GenerationCommander) *httptest.Server {
	t.Helper()
	sess := scs.New()
	sess.Cookie.Name = "koi_session"
	r := chi.NewRouter()
	r.Use(sess.LoadAndSave)
	r.Get("/test-session/{id}", func(w http.ResponseWriter, r *http.Request) {
		sess.Put(r.Context(), "user_id", chi.URLParam(r, "id"))
		w.WriteHeader(http.StatusNoContent)
	})
	r.With(Authorizer(sess)).Route("/api/v1/generations", NewGenerationHandler(q, c).Routes())
	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)

	return ts
}

func TestGenerationHandlerAuthValidationAndCreate(t *testing.T) {
	brandID, runID, userID := uuid.New(), uuid.New(), uuid.New()
	q, c := NewMockGenerationQuerier(t), NewMockGenerationCommander(t)
	ts := newGenerationTestServer(t, q, c)
	client := newCookieClient(t)
	base := ts.URL + "/api/v1" + "/generations"

	for _, method := range []string{http.MethodGet, http.MethodPost} {
		resp := doJSON(t, client, method, base, `{"mode":"initial"}`, brandID)
		var body apiresponse.Envelope
		readJSON(t, resp, &body)
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		require.Equal(t, koidomain.CodeUnauthorized, body.Error.Code)
	}

	brandSession(t, client, ts.URL, userID)

	for _, body := range []string{"{", `{"mode":123}`} {
		resp := doJSON(t, client, http.MethodPost, base, body, brandID)
		var result apiresponse.Envelope
		readJSON(t, resp, &result)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.Equal(t, koidomain.CodeInvalidJSON, result.Error.Code)
	}

	request, err := http.NewRequest(http.MethodPost, base, strings.NewReader(`{"brandId":"`+brandID.String()+`","mode":"initial"}`))
	require.NoError(t, err)
	request.Header.Set("Idempotency-Key", "bad key")
	resp, err := client.Do(request)
	require.NoError(t, err)
	var invalidKey apiresponse.Envelope
	readJSON(t, resp, &invalidKey)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	require.Equal(t, koidomain.CodeValidation, invalidKey.Error.Code)

	run := &koidomain.GenerationRun{
		BaseEntity: koidomain.BaseEntity{
			ID: runID,
		},
		Mode:  koidomain.GenerationInitial,
		State: koidomain.GenerationQueued,
	}
	c.EXPECT().Create(identityIs(userID), brandID, koidomain.GenerationInitial, mock.MatchedBy(func(key *string) bool {
		return key != nil && *key == "setup-1"
	})).Return(run, false, nil)
	request, err = http.NewRequest(http.MethodPost, base, strings.NewReader(`{"brandId":"`+brandID.String()+`","mode":"initial"}`))
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "setup-1")
	resp, err = client.Do(request)
	require.NoError(t, err)
	var accepted map[string]any
	readJSON(t, resp, &accepted)
	require.Equal(t, http.StatusAccepted, resp.StatusCode)
	require.Equal(t, map[string]any{
		"id":        runID.String(),
		"mode":      "initial",
		"state":     "queued",
		"statusUrl": "/api/v1" + "/generations/" + runID.String(),
	}, accepted)
	c.EXPECT().Create(identityIs(userID), brandID, koidomain.GenerationInitial, mock.Anything).Return(run, true, nil)
	resp = doJSON(t, client, http.MethodPost, base, `{"mode":"initial"}`, brandID)
	readJSON(t, resp, &accepted)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestGenerationHandlerStatusListAndConflicts(t *testing.T) {
	brandID, runID, userID := uuid.New(), uuid.New(), uuid.New()
	q, c := NewMockGenerationQuerier(t), NewMockGenerationCommander(t)
	ts := newGenerationTestServer(t, q, c)
	client := newCookieClient(t)
	brandSession(t, client, ts.URL, userID)
	base := ts.URL + "/api/v1" + "/generations"
	createdAt := time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC)

	run := &koidomain.GenerationRun{
		BaseEntity: koidomain.BaseEntity{
			ID:        runID,
			CreatedAt: createdAt,
		},
		BrandID:   brandID,
		Mode:      koidomain.GenerationInitial,
		State:     koidomain.GenerationQueued,
		ModelSlug: "secret-model",
		BrandSnapshot: koidomain.BrandSnapshot{
			Name: "private",
		},
	}

	q.EXPECT().Get(identityIs(userID), runID).Return(run, nil)
	resp := doJSON(t, client, http.MethodGet, base+"/"+runID.String(), "", brandID)
	var status map[string]any
	readJSON(t, resp, &status)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, map[string]any{
		"id":              runID.String(),
		"mode":            "initial",
		"state":           "queued",
		"attempt":         float64(0),
		"promptCount":     float64(0),
		"competitorCount": float64(0),
		"createdAt":       "2026-09-28T10:00:00Z",
	}, status)
	q.EXPECT().List(identityIs(userID), brandID, koidomain.PageParams{Page: 2, PageSize: 1}).Return(
		&koidomain.Page[koidomain.GenerationRun]{
			Items:       []koidomain.GenerationRun{*run},
			TotalItems:  2,
			TotalPages:  2,
			CurrentPage: 2,
			HasPrev:     true,
		}, nil)
	resp = doJSON(t, client, http.MethodGet, base+"?page=2&pageSize=1", "", brandID)
	var list koidomain.Page[map[string]any]
	readJSON(t, resp, &list)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, []map[string]any{status}, list.Items)
	require.EqualValues(t, 2, list.TotalItems)
	require.EqualValues(t, 2, list.TotalPages)
	require.Equal(t, 2, list.CurrentPage)
	require.True(t, list.HasPrev)
	require.False(t, list.HasNext)

	for _, query := range []string{"page=0", "page=abc", "page=1&page=2", "pageSize=0", "pageSize=101"} {
		resp = doJSON(t, client, http.MethodGet, base+"?"+query, "", brandID)
		var body apiresponse.Envelope
		readJSON(t, resp, &body)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.Equal(t, koidomain.CodeValidation, body.Error.Code)
	}

	resp = doJSON(t, client, http.MethodGet, base+"/bad-id", "", brandID)
	var missing apiresponse.Envelope
	readJSON(t, resp, &missing)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	c.EXPECT().Create(identityIs(userID), brandID, koidomain.GenerationInitial, mock.Anything).Return(nil, false,
		&koidomain.Error{
			Code:    koidomain.CodeGenerationConflict,
			Message: "generation conflict",
			RunID:   runID,
		})
	resp = doJSON(t, client, http.MethodPost, base, `{"mode":"initial"}`, brandID)
	var conflict apiresponse.Envelope
	readJSON(t, resp, &conflict)
	require.Equal(t, http.StatusConflict, resp.StatusCode)
	require.Equal(t, koidomain.CodeGenerationConflict, conflict.Error.Code)
	require.Equal(t, "/api/v1"+"/generations/"+runID.String(), conflict.Error.StatusURL)

	c.EXPECT().Create(identityIs(userID), brandID, koidomain.GenerationPrompts, mock.Anything).Return(nil, false, koidomain.ErrIdempotencyConflict)
	resp = doJSON(t, client, http.MethodPost, base, `{"mode":"prompts"}`, brandID)
	readJSON(t, resp, &conflict)
	require.Equal(t, http.StatusConflict, resp.StatusCode)
	require.Equal(t, koidomain.CodeIdempotencyConflict, conflict.Error.Code)

	c.EXPECT().Create(identityIs(userID), brandID, koidomain.GenerationCompetitors, mock.Anything).Return(nil, false, koidomain.ErrFeatureUnavailable)
	resp = doJSON(t, client, http.MethodPost, base, `{"mode":"competitors"}`, brandID)
	readJSON(t, resp, &conflict)
	require.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	require.Equal(t, koidomain.CodeFeatureUnavailable, conflict.Error.Code)
}
