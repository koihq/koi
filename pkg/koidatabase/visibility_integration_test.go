package koidatabase

import (
	"context"
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
)

func liveVisibilityFixture(t *testing.T, riverTables bool) (*gorm.DB, *Store, context.Context, *koidomain.Brand) {
	t.Helper()
	db, err := Open(testdb.URL(t))
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	require.NoError(t, Migrate(db))
	client, err := river.NewClient(riverdatabasesql.New(sqlDB), &river.Config{})
	require.NoError(t, err)
	if riverTables {
		migrator, err := rivermigrate.New(riverdatabasesql.New(sqlDB), nil)
		require.NoError(t, err)
		_, err = migrator.Migrate(t.Context(), rivermigrate.DirectionUp, nil)
		require.NoError(t, err)
	}

	store := NewStore(db, client).WithLiveCollection(true)
	user := &koidomain.User{Email: uuid.NewString() + "@example.com", PasswordHash: "hash"}
	require.NoError(t, store.UserRepo().Create(t.Context(), user))
	organizationID := testOrganizationForUser(t, store, user.ID)
	ctx := koiauth.WithIdentity(t.Context(), koiauth.Identity{UserID: user.ID})
	brand := &koidomain.Brand{OrganizationID: organizationID, Name: "Koi", Domain: "koi.dev", Country: "US"}
	require.NoError(t, store.BrandRepo().Create(ctx, brand))
	_, err = koidomain.NewPromptCommander(store).Create(ctx, brand.ID, koidomain.PromptParams{Text: "Best analytics tools?"})
	require.NoError(t, err)

	return db, store, ctx, brand
}

func testOrganizationForUser(t *testing.T, store *Store, userID uuid.UUID) uuid.UUID {
	t.Helper()
	organization := &koidomain.Organization{Name: "Test"}
	require.NoError(t, store.OrganizationRepo().Create(t.Context(), organization))
	require.NoError(t, store.MembershipRepo().Create(t.Context(), &koidomain.Membership{
		UserID: userID, OrganizationID: organization.ID, Role: koidomain.RoleOwner,
	}))

	return organization.ID
}

func finishVisibilityReviews(t *testing.T, store *Store, ctx context.Context, brandID uuid.UUID) *koidomain.VisibilityRun {
	t.Helper()
	require.NoError(t, store.VisibilityRepo().FinishReview(ctx, brandID, "prompts"))
	require.NoError(t, store.VisibilityRepo().FinishReview(ctx, brandID, "competitors"))
	run, err := store.VisibilityRepo().Active(ctx, brandID)
	require.NoError(t, err)

	return run
}

func TestVisibilityReviewEnqueueRollback(t *testing.T) {
	db, store, ctx, brand := liveVisibilityFixture(t, false)
	require.NoError(t, store.VisibilityRepo().FinishReview(ctx, brand.ID, "prompts"))
	require.Error(t, store.VisibilityRepo().FinishReview(ctx, brand.ID, "competitors"))
	status, err := store.VisibilityRepo().Tracking(ctx, brand.ID)
	require.NoError(t, err)
	require.True(t, status.PromptsReviewed)
	require.False(t, status.CompetitorsReviewed)
	require.Nil(t, status.StartedAt)
	var count int64
	require.NoError(t, db.Model(&koidomain.VisibilityRun{}).Count(&count).Error)
	require.Zero(t, count)
	require.NoError(t, db.Model(&koidomain.VisibilityResult{}).Count(&count).Error)
	require.Zero(t, count)
}

