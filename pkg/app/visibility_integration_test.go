package app

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
	"github.com/koihq/koi/pkg/api"
	"github.com/koihq/koi/pkg/auth"
	"github.com/koihq/koi/pkg/database"
	"github.com/koihq/koi/pkg/domain"
	"github.com/koihq/koi/pkg/providers/cloro"
)

func TestVisibilityTerminalTimeoutAllowsAnotherRun(t *testing.T) {
	db, err := database.Open(testdb.URL(t))
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	require.NoError(t, database.Migrate(db))
	require.NoError(t, migrateRiver(t.Context(), sqlDB))

	client, err := river.NewClient(riverdatabasesql.New(sqlDB), &river.Config{})
	require.NoError(t, err)
	store := database.NewStore(db).WithEnqueueClient(client).WithLiveCollection(true)

	user := &domain.User{
		Email:        uuid.NewString() + "@example.com",
		PasswordHash: "hash",
	}

	require.NoError(t, store.UserRepo().Create(t.Context(), user))
	organization := &domain.Organization{Name: "Koi"}
	require.NoError(t, store.OrganizationRepo().Create(t.Context(), organization))
	require.NoError(t, store.MembershipRepo().Create(t.Context(), &domain.Membership{
		UserID: user.ID, OrganizationID: organization.ID, Role: domain.RoleOwner,
	}))
	identity, err := domain.ResolveIdentity(t.Context(), store, user.ID)
	require.NoError(t, err)
	ctx := auth.WithIdentity(t.Context(), identity)

	brand := &domain.Brand{
		OrganizationID: organization.ID,
		Name:           "Koi",
		Domain:         "koi.dev",
		Country:        "IT",
	}

	require.NoError(t, store.BrandRepo().Create(ctx, brand))
	_, err = domain.NewPromptCommander(store).Create(ctx, brand.ID, domain.CreatePromptParams{Text: "Best tools?"})
	require.NoError(t, err)

	require.NoError(t, store.VisibilityRepo().StartTracking(ctx, brand.ID))
	run, err := store.VisibilityRepo().Active(ctx, brand.ID)
	require.NoError(t, err)

	job := &river.Job[database.VisibilityJobArgs]{
		JobRow: &rivertype.JobRow{
			ID:          *run.RiverJobID,
			Attempt:     3,
			MaxAttempts: 3,
		},
		Args: database.VisibilityJobArgs{
			RunID: run.ID,
		},
	}

	worker := &visibilityWorker{
		store: store,
		provider: visibilityTestProvider{
			fetch: func(ctx context.Context, _ domain.VisibilityResult, _ string) (*cloro.Answer, error) {
				<-ctx.Done()

				return nil, ctx.Err()
			},
		},
	}

	workCtx, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
	defer cancel()
	require.ErrorIs(t, worker.Work(workCtx, job), context.DeadlineExceeded)

	failed, err := domain.NewVisibilityQuerier(store).Get(ctx, run.ID)
	require.NoError(t, err)
	require.Equal(t, domain.VisibilityFailed, failed.State)
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
		SessionLifetime:   8 * time.Hour,
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
		`{"email":"visibility-`+uuid.NewString()+`@example.com","password":"password-1234"}`, 201, nil)
	var capabilities struct {
		AIDrafting     bool `json:"aiDrafting"`
		LiveCollection bool `json:"liveCollection"`
	}
	visibilityHTTP(t, client, http.MethodGet, base+"/capabilities", "", 200, &capabilities)
	require.False(t, capabilities.AIDrafting)
	var brand api.BrandRes
	visibilityHTTP(t, client, http.MethodPost, base+"/brands",
		`{"name":"Koi","domain":"koi.dev","country":"IT"}`, 201, &brand)
	scopeBody := `{"brandId":"` + brand.ID.String() + `"}`
	for _, text := range []string{"Best analytics for startups?", "Alternatives to Example Analytics?"} {
		body, err := json.Marshal(map[string]string{"brandId": brand.ID.String(), "text": text})
		require.NoError(t, err)
		visibilityHTTP(t, client, http.MethodPost, base+"/prompts", string(body), 201, nil)
	}
	visibilityHTTP(t, client, http.MethodPost, base+"/tracking/start", scopeBody, 204, nil)
	var status api.TrackingStatusRes
	visibilityHTTP(t, client, http.MethodGet, base+"/tracking?brandId="+brand.ID.String(), "", 200, &status)
	require.True(t, status.Confirmed)
	require.Equal(t, "collection_unavailable", *status.BlockedReason)
	require.Nil(t, status.StartedAt)
	var runs domain.Page[domain.VisibilityRun]
	visibilityHTTP(t, client, http.MethodGet, base+"/runs?brandId="+brand.ID.String(), "", 200, &runs)
	require.Empty(t, runs.Items)
	require.False(t, capabilities.LiveCollection)

	cancel()
	require.NoError(t, waitApp(t, done))
	cancel, done, address = startTestApp(t, &App{cfg: cfg}, client)
	t.Cleanup(cancel)
	var reloaded api.TrackingStatusRes
	visibilityHTTP(t, client, http.MethodGet, "http://"+address+"/api/v1/tracking?brandId="+brand.ID.String(), "", 200, &reloaded)
	require.Equal(t, status, reloaded)
	cancel()
	require.NoError(t, waitApp(t, done))
}
