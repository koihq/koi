package koidatabase

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverdatabasesql"
	"github.com/riverqueue/river/rivermigrate"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/koihq/koi/internal/testdb"
	"github.com/koihq/koi/pkg/koiauth"
	"github.com/koihq/koi/pkg/koidomain"
	"github.com/koihq/koi/pkg/koiproviders/stub"
)

func visibilityFixture(t *testing.T, withRiver bool) (*gorm.DB, *Store, context.Context, *koidomain.Brand) {
	t.Helper()
	db, err := Open(testdb.URL(t))
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	require.NoError(t, Migrate(db))
	if withRiver {
		migrator, err := rivermigrate.New(riverdatabasesql.New(sqlDB), nil)
		require.NoError(t, err)
		_, err = migrator.Migrate(t.Context(), rivermigrate.DirectionUp, nil)
		require.NoError(t, err)
	}
	client, err := river.NewClient(riverdatabasesql.New(sqlDB), &river.Config{})
	require.NoError(t, err)
	store := NewStore(db, client)
	user := &koidomain.User{Email: uuid.NewString() + "@example.com", PasswordHash: "hash"}
	require.NoError(t, store.UserRepo().Create(t.Context(), user))
	ctx := koiauth.WithIdentity(t.Context(), koiauth.Identity{UserID: user.ID})
	brand := &koidomain.Brand{Name: "Northstar", Domain: "northstar.example", Country: "US"}
	require.NoError(t, store.BrandRepo().Create(ctx, brand))
	return db, store, ctx, brand
}

func TestVisibilityReadinessAndTransactionalRollback(t *testing.T) {
	db, store, ctx, brand := visibilityFixture(t, false)
	commander := koidomain.NewVisibilityCommander(store)
	_, err := commander.Create(ctx, brand.ID)
	var domainErr *koidomain.Error
	require.ErrorAs(t, err, &domainErr)
	require.Equal(t, koidomain.CodeRunNotReady, domainErr.Code)
	require.Equal(t, []koidomain.FieldError{{Field: "prompts", Code: "empty"}}, domainErr.Fields)
	for i := range 101 {
		prompt := &koidomain.Prompt{
			BrandID: brand.ID, Text: fmt.Sprintf("Question %03d?", i),
			Source: koidomain.PromptSourceDrafted, Confirmed: i < 100,
		}
		require.NoError(t, store.PromptRepo().Create(ctx, prompt))
	}
	competitor := &koidomain.Competitor{
		BrandID: brand.ID, Name: "Atlas", Domain: "atlas.example", Source: koidomain.CompetitorSourceDrafted,
	}
	require.NoError(t, store.CompetitorRepo().Create(ctx, competitor))
	_, err = commander.Create(ctx, brand.ID)
	require.ErrorAs(t, err, &domainErr)
	require.Equal(t, []koidomain.FieldError{
		{Field: "prompts", Code: "unconfirmed"},
		{Field: "competitors", Code: "unconfirmed"},
	}, domainErr.Fields)
	require.NoError(t, koidomain.NewPromptCommander(store).ConfirmAll(ctx, brand.ID))
	require.NoError(t, koidomain.NewCompetitorCommander(store).ConfirmAll(ctx, brand.ID))
	// InsertTx fails after the run and all 404 cells have been inserted.
	_, err = commander.Create(ctx, brand.ID)
	require.Error(t, err)
	for _, model := range []any{&koidomain.VisibilityRun{}, &koidomain.VisibilityResult{}} {
		var count int64
		require.NoError(t, db.Model(model).Count(&count).Error)
		require.Zero(t, count)
	}
}

