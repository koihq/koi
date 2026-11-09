package database

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/domain"
)

func firstLookBrand(t *testing.T, store *Store, site string) (context.Context, *domain.Brand) {
	t.Helper()

	user := &domain.User{Email: uuid.NewString() + "@example.com", PasswordHash: "hash"}
	require.NoError(t, store.UserRepo().Create(t.Context(), user))

	organizationID := testOrganizationForUser(t, store, user.ID)
	ctx := testIdentity(t, store, user.ID)

	brand := &domain.Brand{OrganizationID: organizationID, Name: "Brand", Domain: site, Country: "US"}
	require.NoError(t, store.BrandRepo().Create(ctx, brand))

	_, err := domain.NewPromptCommander(store).Create(ctx, brand.ID, domain.CreatePromptParams{Text: "Best tools?"})
	require.NoError(t, err)

	return ctx, brand
}

func activateFirstLookOrganization(t *testing.T, db *gorm.DB, organizationID uuid.UUID) {
	t.Helper()

	now := time.Now().UTC()
	end := now.Add(30 * 24 * time.Hour)

	require.NoError(t, db.Model(&domain.Organization{}).
		Where("id = ?", organizationID).Update("trial_used_at", now).Error)
	require.NoError(t, db.Create(&domain.BillingSubscription{
		OrganizationID:       organizationID,
		StripeSubscriptionID: "sub_" + uuid.NewString(),
		StripePriceID:        "price_test",
		Quantity:             25,
		Status:               "active",
		CurrentPeriodEnd:     &end,
	}).Error)
}

func blockedReason(t *testing.T, store *Store, ctx context.Context, brandID uuid.UUID) string {
	t.Helper()

	status, err := store.VisibilityRepo().Tracking(ctx, brandID)
	require.NoError(t, err)
	if status.BlockedReason == nil {
		return ""
	}

	return *status.BlockedReason
}

func TestFirstLookRunsOnceAndResumesAfterSubscribing(t *testing.T) {
	db, store, ctx, brand := liveVisibilityFixture(t, true)
	store.WithCloudMode(true)

	// A new organization reaches the product and its draft before any Checkout.
	allowed, err := store.BillingRepo().ProductAccess(ctx, brand.OrganizationID)
	require.NoError(t, err)
	require.True(t, allowed)

	budget, err := cloudGenerationBudget(ctx, db, brand.ID)
	require.NoError(t, err)
	require.Equal(t, domain.FirstLookPromptLimit-1, budget)

	run := startVisibilityTracking(t, store, ctx, brand.ID)
	require.True(t, run.FastStart)
	require.Equal(t, domain.TrackingBlockedFirstLookUsed, blockedReason(t, store, ctx, brand.ID))

	collect, err := cloudCollectionAllowed(ctx, db, brand.OrganizationID, true)
	require.NoError(t, err)
	require.True(t, collect)

	collect, err = cloudCollectionAllowed(ctx, db, brand.OrganizationID, false)
	require.NoError(t, err)
	require.False(t, collect)

	// Once it returned an answer, the next daily slot spends nothing and the results stay readable.
	require.NoError(t, db.Model(&domain.VisibilityResult{}).
		Where("run_id = ?", run.ID).Update("outcome", domain.OutcomeMiss).Error)
	require.NoError(t, db.Model(run).Update("state", domain.VisibilitySucceeded).Error)

	status, err := store.VisibilityRepo().Tracking(ctx, brand.ID)
	require.NoError(t, err)
	anchor := *status.StartedAt
	require.NoError(t, store.DispatchVisibility(ctx, anchor.Add(24*time.Hour)))

	var runs int64
	require.NoError(t, db.Model(&domain.VisibilityRun{}).Where("brand_id = ?", brand.ID).Count(&runs).Error)
	require.Equal(t, int64(1), runs)

	allowed, err = store.BillingRepo().ProductAccess(ctx, brand.OrganizationID)
	require.NoError(t, err)
	require.True(t, allowed)

	// Subscribing resumes collection on the next slot without any other step.
	activateFirstLookOrganization(t, db, brand.OrganizationID)
	require.Empty(t, blockedReason(t, store, ctx, brand.ID))
	require.NoError(t, store.DispatchVisibility(ctx, anchor.Add(48*time.Hour)))
	require.NoError(t, db.Model(&domain.VisibilityRun{}).Where("brand_id = ?", brand.ID).Count(&runs).Error)
	require.Equal(t, int64(2), runs)
}

func TestFirstLookEndsWithAnExpiredOrCanceledTrial(t *testing.T) {
	db, store, ctx, brand := liveVisibilityFixture(t, true)
	store.WithCloudMode(true)

	require.NoError(t, db.Model(&domain.Organization{}).
		Where("id = ?", brand.OrganizationID).Update("trial_used_at", time.Now().UTC()).Error)

	allowed, err := store.BillingRepo().ProductAccess(ctx, brand.OrganizationID)
	require.NoError(t, err)
	require.False(t, allowed)

	_, err = cloudGenerationBudget(ctx, db, brand.ID)
	require.ErrorIs(t, err, domain.ErrBillingRequired)
}

