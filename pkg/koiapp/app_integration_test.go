package koiapp

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverdatabasesql"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/internal/testdb"
	"github.com/koihq/koi/pkg/koidatabase"
)

type shutdownJobArgs struct{}

func (shutdownJobArgs) Kind() string { return "koi_test_shutdown" }

func TestRunRiverMigrationsAndShutdown(t *testing.T) {
	scopedDSN := testdb.URL(t)
	checkDB, err := koidatabase.Open(scopedDSN)
	require.NoError(t, err)
	checkSQL, err := checkDB.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, checkSQL.Close()) })

	cfg := Config{
		DatabaseURL:       scopedDSN,
		SessionSecret:     "test-secret",
		SessionLifetime:   8 * time.Hour,
		WorkerConcurrency: 2,
		ProviderTimeout:   45 * time.Second,
		DBMaxOpenConns:    2,
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
		Timeout: time.Second,
	}
	defer httpClient.CloseIdleConnections()
	var firstMigrationCount int

	for attempt := range 2 {
		cancel, done, address := startTestApp(t, &App{cfg: cfg}, httpClient)
		resp, err := httpClient.Get("http://" + address + "/health")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		var health struct {
			Status string `json:"status"`
			DB     string `json:"db"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&health))
		require.Equal(t, "ok", health.Status)
		require.Equal(t, "ok", health.DB)
		require.NoError(t, resp.Body.Close())
		cancel()
		require.NoError(t, waitApp(t, done))
		var migrations int
		require.NoError(t, checkSQL.QueryRowContext(t.Context(), "SELECT count(*) FROM river_migration").Scan(&migrations))
		require.Positive(t, migrations)
		if attempt == 0 {
			firstMigrationCount = migrations
		} else {
			require.Equal(t, firstMigrationCount, migrations)
		}
	}

	workers := river.NewWorkers()
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	river.AddWorker(workers, river.WorkFunc(func(ctx context.Context, _ *river.Job[shutdownJobArgs]) error {
		started <- struct{}{}

		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}))
	cfg.OpenRouterAPIKey = "test-key"
	cancel, done, address := startTestApp(t, &App{cfg: cfg, workers: workers}, httpClient)
	inserter, err := river.NewClient(riverdatabasesql.New(checkSQL), &river.Config{})
	require.NoError(t, err)
	inserted, err := inserter.Insert(t.Context(), shutdownJobArgs{}, &river.InsertOpts{Queue: "generation"})
	require.NoError(t, err)

	select {
	case <-started:
	case <-time.After(10 * time.Second):
		t.Fatal("River did not start the registered worker")
	}

	cancel()
	deadline := time.After(5 * time.Second)

	for {
		connection, err := net.DialTimeout("tcp", address, 100*time.Millisecond)
		if err != nil {
			break
		}

		_ = connection.Close()

		select {
		case <-deadline:
			t.Fatal("HTTP server kept accepting requests during shutdown")
		case <-time.After(20 * time.Millisecond):
		}
	}

	select {
	case err := <-done:
		t.Fatalf("app stopped before River worker finished: %v", err)
	default:
	}

	close(release)
	require.NoError(t, waitApp(t, done))
	var state string
	require.NoError(t, checkSQL.QueryRowContext(t.Context(), "SELECT state FROM river_job WHERE id = $1", inserted.Job.ID).Scan(&state))
	require.Equal(t, "completed", state)
}

func startTestApp(t *testing.T, app *App, client *http.Client) (context.CancelFunc, <-chan error, string) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- app.run(ctx, listener) }()
	address := listener.Addr().String()
	deadline := time.After(10 * time.Second)

	for {
		resp, err := client.Get("http://" + address + "/health")
		if err == nil {
			_ = resp.Body.Close()

			return cancel, done, address
		}

		select {
		case runErr := <-done:
			cancel()
			t.Fatalf("app failed to start: %v", runErr)
		case <-deadline:
			cancel()
			t.Fatalf("app did not start: %v", err)
		case <-time.After(20 * time.Millisecond):
		}
	}
}

func waitApp(t *testing.T, done <-chan error) error {
	t.Helper()

	select {
	case err := <-done:
		return err
	case <-time.After(10 * time.Second):
		t.Fatal("app did not stop")
		return nil
	}
}