func TestVisibilityMatrixSnapshotsRetryOwnershipAndReconciliation(t *testing.T) {
	db, store, ctx, brand := visibilityFixture(t, true)
	for _, text := range []string{"Best analytics for startups?", "Alternatives to Example Analytics?"} {
		_, err := koidomain.NewPromptCommander(store).Create(ctx, brand.ID, koidomain.PromptParams{Text: text})
		require.NoError(t, err)
	}
	commander := koidomain.NewVisibilityCommander(store)
	var wg sync.WaitGroup
	wg.Add(2)
	runs, errs := make([]*koidomain.VisibilityRun, 2), make([]error, 2)
	for i := range 2 {
		go func() {
			defer wg.Done()
			runs[i], errs[i] = commander.Create(ctx, brand.ID)
		}()
	}
	wg.Wait()
	for _, err := range errs {
		require.NoError(t, err)
	}
	require.Equal(t, runs[0].ID, runs[1].ID)
	run := runs[0]
	require.EqualValues(t, 8, run.TotalResults)
	require.NotNil(t, run.RiverJobID)
	var args string
	require.NoError(t, db.Raw("SELECT args::text FROM river_job WHERE id = ?", *run.RiverJobID).Scan(&args).Error)
	require.JSONEq(t, `{"run_id":"`+run.ID.String()+`"}`, args)
	pageParams := koidomain.PageParams{Page: 1, PageSize: 40}
	querier := koidomain.NewVisibilityQuerier(store)
	page, err := querier.ListResults(ctx, brand.ID, run.ID, pageParams)
	require.NoError(t, err)
	require.Len(t, page.Items, 8)
	for _, result := range page.Items {
		require.Nil(t, result.Outcome)
		require.Nil(t, result.FetchedAt)
		require.NotNil(t, result.Sources)
		require.Equal(t, koidomain.CollectorStub, result.Collector)
	}
	duplicate := page.Items[0]
	duplicate.ID = uuid.Nil
	requireConstraint(t, db.Create(&duplicate).Error, "idx_visibility_results_cell")
	duplicateRun := *run
	duplicateRun.ID = uuid.Nil
	requireConstraint(t, db.Create(&duplicateRun).Error, "idx_visibility_runs_active")
	_, pending, err := store.BeginVisibilityAttempt(ctx, run.ID, *run.RiverJobID, 1)
	require.NoError(t, err)
	require.Len(t, pending, 8)
	first, err := stub.Collect(ctx, run.BrandSnapshot, pending[0])
	require.NoError(t, err)
	now := time.Now().UTC()
	first.FetchedAt = &now
	require.NoError(t, store.SaveVisibilityResult(ctx, run.ID, 1, first))
	originalAnswer := first.RawAnswer
	first.RawAnswer = "Must never overwrite the original answer"
	require.NoError(t, store.SaveVisibilityResult(ctx, run.ID, 1, first))
	status, err := querier.Get(ctx, brand.ID, run.ID)
	require.NoError(t, err)
	require.EqualValues(t, 1, status.CompletedResults)
	require.Error(t, store.FinishVisibilityAttempt(ctx, run.ID, 1))
	_, pending, err = store.BeginVisibilityAttempt(ctx, run.ID, *run.RiverJobID, 2)
	require.NoError(t, err)
	require.Len(t, pending, 7)
	require.Error(t, store.SaveVisibilityResult(ctx, run.ID, 1, first))
	for _, result := range pending {
		answer, err := stub.Collect(ctx, run.BrandSnapshot, result)
		require.NoError(t, err)
		answer.FetchedAt = &now
		require.NoError(t, store.SaveVisibilityResult(ctx, run.ID, 2, answer))
	}
	require.NoError(t, store.FinishVisibilityAttempt(ctx, run.ID, 2))
	finished, pending, err := store.BeginVisibilityAttempt(ctx, run.ID, *run.RiverJobID, 3)
	require.NoError(t, err)
	require.Nil(t, finished)
	require.Nil(t, pending)
	status, err = querier.Get(ctx, brand.ID, run.ID)
	require.NoError(t, err)
	require.Equal(t, koidomain.VisibilitySucceeded, status.State)
	require.EqualValues(t, 8, status.CompletedResults)
	require.NotNil(t, status.FinishedAt)
	history, err := querier.List(ctx, brand.ID, koidomain.PageParams{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.EqualValues(t, 8, history.Items[0].CompletedResults)
	page, err = querier.ListResults(ctx, brand.ID, run.ID, pageParams)
	require.NoError(t, err)
	require.Equal(t, originalAnswer, page.Items[0].RawAnswer)
	require.WithinDuration(t, now, *page.Items[0].FetchedAt, time.Microsecond)
	outcomes := map[koidomain.Outcome]int{}
	for _, result := range page.Items {
		require.NotNil(t, result.FetchedAt)
		require.NoError(t, result.Validate())
		outcomes[*result.Outcome]++
	}
	require.Len(t, outcomes, 4)
	firstID, firstText := page.Items[0].PromptID, page.Items[0].PromptText
	text := "Edited later?"
	_, err = koidomain.NewPromptCommander(store).Update(ctx, brand.ID, firstID, koidomain.PromptPatch{Text: &text})
	require.NoError(t, err)
	require.NoError(t, koidomain.NewPromptCommander(store).Delete(ctx, brand.ID, firstID))
	page, err = querier.ListResults(ctx, brand.ID, run.ID, koidomain.PageParams{Page: 1, PageSize: 4})
	require.NoError(t, err)
	require.Len(t, page.Items, 4)
	require.True(t, page.HasNext)
	require.Equal(t, firstText, page.Items[0].PromptText)
	other := &koidomain.User{Email: uuid.NewString() + "@example.com", PasswordHash: "hash"}
	require.NoError(t, store.UserRepo().Create(t.Context(), other))
	otherCtx := koiauth.WithIdentity(t.Context(), koiauth.Identity{UserID: other.ID})
	_, err = commander.Create(otherCtx, brand.ID)
	require.ErrorIs(t, err, koidomain.ErrNotFound)
	_, err = querier.Get(otherCtx, brand.ID, run.ID)
	require.ErrorIs(t, err, koidomain.ErrNotFound)
	_, err = querier.List(otherCtx, brand.ID, pageParams)
	require.ErrorIs(t, err, koidomain.ErrNotFound)
	_, err = querier.ListResults(otherCtx, brand.ID, run.ID, pageParams)
	require.ErrorIs(t, err, koidomain.ErrNotFound)
	otherBrand := &koidomain.Brand{Name: "Other", Domain: "other.example", Country: "US"}
	require.NoError(t, store.BrandRepo().Create(otherCtx, otherBrand))
	_, err = querier.ListResults(otherCtx, otherBrand.ID, run.ID, pageParams)
	require.ErrorIs(t, err, koidomain.ErrNotFound)
	lost, err := commander.Create(ctx, brand.ID)
	require.NoError(t, err)
	require.NotEqual(t, run.ID, lost.ID)
	require.NoError(t, db.Table("river_job").Where("id = ?", *lost.RiverJobID).Updates(map[string]any{
		"state": "discarded", "finalized_at": now,
	}).Error)
	require.NoError(t, store.ReconcileVisibilityRuns(ctx))
	status, err = querier.Get(ctx, brand.ID, lost.ID)
	require.NoError(t, err)
	require.Equal(t, koidomain.VisibilityFailed, status.State)
	require.Equal(t, "job_lost", *status.ErrorCode)
	require.NoError(t, db.Delete(&koidomain.Brand{}, "id = ?", brand.ID).Error)
	var count int64
	require.NoError(t, db.Model(&koidomain.VisibilityResult{}).Count(&count).Error)
	require.Zero(t, count)
}
