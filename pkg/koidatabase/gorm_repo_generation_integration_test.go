package koidatabase

import (
	"errors"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverdatabasesql"
	"github.com/riverqueue/river/rivermigrate"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/koiauth"
	"github.com/koihq/koi/pkg/koidomain"
)

func TestGenerationTransactionalEnqueueAndIndexes(t *testing.T) {
	dsn := os.Getenv("KOI_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set KOI_TEST_DATABASE_URL to run the Postgres integration test")
	}
	admin, err := Open(dsn)
	require.NoError(t, err)
	schemaName := "koi_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	require.NoError(t, admin.Exec("CREATE SCHEMA "+schemaName).Error)
	t.Cleanup(func() { require.NoError(t, admin.Exec("DROP SCHEMA "+schemaName+" CASCADE").Error) })
	scopedURL, err := url.Parse(dsn)
	require.NoError(t, err)
	params := scopedURL.Query()
	params.Set("search_path", schemaName)
	scopedURL.RawQuery = params.Encode()
	db, err := Open(scopedURL.String())
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	require.NoError(t, Migrate(db))
	client, err := river.NewClient(riverdatabasesql.New(sqlDB), &river.Config{})
	require.NoError(t, err)
	store := NewStore(db, client)
	user := &koidomain.User{Email: uuid.NewString() + "@example.com", PasswordHash: "hash"}
	require.NoError(t, store.UserRepo().Create(t.Context(), user))
	ctx := koiauth.WithIdentity(t.Context(), koiauth.Identity{UserID: user.ID})
	brand := &koidomain.Brand{Name: "Koi", Domain: "koi.dev", Country: "IT"}
	require.NoError(t, store.BrandRepo().Create(ctx, brand))
	key := "setup-1"

	// A missing River table makes InsertTx fail after GORM inserts the run; Atomic must roll it back.
	_, _, err = koidomain.NewGenerationCommander(store, true).Create(ctx, brand.ID, koidomain.GenerationInitial, &key)
	require.Error(t, err)
	var runCount, jobCount int64
	require.NoError(t, db.Model(&koidomain.GenerationRun{}).Count(&runCount).Error)
	require.Zero(t, runCount)

	migrator, err := rivermigrate.New(riverdatabasesql.New(sqlDB), nil)
	require.NoError(t, err)
	_, err = migrator.Migrate(t.Context(), rivermigrate.DirectionUp, nil)
	require.NoError(t, err)

	type result struct {
		run *koidomain.GenerationRun
		err error
	}
	results := make(chan result, 2)
	var start sync.WaitGroup
	start.Add(1)
	for range 2 {
		go func() {
			start.Wait()
			run, _, createErr := koidomain.NewGenerationCommander(store, true).Create(ctx, brand.ID, koidomain.GenerationInitial, &key)
			results <- result{run: run, err: createErr}
		}()
	}
	start.Done()
	first, second := <-results, <-results
	require.NoError(t, first.err)
	require.NoError(t, second.err)
	require.Equal(t, first.run.ID, second.run.ID)
	require.NotNil(t, first.run.RiverJobID)
	require.Equal(t, first.run.RiverJobID, second.run.RiverJobID)
	require.NoError(t, db.Model(&koidomain.GenerationRun{}).Count(&runCount).Error)
	require.EqualValues(t, 1, runCount)
	require.NoError(t, db.Table("river_job").Count(&jobCount).Error)
	require.EqualValues(t, 1, jobCount)
	var jobArgs string
	require.NoError(t, sqlDB.QueryRowContext(t.Context(), "SELECT args::text FROM river_job WHERE id = $1", *first.run.RiverJobID).Scan(&jobArgs))
	require.JSONEq(t, `{"run_id":"`+first.run.ID.String()+`"}`, jobArgs)

	// The active and idempotency constraints remain effective for direct writes.
	base := koidomain.GenerationRun{BrandID: brand.ID, Mode: koidomain.GenerationPrompts, State: koidomain.GenerationRunning,
		ModelSlug: koidomain.GenerationModelSlug, PromptVersion: koidomain.GenerationPromptVersion,
		BrandSnapshot: koidomain.BrandSnapshot{Name: "Koi", Domain: "koi.dev", Country: "IT"}}
	requireConstraint(t, db.Create(&base).Error, "idx_generation_runs_active")
	require.NoError(t, db.Model(&koidomain.GenerationRun{}).Where("id = ?", first.run.ID).Update("state", koidomain.GenerationFailed).Error)
	duplicate := base
	duplicate.ID = uuid.Nil
	duplicate.State = koidomain.GenerationFailed
	duplicate.IdempotencyKey = &key
	requireConstraint(t, db.Create(&duplicate).Error, "idx_generation_runs_key")
	require.NoError(t, db.Model(&koidomain.GenerationRun{}).Where("id = ?", first.run.ID).
		Update("created_at", time.Now().Add(-time.Hour)).Error)
	newer, _, err := koidomain.NewGenerationCommander(store, true).Create(ctx, brand.ID, koidomain.GenerationPrompts, nil)
	require.NoError(t, err)
	page1, err := koidomain.NewGenerationQuerier(store).List(ctx, brand.ID, koidomain.PageParams{Page: 1, PageSize: 1})
	require.NoError(t, err)
	require.EqualValues(t, 2, page1.TotalItems)
	require.EqualValues(t, 2, page1.TotalPages)
	require.True(t, page1.HasNext)
	require.Equal(t, newer.ID, page1.Items[0].ID)
	page2, err := koidomain.NewGenerationQuerier(store).List(ctx, brand.ID, koidomain.PageParams{Page: 2, PageSize: 1})
	require.NoError(t, err)
	require.True(t, page2.HasPrev)
	require.Equal(t, first.run.ID, page2.Items[0].ID)

	otherUser := &koidomain.User{Email: uuid.NewString() + "@example.com", PasswordHash: "hash"}
	require.NoError(t, store.UserRepo().Create(t.Context(), otherUser))
	otherCtx := koiauth.WithIdentity(t.Context(), koiauth.Identity{UserID: otherUser.ID})
	_, err = koidomain.NewGenerationQuerier(store).Get(otherCtx, brand.ID, first.run.ID)
	require.ErrorIs(t, err, koidomain.ErrNotFound)
	_, err = koidomain.NewGenerationQuerier(store).List(otherCtx, brand.ID, koidomain.PageParams{Page: 1, PageSize: 10})
	require.ErrorIs(t, err, koidomain.ErrNotFound)
	_, _, err = koidomain.NewGenerationCommander(store, true).Create(otherCtx, brand.ID, koidomain.GenerationPrompts, nil)
	require.ErrorIs(t, err, koidomain.ErrNotFound)

	require.NoError(t, db.Delete(&koidomain.Brand{}, "id = ?", brand.ID).Error)
	require.NoError(t, db.Model(&koidomain.GenerationRun{}).Count(&runCount).Error)
	require.Zero(t, runCount)
}

func requireConstraint(t *testing.T, err error, name string) {
	t.Helper()
	require.Error(t, err)
	var pgErr *pgconn.PgError
	require.True(t, errors.As(err, &pgErr), "error: %v", err)
	require.Equal(t, "23505", pgErr.Code)
	require.Equal(t, name, pgErr.ConstraintName)
}
