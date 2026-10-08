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
	store := koidatabase.NewStore(db, client)

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

	commander := koidomain.NewVisibilityCommander(store)
	run, err := commander.Create(ctx, brand.ID)
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
		collect: func(ctx context.Context, _ koidomain.BrandSnapshot, _ koidomain.VisibilityResult) (koidomain.VisibilityResult, error) {
			<-ctx.Done()

			return koidomain.VisibilityResult{}, ctx.Err()
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
	newRun, err := commander.Create(ctx, brand.ID)
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
	visibilityHTTP(t, client, http.MethodPost, brandURL+"/runs", "", 409, nil)

	for _, text := range []string{"Best analytics for startups?", "Alternatives to Example Analytics?"} {
		body, err := json.Marshal(map[string]string{"text": text})
		require.NoError(t, err)
		visibilityHTTP(t, client, http.MethodPost, brandURL+"/prompts", string(body), 201, nil)
	}

	var accepted struct {
		ID        uuid.UUID `json:"id"`
		StatusURL string    `json:"status_url"`
	}
	visibilityHTTP(t, client, http.MethodPost, brandURL+"/runs", "", 202, &accepted)
	var status struct {
		State     koidomain.VisibilityState `json:"state"`
		Completed int                       `json:"completed_results"`
		Total     int                       `json:"total_results"`
	}
	require.Eventually(t, func() bool {
		visibilityHTTP(t, client, http.MethodGet, "http://"+address+accepted.StatusURL, "", 200, &status)

		return status.State == koidomain.VisibilitySucceeded
	}, 10*time.Second, 50*time.Millisecond)
	require.Equal(t, 8, status.Completed)
	require.Equal(t, 8, status.Total)
	var results koidomain.Page[koidomain.VisibilityResult]
	resultsPath := accepted.StatusURL + "/results?page=1&pageSize=40"
	visibilityHTTP(t, client, http.MethodGet, "http://"+address+resultsPath, "", 200, &results)
	require.Len(t, results.Items, 8)

	for _, result := range results.Items {
		require.NoError(t, result.Validate())
		require.Equal(t, koidomain.CollectorStub, result.Collector)
	}

	otherJar, err := cookiejar.New(nil)
	require.NoError(t, err)

	other := &http.Client{
		Jar:     otherJar,
		Timeout: 2 * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
	}

	t.Cleanup(other.CloseIdleConnections)
	visibilityHTTP(t, other, http.MethodPost, base+"/auth/signup",
		`{"email":"visibility-`+uuid.NewString()+`@example.com","password":"password123"}`, 201, nil)

	for _, path := range []string{brandURL + "/runs", "http://" + address + accepted.StatusURL, "http://" + address + resultsPath} {
		visibilityHTTP(t, other, http.MethodGet, path, "", 404, nil)
	}

	visibilityHTTP(t, other, http.MethodPost, brandURL+"/runs", "", 404, nil)
	visibilityHTTP(t, http.DefaultClient, http.MethodGet, "http://"+address+accepted.StatusURL, "", 401, nil)
	cancel()
	require.NoError(t, waitApp(t, done))
	// A new app instance reads the same run and results with the persisted cookie session.
	cancel, done, address = startTestApp(t, &App{cfg: cfg}, client)
	t.Cleanup(cancel)
	var reloaded koidomain.Page[koidomain.VisibilityResult]
	visibilityHTTP(t, client, http.MethodGet, "http://"+address+resultsPath, "", 200, &reloaded)
	require.Equal(t, results, reloaded)
	visibilityHTTP(t, client, http.MethodGet, "http://"+address+accepted.StatusURL, "", 200, &status)
	require.Equal(t, koidomain.VisibilitySucceeded, status.State)
	cancel()
	require.NoError(t, waitApp(t, done))
}