func TestFirstLookAllowsThreeOrganizationsPerSite(t *testing.T) {
	_, store, ctx, brand := liveVisibilityFixture(t, true)
	store.WithCloudMode(true)
	startVisibilityTracking(t, store, ctx, brand.ID)

	// Colleagues who sign up on their own can still run it.
	for range domain.FirstLookSiteLimit - 1 {
		colleagueCtx, colleague := firstLookBrand(t, store, "koi.dev")
		startVisibilityTracking(t, store, colleagueCtx, colleague.ID)
	}

	otherCtx, other := firstLookBrand(t, store, "KOI.dev")
	require.NoError(t, store.VisibilityRepo().StartTracking(otherCtx, other.ID))

	_, err := store.VisibilityRepo().Active(otherCtx, other.ID)
	require.ErrorIs(t, err, domain.ErrNotFound)
	require.Equal(t, domain.TrackingBlockedFirstLookSiteUsed, blockedReason(t, store, otherCtx, other.ID))

	// The trial still works for that organization, and collection starts by itself.
	db := store.db
	activateFirstLookOrganization(t, db, other.OrganizationID)
	require.NoError(t, store.DispatchVisibility(otherCtx, time.Now()))

	_, err = store.VisibilityRepo().Active(otherCtx, other.ID)
	require.NoError(t, err)
}

func TestFirstLookStopsAtTheDailyLimit(t *testing.T) {
	_, store, ctx, brand := liveVisibilityFixture(t, true)
	store.WithCloudMode(true).WithFirstLookDailyLimit(1)
	startVisibilityTracking(t, store, ctx, brand.ID)

	otherCtx, other := firstLookBrand(t, store, "other.dev")
	require.NoError(t, store.VisibilityRepo().StartTracking(otherCtx, other.ID))

	_, err := store.VisibilityRepo().Active(otherCtx, other.ID)
	require.ErrorIs(t, err, domain.ErrNotFound)
	require.Equal(t, domain.TrackingBlockedFirstLookPaused, blockedReason(t, store, otherCtx, other.ID))
}

func TestStartTrackingAsOwnerRunsWithoutARequestIdentity(t *testing.T) {
	_, store, ctx, brand := liveVisibilityFixture(t, true)
	store.WithCloudMode(true)

	require.NoError(t, store.StartTrackingAsOwner(t.Context(), brand.ID))

	run, err := store.VisibilityRepo().Active(ctx, brand.ID)
	require.NoError(t, err)
	require.True(t, run.FastStart)

	// The page may retry the same start without creating another run.
	require.NoError(t, store.VisibilityRepo().StartTracking(ctx, brand.ID))

	var runs int64
	require.NoError(t, store.db.Model(&domain.VisibilityRun{}).Where("brand_id = ?", brand.ID).Count(&runs).Error)
	require.Equal(t, int64(1), runs)
}

func TestGenerationKeepsWhatResearchUnderstood(t *testing.T) {
	_, store, ctx, brand := liveVisibilityFixture(t, true)

	run, _, err := domain.NewGenerationCommander(store, true).Create(ctx, brand.ID, nil, true)
	require.NoError(t, err)
	require.True(t, run.StartTracking)
	require.Nil(t, run.Summary)

	summary := domain.GenerationSummary{
		Product:   "Koi",
		Category:  "AI visibility tracking",
		Audience:  "Founders",
		Functions: []string{"Tracks mentions"},
	}
	require.NoError(t, store.SaveGenerationSummary(ctx, run.ID, summary))

	saved, err := store.GenerationRepo().Get(ctx, run.ID)
	require.NoError(t, err)
	require.Equal(t, &summary, saved.Summary)
	require.Equal(t, brand.Name, saved.BrandSnapshot.Name)
}

func TestFirstCheckEmailCountsOnlyRealAnswers(t *testing.T) {
	_, store, ctx, brand := liveVisibilityFixture(t, true)
	run := startVisibilityTracking(t, store, ctx, brand.ID)

	to, name, answers, err := store.FirstCheckEmail(t.Context(), run.ID)
	require.NoError(t, err)
	require.Contains(t, to, "@example.com")
	require.Equal(t, brand.Name, name)
	require.Zero(t, answers)

	outcome := domain.OutcomeMiss
	failed := domain.OutcomeError
	require.NoError(t, store.db.Model(&domain.VisibilityResult{}).
		Where("run_id = ? AND engine = ?", run.ID, domain.EngineChatGPT).Update("outcome", outcome).Error)
	require.NoError(t, store.db.Model(&domain.VisibilityResult{}).
		Where("run_id = ? AND engine = ?", run.ID, domain.EngineGemini).Update("outcome", failed).Error)

	_, _, answers, err = store.FirstCheckEmail(t.Context(), run.ID)
	require.NoError(t, err)
	require.Equal(t, 1, answers)
}

