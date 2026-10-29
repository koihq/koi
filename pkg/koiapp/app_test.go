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
	t.Setenv("KOI_DEPLOYMENT_MODE", "selfhosted")
	t.Setenv("KOI_CLORO_API_KEY", "")
	t.Setenv("KOI_LOG_LEVEL", "info")
	t.Setenv("KOI_LOG_FORMAT", "text")

	for _, key := range []string{"KOI_SESSION_LIFETIME", "KOI_OPENROUTER_API_KEY", "KOI_WORKER_CONCURRENCY", "KOI_PROVIDER_TIMEOUT", "KOI_DB_MAX_OPEN_CONNS"} {
		t.Setenv(key, "")
		require.NoError(t, os.Unsetenv(key))
	}

	cfg, err := LoadConfig()
	require.NoError(t, err)
	require.Equal(t, 4, cfg.WorkerConcurrency)
	require.Equal(t, 8*time.Hour, cfg.SessionLifetime)
	require.Equal(t, 3*time.Minute, cfg.ProviderTimeout)
	require.Equal(t, 12, cfg.DBMaxOpenConns)
	require.Empty(t, cfg.OpenRouterAPIKey)
	require.Empty(t, cfg.CloroAPIKey)

	t.Setenv("KOI_OPENROUTER_API_KEY", " test-key ")
	t.Setenv("KOI_SESSION_LIFETIME", "12h")
	t.Setenv("KOI_WORKER_CONCURRENCY", "2")
	t.Setenv("KOI_PROVIDER_TIMEOUT", "1m15s")
	t.Setenv("KOI_DB_MAX_OPEN_CONNS", "3")
	cfg, err = LoadConfig()
	require.NoError(t, err)
	require.Equal(t, "test-key", cfg.OpenRouterAPIKey)
	require.Equal(t, 12*time.Hour, cfg.SessionLifetime)
	require.Equal(t, 2, cfg.WorkerConcurrency)
	require.Equal(t, 75*time.Second, cfg.ProviderTimeout)
	require.Equal(t, 3, cfg.DBMaxOpenConns)
}

func TestLoadConfigWelcomeEmailPair(t *testing.T) {
	t.Setenv("KOI_PORT", ":8080")
	t.Setenv("KOI_DATABASE_URL", "postgres://example/test")
	t.Setenv("KOI_SESSION_SECRET", "test-secret")
	t.Setenv("KOI_DEPLOYMENT_MODE", "selfhosted")
	t.Setenv("KOI_RESEND_API_KEY", "test-key")
	t.Setenv("KOI_EMAIL_FROM", "")

	_, err := LoadConfig()
	require.ErrorContains(t, err, "KOI_EMAIL_FROM")

	t.Setenv("KOI_EMAIL_FROM", "Koi <hello@mail.example.com>")
	cfg, err := LoadConfig()
	require.NoError(t, err)
	require.Equal(t, "test-key", cfg.ResendAPIKey)
}

func TestLoadConfigRequiredValues(t *testing.T) {
	for _, tc := range []struct {
		mode string
		key  string
	}{
		{"selfhosted", "KOI_PORT"},
		{"selfhosted", "KOI_DATABASE_URL"},
		{"selfhosted", "KOI_SESSION_SECRET"},
		{"cloud", "KOI_PORT"},
		{"cloud", "KOI_DATABASE_URL"},
		{"cloud", "KOI_SESSION_SECRET"},
		{"cloud", "KOI_CLORO_API_KEY"},
		{"cloud", "KOI_STRIPE_SECRET_KEY"},
		{"cloud", "KOI_STRIPE_WEBHOOK_SECRET"},
		{"cloud", "KOI_STRIPE_PRICE_ID"},
		{"cloud", "KOI_PUBLIC_URL"},
	} {
		t.Run(tc.mode+"/"+tc.key, func(t *testing.T) {
			t.Setenv("KOI_DEPLOYMENT_MODE", tc.mode)
			t.Setenv("KOI_PORT", ":8080")
			t.Setenv("KOI_DATABASE_URL", "postgres://example/test")
			t.Setenv("KOI_SESSION_SECRET", "test-secret")
			t.Setenv("KOI_CLORO_API_KEY", "cloro-key")
			t.Setenv("KOI_STRIPE_SECRET_KEY", "stripe-key")
			t.Setenv("KOI_STRIPE_WEBHOOK_SECRET", "webhook-key")
			t.Setenv("KOI_STRIPE_PRICE_ID", "price-id")
			t.Setenv("KOI_PUBLIC_URL", "https://koi.example")
			t.Setenv("KOI_LOG_LEVEL", "info")
			t.Setenv("KOI_LOG_FORMAT", "text")

			_, err := LoadConfig()
			require.NoError(t, err)

			t.Setenv(tc.key, " \t ")
			_, err = LoadConfig()
			require.ErrorContains(t, err, tc.key)
		})
	}
}

func TestLoadConfigRejectsInvalidValues(t *testing.T) {
	for _, tc := range []struct{ key, value string }{
		{"KOI_SESSION_LIFETIME", ""},
		{"KOI_SESSION_LIFETIME", "0s"},
		{"KOI_SESSION_LIFETIME", "-1h"},
		{"KOI_SESSION_LIFETIME", "8"},
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
		{"KOI_LOG_LEVEL", "verbose"},
		{"KOI_LOG_FORMAT", "yaml"},
	} {
		t.Run(tc.key+"="+tc.value, func(t *testing.T) {
			t.Setenv("KOI_PORT", ":8080")
			t.Setenv("KOI_DATABASE_URL", "postgres://example/test")
			t.Setenv("KOI_SESSION_SECRET", "test-secret")
			t.Setenv("KOI_DEPLOYMENT_MODE", "selfhosted")
			t.Setenv("KOI_LOG_LEVEL", "info")
			t.Setenv("KOI_LOG_FORMAT", "text")
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
