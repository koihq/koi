package koiapp

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverdatabasesql"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/internal/testdb"
	"github.com/koihq/koi/pkg/koidatabase"
	"github.com/koihq/koi/pkg/koidomain"
)

func TestMaintenanceRunsWithoutCloro(t *testing.T) {
	cfg := Config{
		DatabaseURL:       testdb.URL(t),
		SessionSecret:     "test-secret",
		OpenRouterAPIKey:  "test-key",
		WorkerConcurrency: 1,
		ProviderTimeout:   3 * time.Minute,
		DBMaxOpenConns:    4,
	}
	client := &http.Client{Timeout: time.Second}
	t.Cleanup(client.CloseIdleConnections)
	cancel, done, _ := startTestApp(t, &App{cfg: cfg}, client)
	t.Cleanup(cancel)
	db, err := koidatabase.Open(cfg.DatabaseURL)
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

	// The RunOnStart periodic job exists even though collection is disabled.
	require.Eventually(t, func() bool {
		var count int64
		err := db.Table("river_job").Where("kind = ? AND queue = ? AND state = ?", "koi_maintenance", "maintenance", "completed").Count(&count).Error
		return err == nil && count > 0
	}, 10*time.Second, 20*time.Millisecond)

	user := &koidomain.User{Email: uuid.NewString() + "@example.com", PasswordHash: "hash"}
	require.NoError(t, db.Create(user).Error)
	brand := &koidomain.Brand{UserID: user.ID, Name: "Koi", Domain: "koi.dev", Country: "IT"}
	require.NoError(t, db.Create(brand).Error)
	run := &koidomain.GenerationRun{
		BrandID:       brand.ID,
		Mode:          koidomain.GenerationPrompts,
		State:         koidomain.GenerationRunning,
		ModelSlug:     "test",
		BrandSnapshot: brand.Snapshot(),
	}
	require.NoError(t, db.Create(run).Error)
	inserter, err := river.NewClient(riverdatabasesql.New(sqlDB), &river.Config{})
	require.NoError(t, err)
	_, err = inserter.Insert(t.Context(), maintenanceArgs{}, &river.InsertOpts{Queue: "maintenance"})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		var status koidomain.GenerationRun
		return db.First(&status, "id = ?", run.ID).Error == nil && status.State == koidomain.GenerationFailed && status.ErrorCode != nil && *status.ErrorCode == "job_lost"
	}, 10*time.Second, 20*time.Millisecond)

	cancel()
	require.NoError(t, waitApp(t, done))
}