func TestFirstLookWithoutARealAnswerDoesNotUseUpTheSite(t *testing.T) {
	db, store, ctx, brand := liveVisibilityFixture(t, true)
	store.WithCloudMode(true)

	// Each earlier organization ran a check on the site, and it ended with the given result.
	finished := func(runCtx context.Context, brandID uuid.UUID, state domain.VisibilityState, outcome domain.Outcome) {
		t.Helper()

		run := startVisibilityTracking(t, store, runCtx, brandID)

		require.NoError(t, db.Model(&domain.VisibilityResult{}).
			Where("run_id = ?", run.ID).Update("outcome", outcome).Error)
		require.NoError(t, db.Model(&domain.VisibilityRun{}).
			Where("id = ?", run.ID).Update("state", state).Error)
	}

	finished(ctx, brand.ID, domain.VisibilityFailed, domain.OutcomeError)

	for range domain.FirstLookSiteLimit - 1 {
		failedCtx, failed := firstLookBrand(t, store, "koi.dev")
		finished(failedCtx, failed.ID, domain.VisibilitySucceeded, domain.OutcomeError)
	}

	nextCtx, next := firstLookBrand(t, store, "koi.dev")
	require.NoError(t, store.VisibilityRepo().StartTracking(nextCtx, next.ID))
	require.NotEqual(t, domain.TrackingBlockedFirstLookSiteUsed, blockedReason(t, store, nextCtx, next.ID))

	// A failed run that still returned an answer did give something, so it counts.
	var active domain.VisibilityRun
	require.NoError(t, db.Where("brand_id = ?", next.ID).Take(&active).Error)
	require.NoError(t, db.Model(&domain.VisibilityResult{}).
		Where("run_id = ?", active.ID).Update("outcome", domain.OutcomeMiss).Error)
	require.NoError(t, db.Model(&active).Update("state", domain.VisibilityFailed).Error)

	for range domain.FirstLookSiteLimit - 1 {
		answeredCtx, answered := firstLookBrand(t, store, "koi.dev")
		finished(answeredCtx, answered.ID, domain.VisibilitySucceeded, domain.OutcomeCited)
	}

	lastCtx, last := firstLookBrand(t, store, "koi.dev")
	require.NoError(t, store.VisibilityRepo().StartTracking(lastCtx, last.ID))
	require.Equal(t, domain.TrackingBlockedFirstLookSiteUsed, blockedReason(t, store, lastCtx, last.ID))
}

func TestFirstLookRetriesARunWithoutAnswersOnTheNextSlot(t *testing.T) {
	db, store, ctx, brand := liveVisibilityFixture(t, true)
	store.WithCloudMode(true)

	run := startVisibilityTracking(t, store, ctx, brand.ID)

	status, err := store.VisibilityRepo().Tracking(ctx, brand.ID)
	require.NoError(t, err)
	anchor := *status.StartedAt

	fail := func(run *domain.VisibilityRun) {
		t.Helper()

		require.NoError(t, db.Model(&domain.VisibilityResult{}).
			Where("run_id = ?", run.ID).Update("outcome", domain.OutcomeError).Error)
		require.NoError(t, db.Model(run).Update("state", domain.VisibilityFailed).Error)
	}

	runCount := func() int64 {
		t.Helper()

		var count int64
		require.NoError(t, db.Model(&domain.VisibilityRun{}).Where("brand_id = ?", brand.ID).Count(&count).Error)

		return count
	}

	fail(run)
	require.Empty(t, blockedReason(t, store, ctx, brand.ID))

	// Each retry may collect, which the worker only allows for a fast start.
	for attempt := int64(2); attempt <= domain.FirstLookRunAttempts; attempt++ {
		require.NoError(t, store.DispatchVisibility(ctx, anchor.Add(time.Duration(attempt-1)*24*time.Hour)))
		require.Equal(t, attempt, runCount())

		retry, err := store.VisibilityRepo().Active(ctx, brand.ID)
		require.NoError(t, err)
		require.True(t, retry.FastStart)

		collect, err := cloudCollectionAllowed(ctx, db, brand.OrganizationID, retry.FastStart)
		require.NoError(t, err)
		require.True(t, collect)

		fail(retry)
	}

	require.Equal(t, domain.TrackingBlockedFirstLookUsed, blockedReason(t, store, ctx, brand.ID))
	require.NoError(t, store.DispatchVisibility(ctx, anchor.Add(time.Duration(domain.FirstLookRunAttempts)*24*time.Hour)))
	require.Equal(t, int64(domain.FirstLookRunAttempts), runCount())
}

func TestFirstLookStopsRetryingOnceARunAnswers(t *testing.T) {
	db, store, ctx, brand := liveVisibilityFixture(t, true)
	store.WithCloudMode(true)

	run := startVisibilityTracking(t, store, ctx, brand.ID)

	// Some engines failing still leaves something to show, so the free check is spent.
	var result domain.VisibilityResult
	require.NoError(t, db.Where("run_id = ?", run.ID).Take(&result).Error)
	require.NoError(t, db.Model(&result).Update("outcome", domain.OutcomeMiss).Error)
	require.NoError(t, db.Model(run).Update("state", domain.VisibilityFailed).Error)

	require.Equal(t, domain.TrackingBlockedFirstLookUsed, blockedReason(t, store, ctx, brand.ID))
}
