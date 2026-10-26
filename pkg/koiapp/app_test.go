package koiapp

import (
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	t.Setenv("KOI_PORT", ":8080")
	t.Setenv("KOI_DATABASE_URL", "postgres://example/test")
	t.Setenv("KOI_SESSION_SECRET", "test-secret")

	for _, key := range []string{"KOI_OPENROUTER_API_KEY", "KOI_WORKER_CONCURRENCY", "KOI_PROVIDER_TIMEOUT", "KOI_DB_MAX_OPEN_CONNS"} {
		t.Setenv(key, "")
		require.NoError(t, os.Unsetenv(key))
	}

	cfg, err := LoadConfig()
	require.NoError(t, err)
	require.Equal(t, 4, cfg.WorkerConcurrency)
	require.Equal(t, 3*time.Minute, cfg.ProviderTimeout)
	require.Equal(t, 12, cfg.DBMaxOpenConns)
	require.Empty(t, cfg.OpenRouterAPIKey)

	t.Setenv("KOI_OPENROUTER_API_KEY", " test-key ")
	t.Setenv("KOI_WORKER_CONCURRENCY", "2")
	t.Setenv("KOI_PROVIDER_TIMEOUT", "1m15s")
	t.Setenv("KOI_DB_MAX_OPEN_CONNS", "3")
	cfg, err = LoadConfig()
	require.NoError(t, err)
	require.Equal(t, "test-key", cfg.OpenRouterAPIKey)
	require.Equal(t, 2, cfg.WorkerConcurrency)
	require.Equal(t, 75*time.Second, cfg.ProviderTimeout)
	require.Equal(t, 3, cfg.DBMaxOpenConns)
}

func TestLoadConfigRejectsInvalidValues(t *testing.T) {
	for _, tc := range []struct{ key, value string }{
		{"KOI_WORKER_CONCURRENCY", ""},
		{"KOI_WORKER_CONCURRENCY", "0"},
		{"KOI_WORKER_CONCURRENCY", "10001"},
		{"KOI_WORKER_CONCURRENCY", "abc"},
		{"KOI_PROVIDER_TIMEOUT", ""},
		{"KOI_PROVIDER_TIMEOUT", "0s"},
		{"KOI_PROVIDER_TIMEOUT", "-1s"},
		{"KOI_PROVIDER_TIMEOUT", "45"},
		{"KOI_DB_MAX_OPEN_CONNS", ""},
		{"KOI_DB_MAX_OPEN_CONNS", "-1"},
		{"KOI_DB_MAX_OPEN_CONNS", "1.5"},
	} {
		t.Run(tc.key+"="+tc.value, func(t *testing.T) {
			t.Setenv("KOI_PORT", ":8080")
			t.Setenv("KOI_DATABASE_URL", "postgres://example/test")
			t.Setenv("KOI_SESSION_SECRET", "test-secret")
			t.Setenv("KOI_WORKER_CONCURRENCY", "4")
			t.Setenv("KOI_PROVIDER_TIMEOUT", "45s")
			t.Setenv("KOI_DB_MAX_OPEN_CONNS", "12")
			t.Setenv(tc.key, tc.value)
			_, err := LoadConfig()
			require.ErrorContains(t, err, tc.key)
		})
	}
}

func TestConfigurePool(t *testing.T) {
	for _, maxOpen := range []int{2, 12} {
		db, err := sql.Open("pgx", "postgres://localhost/unused")
		require.NoError(t, err)
		configurePool(db, maxOpen)
		require.Equal(t, maxOpen, db.Stats().MaxOpenConnections)
		require.NoError(t, db.Close())
	}
}