func TestVisibilityScheduleAndReviewInvalidation(t *testing.T) {
	db, store, ctx, brand := liveVisibilityFixture(t, true)
	// Manual rows are already confirmed; an explicit review is still required.
	require.NoError(t, store.DispatchVisibility(ctx, time.Now()))
	_, err := store.VisibilityRepo().Active(ctx, brand.ID)
	require.ErrorIs(t, err, koidomain.ErrNotFound)
	run := finishVisibilityReviews(t, store, ctx, brand.ID)
	require.Equal(t, int64(4), run.TotalResults)
	status, err := store.VisibilityRepo().Tracking(ctx, brand.ID)
	require.NoError(t, err)
	require.Equal(t, status.StartedAt.Add(24*time.Hour), *status.NextRunAt)
	anchor := *status.StartedAt

	// Re-finishing reviews and early dispatch never enqueue an extra paid run.
	require.NoError(t, store.VisibilityRepo().FinishReview(ctx, brand.ID, "prompts"))
	require.NoError(t, store.DispatchVisibility(ctx, anchor.Add(23*time.Hour)))
	require.NoError(t, db.Model(run).Update("state", koidomain.VisibilitySucceeded).Error)

	// Two dispatchers race on the same due slot: the brand lock and unique index fence it.
	var group sync.WaitGroup
	errors := make(chan error, 2)
	for range 2 {
		group.Go(func() { errors <- store.DispatchVisibility(context.Background(), anchor.Add(24*time.Hour)) })
	}
	group.Wait()
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}
	runs, err := store.VisibilityRepo().List(ctx, brand.ID, koidomain.PageParams{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(2), runs.TotalItems)
	current, err := store.VisibilityRepo().Active(ctx, brand.ID)
	require.NoError(t, err)
	require.Equal(t, anchor.Add(24*time.Hour), *current.ScheduledFor)
	require.NoError(t, db.Model(current).Update("state", koidomain.VisibilitySucceeded).Error)

	// A prompt edit invalidates review, but preserves the current run's snapshot and anchor.
	prompts, err := store.PromptRepo().List(ctx, koidomain.PromptFilter{BrandID: brand.ID, Page: koidomain.PageParams{Page: 1, PageSize: 10}})
	require.NoError(t, err)
	text := "New question"
	require.NoError(t, store.PromptRepo().Update(ctx, koidomain.PromptFilter{BrandID: brand.ID, ID: prompts.Items[0].ID}, koidomain.PromptUpdate{Text: &text}))
	status, err = store.VisibilityRepo().Tracking(ctx, brand.ID)
	require.NoError(t, err)
	require.False(t, status.PromptsReviewed)
	require.Equal(t, anchor, *status.StartedAt)
	require.NoError(t, store.DispatchVisibility(ctx, anchor.Add(48*time.Hour)))
	require.NoError(t, store.VisibilityRepo().FinishReview(ctx, brand.ID, "prompts"))
	runs, err = store.VisibilityRepo().List(ctx, brand.ID, koidomain.PageParams{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(2), runs.TotalItems)

	// After downtime, collect the latest due slot only. Restarting the Store changes nothing.
	restarted := NewStore(db, store.enqueueClient).WithLiveCollection(true)
	require.NoError(t, restarted.DispatchVisibility(ctx, anchor.Add(6*24*time.Hour+time.Hour)))
	latest, err := restarted.VisibilityRepo().Active(ctx, brand.ID)
	require.NoError(t, err)
	require.Equal(t, anchor.Add(6*24*time.Hour), *latest.ScheduledFor)
	status, err = restarted.VisibilityRepo().Tracking(ctx, brand.ID)
	require.NoError(t, err)
	require.Equal(t, anchor.Add(7*24*time.Hour), *status.NextRunAt)
	cells, err := restarted.VisibilityRepo().ListResults(ctx, brand.ID, latest.ID, koidomain.PageParams{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, "New question", cells.Items[0].PromptText)
	old, err := restarted.VisibilityRepo().ListResults(ctx, brand.ID, run.ID, koidomain.PageParams{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, "Best analytics tools?", old.Items[0].PromptText)

	// Mutations from another owner fail before touching reviews or list rows.
	stranger := koiauth.WithIdentity(t.Context(), koiauth.Identity{UserID: uuid.New()})
	require.ErrorIs(t, store.VisibilityRepo().FinishReview(stranger, brand.ID, "prompts"), koidomain.ErrNotFound)
	require.ErrorIs(t, store.PromptRepo().Delete(stranger, koidomain.PromptFilter{BrandID: brand.ID, ID: prompts.Items[0].ID}), koidomain.ErrNotFound)
	_, err = store.VisibilityRepo().Get(stranger, brand.ID, run.ID)
	require.ErrorIs(t, err, koidomain.ErrNotFound)
}

func TestVisibilityCellsResumeAndPreserveAnswers(t *testing.T) {
	_, store, ctx, brand := liveVisibilityFixture(t, true)
	run := finishVisibilityReviews(t, store, ctx, brand.ID)
	_, cells, err := store.BeginVisibilityAttempt(ctx, run.ID, *run.RiverJobID, 1)
	require.NoError(t, err)
	cell := cells[0]
	now := time.Now().UTC()
	require.NoError(t, store.ContactVisibilityTask(ctx, run.ID, 1, &cell, now))
	taskID := uuid.NewString()
	require.NoError(t, store.SaveVisibilityTask(ctx, run.ID, 1, cell.ID, taskID))
	require.Error(t, store.SaveVisibilityTask(ctx, run.ID, 1, cell.ID, uuid.NewString()))
	_, resumed, err := store.BeginVisibilityAttempt(ctx, run.ID, *run.RiverJobID, 2)
	require.NoError(t, err)
	var saved koidomain.VisibilityResult
	for _, item := range resumed {
		if item.ID == cell.ID {
			saved = item
		}
	}
	require.Equal(t, taskID, *saved.CloroTaskID)
	require.WithinDuration(t, now, *saved.SubmittedAt, time.Microsecond)
	cell.RawAnswer = "Koi is recommended."
	cell.Sources = []string{"https://koi.dev"}
	outcome := koidomain.OutcomeCited
	cell.Outcome, cell.FetchedAt = &outcome, &now
	require.Error(t, store.SaveVisibilityResult(ctx, run.ID, 1, cell)) // stale attempt
	require.NoError(t, store.SaveVisibilityResult(ctx, run.ID, 2, cell))
	cell.RawAnswer = "overwrite"
	require.NoError(t, store.SaveVisibilityResult(ctx, run.ID, 2, cell))
	read, err := store.VisibilityRepo().ListResults(ctx, brand.ID, run.ID, koidomain.PageParams{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, "Koi is recommended.", read.Items[0].RawAnswer)
	require.Error(t, store.FinishVisibilityAttempt(ctx, run.ID, 2))
	progress, err := store.VisibilityRepo().Get(ctx, brand.ID, run.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), progress.CompletedResults)
}

func TestVisibilityDisabledReviewsAndEmptyPrompts(t *testing.T) {
	_, store, ctx, brand := liveVisibilityFixture(t, true)
	store.WithLiveCollection(false)
	require.NoError(t, store.VisibilityRepo().FinishReview(ctx, brand.ID, "competitors"))
	require.NoError(t, store.VisibilityRepo().FinishReview(ctx, brand.ID, "prompts"))
	status, err := store.VisibilityRepo().Tracking(ctx, brand.ID)
	require.NoError(t, err)
	require.True(t, status.PromptsReviewed && status.CompetitorsReviewed)
	require.Equal(t, "collection_unavailable", *status.BlockedReason)
	require.Nil(t, status.StartedAt)
	store.WithLiveCollection(true)
	require.NoError(t, store.DispatchVisibility(ctx, time.Now().UTC()))
	_, err = store.VisibilityRepo().Active(ctx, brand.ID)
	require.NoError(t, err)
	prompts, err := store.PromptRepo().List(ctx, koidomain.PromptFilter{BrandID: brand.ID, Page: koidomain.PageParams{Page: 1, PageSize: 10}})
	require.NoError(t, err)
	require.NoError(t, store.PromptRepo().Delete(ctx, koidomain.PromptFilter{BrandID: brand.ID, ID: prompts.Items[0].ID}))
	require.ErrorIs(t, store.VisibilityRepo().FinishReview(ctx, brand.ID, "prompts"), koidomain.ErrValidation)
}
