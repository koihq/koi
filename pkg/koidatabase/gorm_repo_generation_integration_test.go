package koidatabase

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverdatabasesql"
	"github.com/riverqueue/river/rivermigrate"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/internal/testdb"
	"github.com/koihq/koi/pkg/koiauth"
	"github.com/koihq/koi/pkg/koidomain"
)

func TestGenerationTransactionalEnqueueAndIndexes(t *testing.T) {
	db, err := Open(testdb.URL(t))
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
	promptsFromInitial, _, err := koidomain.NewGenerationCommander(store, true).Create(ctx, brand.ID, koidomain.GenerationPrompts, nil)
	require.NoError(t, err)
	require.Equal(t, first.run.ID, promptsFromInitial.ID)
	_, _, err = koidomain.NewGenerationCommander(store, true).Create(ctx, brand.ID, koidomain.GenerationPrompts, &key)
	require.ErrorIs(t, err, koidomain.ErrIdempotencyConflict)

	// The active and idempotency constraints remain effective for direct writes.
	base := koidomain.GenerationRun{
		BrandID:   brand.ID,
		Mode:      koidomain.GenerationPrompts,
		State:     koidomain.GenerationRunning,
		ModelSlug: koidomain.GenerationModelSlug,
		BrandSnapshot: koidomain.BrandSnapshot{
			Name:    "Koi",
			Domain:  "koi.dev",
			Country: "IT",
		},
	}

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
	_, _, err = koidomain.NewGenerationCommander(store, true).Create(ctx, brand.ID, koidomain.GenerationCompetitors, nil)
	require.ErrorIs(t, err, koidomain.ErrGenerationConflict)
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

	for i := range 120 {
		require.NoError(t, store.PromptRepo().Create(ctx, &koidomain.Prompt{
			BrandID:   brand.ID,
			Text:      fmt.Sprintf("Question %d?", i),
			Source:    koidomain.PromptSourceManual,
			Confirmed: true,
		}))
	}

	started, draftInput, err := store.BeginGenerationAttempt(ctx, newer.ID, *newer.RiverJobID, 1)
	require.NoError(t, err)
	require.NotNil(t, started)
	require.Len(t, draftInput.ExistingPrompts, 120)
	require.Equal(t, newer.BrandSnapshot, draftInput.Brand)
	retryAt := time.Now().Add(time.Minute)
	require.NoError(t, store.RecordGenerationFailure(ctx, newer.ID, 1, "provider_rate_limited", &retryAt))
	var retrying koidomain.GenerationRun
	require.NoError(t, db.First(&retrying, "id = ?", newer.ID).Error)
	require.Equal(t, koidomain.GenerationRunning, retrying.State)
	require.Equal(t, "provider_rate_limited", *retrying.ErrorCode)
	require.NotNil(t, retrying.NextRetryAt)
	started, draftInput, err = store.BeginGenerationAttempt(ctx, newer.ID, *newer.RiverJobID, 2)
	require.NoError(t, err)
	require.NotNil(t, started)
	require.Len(t, draftInput.ExistingPrompts, 120)
	require.NoError(t, store.PromptRepo().Create(ctx, &koidomain.Prompt{
		BrandID:   brand.ID,
		Text:      "Added during generation?",
		Source:    koidomain.PromptSourceManual,
		Confirmed: true,
	}))
	require.NoError(t, db.Exec("ALTER TABLE prompts ADD CONSTRAINT reject_draft_test CHECK (text <> 'New AI question?')").Error)

	_, _, err = store.FinishGenerationAttempt(ctx, newer.ID, 2, koidomain.GenerationResult{
		Prompts: []string{"First inserted?", "New AI question?"},
	})

	require.Error(t, err)
	var rolledBack int64
	require.NoError(t, db.Model(&koidomain.Prompt{}).Where("brand_id = ? AND text = ?", brand.ID, "First inserted?").Count(&rolledBack).Error)
	require.Zero(t, rolledBack)
	var stillRunning koidomain.GenerationRun
	require.NoError(t, db.First(&stillRunning, "id = ?", newer.ID).Error)
	require.Equal(t, koidomain.GenerationRunning, stillRunning.State)
	require.NoError(t, db.Exec("ALTER TABLE prompts DROP CONSTRAINT reject_draft_test").Error)

	promptCount, competitorCount, err := store.FinishGenerationAttempt(ctx, newer.ID, 2, koidomain.GenerationResult{
		Prompts: []string{"Question 7?", "Added during generation?", "New AI question?", "New AI question?", "  "},
		Competitors: []koidomain.CompetitorIdentity{
			{
				Name:   "Unexpected",
				Domain: "unexpected.example",
			},
		},
	})

	require.NoError(t, err)
	require.Equal(t, 1, promptCount)
	require.Zero(t, competitorCount)
	retrying = koidomain.GenerationRun{}
	require.NoError(t, db.First(&retrying, "id = ?", newer.ID).Error)
	require.Equal(t, koidomain.GenerationSucceeded, retrying.State)
	require.Equal(t, 2, retrying.Attempt)
	require.Equal(t, 1, retrying.PromptCount)
	require.Nil(t, retrying.ErrorCode)
	require.Nil(t, retrying.NextRetryAt)
	var unexpectedCount int64
	require.NoError(t, db.Model(&koidomain.Competitor{}).Where("brand_id = ? AND domain = ?", brand.ID, "unexpected.example").Count(&unexpectedCount).Error)
	require.Zero(t, unexpectedCount)
	started, _, err = store.BeginGenerationAttempt(ctx, newer.ID, *newer.RiverJobID, 3)
	require.NoError(t, err)
	require.Nil(t, started)

	raceRun, _, err := koidomain.NewGenerationCommander(store, true).Create(ctx, brand.ID, koidomain.GenerationPrompts, nil)
	require.NoError(t, err)
	_, _, err = store.BeginGenerationAttempt(ctx, raceRun.ID, *raceRun.RiverJobID, 1)
	require.NoError(t, err)
	manualResult := make(chan error, 1)
	startRace := make(chan struct{})
	go func() {
		<-startRace
		manualResult <- store.PromptRepo().Create(ctx, &koidomain.Prompt{
			BrandID:   brand.ID,
			Text:      "Shared question?",
			Source:    koidomain.PromptSourceManual,
			Confirmed: true,
		})
	}()
	close(startRace)
	raceCount, _, err := store.FinishGenerationAttempt(ctx, raceRun.ID, 1, koidomain.GenerationResult{
		Prompts: []string{"Shared question?"},
	})
	require.NoError(t, err)
	manualErr := <-manualResult
	if raceCount == 0 {
		require.NoError(t, manualErr)
	} else {
		require.Error(t, manualErr)
	}

	var sharedCount int64
	require.NoError(t, db.Model(&koidomain.Prompt{}).Where("brand_id = ? AND lower(text) = ?", brand.ID, "shared question?").Count(&sharedCount).Error)
	require.EqualValues(t, 1, sharedCount)

	require.NoError(t, store.CompetitorRepo().Create(ctx, &koidomain.Competitor{
		BrandID:   brand.ID,
		Name:      "Atlas",
		Domain:    "atlas.example",
		Source:    koidomain.CompetitorSourceManual,
		Confirmed: true,
	}))
	competitorRun, _, err := koidomain.NewGenerationCommander(store, true).Create(ctx, brand.ID, koidomain.GenerationCompetitors, nil)
	require.NoError(t, err)
	_, draftInput, err = store.BeginGenerationAttempt(ctx, competitorRun.ID, *competitorRun.RiverJobID, 1)
	require.NoError(t, err)
	require.Equal(t, []koidomain.CompetitorIdentity{
		{
			Name:   "Atlas",
			Domain: "atlas.example",
		},
	}, draftInput.ExistingCompetitors)

	promptCount, competitorCount, err = store.FinishGenerationAttempt(ctx, competitorRun.ID, 1, koidomain.GenerationResult{
		Competitors: []koidomain.CompetitorIdentity{
			{
				Name:   "Atlas again",
				Domain: "ATLAS.EXAMPLE",
			},
			{
				Name:   "Koi",
				Domain: "koi.dev",
			},
			{
				Name:   "Beta",
				Domain: "beta.example",
			},
			{
				Name:   "Invalid",
				Domain: "https://invalid.example",
			},
		},
		Prompts: []string{"Unexpected question?"},
	})

	require.NoError(t, err)
	require.Zero(t, promptCount)
	require.Equal(t, 1, competitorCount)
	var beta koidomain.Competitor
	require.NoError(t, db.Where("brand_id = ? AND domain = ?", brand.ID, "beta.example").First(&beta).Error)
	require.Equal(t, koidomain.CompetitorSourceDrafted, beta.Source)
	require.False(t, beta.Confirmed)
	require.NoError(t, db.Model(&koidomain.Prompt{}).Where("brand_id = ? AND text = ?", brand.ID, "Unexpected question?").Count(&unexpectedCount).Error)
	require.Zero(t, unexpectedCount)
	var atlas koidomain.Competitor
	require.NoError(t, db.Where("brand_id = ? AND domain = ?", brand.ID, "atlas.example").First(&atlas).Error)
	require.Equal(t, koidomain.CompetitorSourceManual, atlas.Source)
	require.True(t, atlas.Confirmed)

	zeroRun, _, err := koidomain.NewGenerationCommander(store, true).Create(ctx, brand.ID, koidomain.GenerationCompetitors, nil)
	require.NoError(t, err)
	_, _, err = store.BeginGenerationAttempt(ctx, zeroRun.ID, *zeroRun.RiverJobID, 1)
	require.NoError(t, err)
	_, competitorCount, err = store.FinishGenerationAttempt(ctx, zeroRun.ID, 1, koidomain.GenerationResult{
		Competitors: []koidomain.CompetitorIdentity{
			{
				Name:   "Atlas",
				Domain: "atlas.example",
			},
		},
	})
	require.NoError(t, err)
	require.Zero(t, competitorCount)
	var zeroStatus koidomain.GenerationRun
	require.NoError(t, db.First(&zeroStatus, "id = ?", zeroRun.ID).Error)
	require.Equal(t, koidomain.GenerationSucceeded, zeroStatus.State)
	require.Zero(t, zeroStatus.CompetitorCount)

	lostRun, _, err := koidomain.NewGenerationCommander(store, true).Create(ctx, brand.ID, koidomain.GenerationPrompts, nil)
	require.NoError(t, err)
	_, _, err = store.BeginGenerationAttempt(ctx, lostRun.ID, *lostRun.RiverJobID, 1)
	require.NoError(t, err)
	require.NoError(t, db.Table("river_job").Where("id = ?", *lostRun.RiverJobID).
		Updates(map[string]any{"state": "discarded", "finalized_at": time.Now().UTC()}).Error)
	require.NoError(t, store.ReconcileGenerationRuns(ctx))
	var lostStatus koidomain.GenerationRun
	require.NoError(t, db.First(&lostStatus, "id = ?", lostRun.ID).Error)
	require.Equal(t, koidomain.GenerationFailed, lostStatus.State)
	require.Equal(t, "job_lost", *lostStatus.ErrorCode)

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
