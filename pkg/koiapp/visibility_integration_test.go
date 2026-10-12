package koiapp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverdatabasesql"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/internal/testdb"
	"github.com/koihq/koi/pkg/koiauth"
	"github.com/koihq/koi/pkg/koidatabase"
	"github.com/koihq/koi/pkg/koidomain"
)

func TestVisibilityTerminalTimeoutAllowsAnotherRun(t *testing.T) {
	db, err := koidatabase.Open(testdb.URL(t))
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	require.NoError(t, koidatabase.Migrate(db))
	require.NoError(t, migrateRiver(t.Context(), sqlDB))

	client, err := river.NewClient(riverdatabasesql.New(sqlDB), &river.Config{})
	require.NoError(t, err)
	store := koidatabase.NewStore(db, client).WithLiveCollection(true)

	user := &koidomain.User{
		Email:        uuid.NewString() + "@example.com",
		PasswordHash: "hash",
	}

	require.NoError(t, store.UserRepo().Create(t.Context(), user))
	ctx := koiauth.WithIdentity(t.Context(), koiauth.Identity{UserID: user.ID})

	brand := &koidomain.Brand{
		Name:    "Koi",
		Domain:  "koi.dev",
		Country: "IT",
	}

	require.NoError(t, store.BrandRepo().Create(ctx, brand))
	_, err = koidomain.NewPromptCommander(store).Create(ctx, brand.ID, koidomain.PromptParams{Text: "Best tools?"})
	require.NoError(t, err)

	require.NoError(t, store.VisibilityRepo().FinishReview(ctx, brand.ID, "prompts"))
	require.NoError(t, store.VisibilityRepo().FinishReview(ctx, brand.ID, "competitors"))
	run, err := store.VisibilityRepo().Active(ctx, brand.ID)
	require.NoError(t, err)

	job := &river.Job[koidatabase.VisibilityJobArgs]{
		JobRow: &rivertype.JobRow{
			ID:          *run.RiverJobID,
			Attempt:     3,
			MaxAttempts: 3,
		},
		Args: koidatabase.VisibilityJobArgs{
			RunID: run.ID,
		},
	}

	worker := &visibilityWorker{
		store: store,
		provider: visibilityTestProvider{
			submit: func(ctx context.Context, _ koidomain.VisibilityResult, _ string) (string, error) {
				<-ctx.Done()

				return "", ctx.Err()
			},
		},
	}

	workCtx, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
	defer cancel()
	require.ErrorIs(t, worker.Work(workCtx, job), context.DeadlineExceeded)

	failed, err := koidomain.NewVisibilityQuerier(store).Get(ctx, brand.ID, run.ID)
	require.NoError(t, err)
	require.Equal(t, koidomain.VisibilityFailed, failed.State)
	require.NotNil(t, failed.ErrorCode)
	require.Equal(t, "job_timeout", *failed.ErrorCode)
	require.NotNil(t, failed.FinishedAt)
	status, err := store.VisibilityRepo().Tracking(ctx, brand.ID)
	require.NoError(t, err)
	require.NoError(t, store.DispatchVisibility(ctx, *status.NextRunAt))
	newRun, err := store.VisibilityRepo().Active(ctx, brand.ID)
	require.NoError(t, err)
	require.NotEqual(t, run.ID, newRun.ID)
}

func visibilityHTTP(t *testing.T, client *http.Client, method, url, body string, expected int, output any) {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	require.NoError(t, err)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, expected, resp.StatusCode, "response: %s", data)
	if output != nil {
		require.NoError(t, json.Unmarshal(data, output))
	}
}

func TestVisibilityHTTPWithoutProviderKeysSurvivesRestart(t *testing.T) {
	cfg := Config{
		DatabaseURL:       testdb.URL(t),
		SessionSecret:     "test-secret",
		WorkerConcurrency: 2,
		ProviderTimeout:   45 * time.Second,
		DBMaxOpenConns:    4,
	}

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)

	client := &http.Client{
		Jar:     jar,
		Timeout: 2 * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
	}

	t.Cleanup(client.CloseIdleConnections)
	cancel, done, address := startTestApp(t, &App{cfg: cfg}, client)
	t.Cleanup(cancel)
	base := "http://" + address + "/api/v1"
	visibilityHTTP(t, client, http.MethodPost, base+"/auth/signup",
		`{"email":"visibility-`+uuid.NewString()+`@example.com","password":"password123"}`, 201, nil)
	var capabilities map[string]bool
	visibilityHTTP(t, client, http.MethodGet, base+"/capabilities", "", 200, &capabilities)
	require.False(t, capabilities["aiDrafting"])
	var brand koidomain.Brand
	visibilityHTTP(t, client, http.MethodPost, base+"/brands",
		`{"name":"Koi","domain":"koi.dev","country":"IT"}`, 201, &brand)
	brandURL := base + "/brands/" + brand.ID.String()
	for _, text := range []string{"Best analytics for startups?", "Alternatives to Example Analytics?"} {
		body, err := json.Marshal(map[string]string{"text": text})
		require.NoError(t, err)
		visibilityHTTP(t, client, http.MethodPost, brandURL+"/prompts", string(body), 201, nil)
	}
	visibilityHTTP(t, client, http.MethodPost, brandURL+"/prompts/confirm", "", 204, nil)
	visibilityHTTP(t, client, http.MethodPost, brandURL+"/competitors/confirm", "", 204, nil)
	var status koidomain.TrackingStatus
	visibilityHTTP(t, client, http.MethodGet, brandURL+"/tracking", "", 200, &status)
	require.True(t, status.PromptsReviewed && status.CompetitorsReviewed)
	require.Equal(t, "collection_unavailable", *status.BlockedReason)
	require.Nil(t, status.StartedAt)
	var runs koidomain.Page[koidomain.VisibilityRun]
	visibilityHTTP(t, client, http.MethodGet, brandURL+"/runs", "", 200, &runs)
	require.Empty(t, runs.Items)
	require.False(t, capabilities["liveCollection"])

	cancel()
	require.NoError(t, waitApp(t, done))
	cancel, done, address = startTestApp(t, &App{cfg: cfg}, client)
	t.Cleanup(cancel)
	var reloaded koidomain.TrackingStatus
	visibilityHTTP(t, client, http.MethodGet, "http://"+address+"/api/v1/brands/"+brand.ID.String()+"/tracking", "", 200, &reloaded)
	require.Equal(t, status, reloaded)
	cancel()
	require.NoError(t, waitApp(t, done))
}
