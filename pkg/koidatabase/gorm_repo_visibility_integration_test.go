package koidatabase

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/koiauth"
	"github.com/koihq/koi/pkg/koidomain"
)

func TestVisibilityLargeMatrixEnqueueRollback(t *testing.T) {
	db, store, ctx, brand := liveVisibilityFixture(t, false)
	for i := range 100 {
		_, err := koidomain.NewPromptCommander(store).Create(ctx, brand.ID, koidomain.PromptParams{Text: fmt.Sprintf("Question %03d?", i)})
		require.NoError(t, err)
	}
	require.NoError(t, store.VisibilityRepo().FinishReview(ctx, brand.ID, "prompts"))
	// InsertTx fails after the run and all 404 cells have been inserted.
	require.Error(t, store.VisibilityRepo().FinishReview(ctx, brand.ID, "competitors"))
	for _, model := range []any{&koidomain.VisibilityRun{}, &koidomain.VisibilityResult{}} {
		var count int64
		require.NoError(t, db.Model(model).Count(&count).Error)
		require.Zero(t, count)
	}
}

func completedVisibilityCell(cell koidomain.VisibilityResult) koidomain.VisibilityResult {
	outcome := koidomain.OutcomeCited
	cell.Outcome = &outcome
	cell.RawAnswer = "Koi is recommended"
	cell.Sources = []string{"https://koi.dev"}
	return cell
}

func TestVisibilityMatrixSnapshotsRetryOwnershipAndReconciliation(t *testing.T) {
	db, store, ctx, brand := liveVisibilityFixture(t, true)

	for _, text := range []string{"Alternatives to Example Analytics?"} {
		_, err := koidomain.NewPromptCommander(store).Create(ctx, brand.ID, koidomain.PromptParams{Text: text})
		require.NoError(t, err)
	}

	require.NoError(t, store.VisibilityRepo().FinishReview(ctx, brand.ID, "prompts"))
	var wg sync.WaitGroup
	wg.Add(2)
	runs, errs := make([]*koidomain.VisibilityRun, 2), make([]error, 2)

	for i := range 2 {
		go func() {
			defer wg.Done()
			errs[i] = store.VisibilityRepo().FinishReview(ctx, brand.ID, "competitors")
			if errs[i] == nil {
				runs[i], errs[i] = store.VisibilityRepo().Active(ctx, brand.ID)
			}
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
		require.Equal(t, koidomain.CollectorCloro, result.Collector)
	}

	duplicate := page.Items[0]
	duplicate.ID = uuid.Nil
	requireConstraint(t, db.Create(&duplicate).Error, "idx_visibility_results_cell")
	duplicateRun := *run
	duplicateRun.ID = uuid.Nil
	requireConstraint(t, db.Create(&duplicateRun).Error, "idx_visibility_runs_active")
	duplicateRun.ID = uuid.Nil
	duplicateRun.State = koidomain.VisibilitySucceeded
	requireConstraint(t, db.Create(&duplicateRun).Error, "idx_visibility_runs_slot")
	_, pending, err := store.BeginVisibilityAttempt(ctx, run.ID, *run.RiverJobID, 1)
	require.NoError(t, err)
	require.Len(t, pending, 8)
	first := completedVisibilityCell(pending[0])
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
		answer := completedVisibilityCell(result)
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

	require.Equal(t, 8, outcomes[koidomain.OutcomeCited])
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
	err = store.VisibilityRepo().FinishReview(otherCtx, brand.ID, "prompts")
	require.ErrorIs(t, err, koidomain.ErrNotFound)
	_, err = querier.Get(otherCtx, brand.ID, run.ID)
	require.ErrorIs(t, err, koidomain.ErrNotFound)
	_, err = querier.List(otherCtx, brand.ID, pageParams)
	require.ErrorIs(t, err, koidomain.ErrNotFound)
	_, err = querier.ListResults(otherCtx, brand.ID, run.ID, pageParams)
	require.ErrorIs(t, err, koidomain.ErrNotFound)
	otherOrganizationID := testOrganizationForUser(t, store, other.ID)
	otherBrand := &koidomain.Brand{OrganizationID: otherOrganizationID, Name: "Other", Domain: "other.example", Country: "US"}
	require.NoError(t, store.BrandRepo().Create(otherCtx, otherBrand))
	_, err = querier.ListResults(otherCtx, otherBrand.ID, run.ID, pageParams)
	require.ErrorIs(t, err, koidomain.ErrNotFound)
	require.NoError(t, store.VisibilityRepo().FinishReview(ctx, brand.ID, "prompts"))
	tracking, err := store.VisibilityRepo().Tracking(ctx, brand.ID)
	require.NoError(t, err)
	require.NoError(t, store.DispatchVisibility(ctx, *tracking.NextRunAt))
	lost, err := store.VisibilityRepo().Active(ctx, brand.ID)
	require.NoError(t, err)
	require.NotEqual(t, run.ID, lost.ID)
	require.NoError(t, db.Table("river_job").Where("id = ?", *lost.RiverJobID).Updates(map[string]any{
		"state":        "discarded",
		"finalized_at": now,
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
