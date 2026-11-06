package database

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/internal/testdb"
	"github.com/koihq/koi/pkg/domain"
)

func TestCloudPromptAllowanceIsOrganizationWideAndIgnoresInactivePrompts(t *testing.T) {
	db, err := Open(testdb.URL(t))
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	require.NoError(t, Migrate(db))

	store := NewStore(db).WithCloudMode(true)
	user := &domain.User{Email: uuid.NewString() + "@example.com", PasswordHash: "hash"}
	require.NoError(t, store.UserRepo().Create(t.Context(), user))
	organization := &domain.Organization{Name: "Koi"}
	require.NoError(t, store.OrganizationRepo().Create(t.Context(), organization))
	require.NoError(t, store.MembershipRepo().Create(t.Context(), &domain.Membership{
		UserID: user.ID, OrganizationID: organization.ID, Role: domain.RoleOwner,
	}))
	periodEnd := time.Now().Add(30 * 24 * time.Hour)
	trialEnd := time.Now().Add(7 * 24 * time.Hour)
	require.NoError(t, db.Create(&domain.BillingSubscription{
		OrganizationID:       organization.ID,
		StripeSubscriptionID: "sub_123",
		StripePriceID:        "price_123",
		Quantity:             15,
		Status:               "trialing",
		CurrentPeriodEnd:     &periodEnd,
		TrialEnd:             &trialEnd,
	}).Error)

	ctx := testIdentity(t, store, user.ID)
	brands := make([]*domain.Brand, 2)
	for i := range brands {
		brands[i] = &domain.Brand{OrganizationID: organization.ID, Name: fmt.Sprintf("Brand %d", i), Domain: fmt.Sprintf("brand%d.example", i), Country: "IT"}
		require.NoError(t, store.BrandRepo().Create(ctx, brands[i]))
	}
	commander := domain.NewPromptCommander(store)
	for i := range domain.TrialPromptLimit {
		_, err := commander.Create(ctx, brands[i%2].ID, domain.CreatePromptParams{Text: fmt.Sprintf("Prompt %d", i)})
		require.NoError(t, err)
	}

	_, err = commander.Create(ctx, brands[1].ID, domain.CreatePromptParams{Text: "Extra"})
	require.ErrorIs(t, err, domain.ErrPromptLimitReached)
	extra := &domain.Prompt{BrandID: brands[1].ID, Text: "Extra", Source: domain.PromptSourceManual, Confirmed: true, Inactive: true}
	require.NoError(t, store.PromptRepo().Create(ctx, extra))
	active := false
	_, err = commander.Update(ctx, extra.ID, domain.UpdatePromptParams{Inactive: &active})
	require.ErrorIs(t, err, domain.ErrPromptLimitReached)

	firstPage, err := store.PromptRepo().List(ctx, domain.PromptFilter{BrandFilter: domain.BrandFilter{BrandID: brands[0].ID}, Query: "Prompt 0"}, domain.PageParams{Page: 1, PageSize: 10})
	first := &firstPage.Items[0]
	require.NoError(t, err)
	inactive := true
	_, err = commander.Update(ctx, first.ID, domain.UpdatePromptParams{Inactive: &inactive})
	require.NoError(t, err)
	draft := &domain.Prompt{BrandID: brands[0].ID, Text: "Drafted prompt", Source: domain.PromptSourceDrafted}
	require.NoError(t, db.Create(draft).Error)
	_, err = commander.Update(ctx, extra.ID, domain.UpdatePromptParams{Inactive: &active})
	require.ErrorIs(t, err, domain.ErrPromptLimitReached)
	require.NoError(t, db.Delete(draft).Error)
	_, err = commander.Update(ctx, extra.ID, domain.UpdatePromptParams{Inactive: &active})
	require.NoError(t, err)
	require.NoError(t, db.Where("brand_id = ? AND text IN ?", brands[0].ID,
		[]string{"Prompt 0", "Prompt 2", "Prompt 4", "Prompt 6", "Prompt 8"}).Delete(&domain.Prompt{}).Error)
	require.NoError(t, db.Where("brand_id = ? AND text = ?", brands[1].ID, "Prompt 1").Delete(&domain.Prompt{}).Error)
	run := &domain.GenerationRun{
		BrandID:       brands[0].ID,
		State:         domain.GenerationRunning,
		Attempt:       1,
		ModelSlug:     domain.GenerationModelSlug,
		BrandSnapshot: brands[0].Snapshot(),
	}
	require.NoError(t, db.Create(run).Error)
	suggestions := make([]string, 15)
	for i := range suggestions {
		suggestions[i] = fmt.Sprintf("Generated prompt %d?", i)
	}
	inserted, _, err := store.FinishGenerationAttempt(ctx, run.ID, 1,
		domain.GenerationResult{Prompts: suggestions})
	require.NoError(t, err)
	require.Equal(t, 5, inserted)
	count, err := storedPromptCount(ctx, db, organization.ID)
	require.NoError(t, err)
	require.EqualValues(t, domain.TrialPromptLimit, count)

	used, err := store.BillingRepo().UsedPromptCount(ctx, organization.ID)
	require.NoError(t, err)
	require.Equal(t, count, used, "the reported usage is the count that enforces the allowance")

	_, err = store.BillingRepo().UsedPromptCount(ctx, uuid.New())
	require.ErrorIs(t, err, domain.ErrUnauthorized)
}

func TestCloudTrialInitialDraftLeavesRoomForTheCustomer(t *testing.T) {
	db, err := Open(testdb.URL(t))
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	require.NoError(t, Migrate(db))

	store := NewStore(db).WithCloudMode(true)
	organization := &domain.Organization{Name: "Koi"}
	require.NoError(t, store.OrganizationRepo().Create(t.Context(), organization))

	end := time.Now().Add(7 * 24 * time.Hour)
	subscription := &domain.BillingSubscription{
		OrganizationID:       organization.ID,
		StripeSubscriptionID: "sub_trial",
		StripePriceID:        "price_trial",
		Quantity:             100,
		Status:               "trialing",
		CurrentPeriodEnd:     &end,
		TrialEnd:             &end,
	}
	require.NoError(t, db.Create(subscription).Error)

	brand := &domain.Brand{OrganizationID: organization.ID, Name: "Brand", Domain: "brand.example", Country: "IT"}
	require.NoError(t, db.Create(brand).Error)

	suggestions := make([]string, 15)
	for i := range suggestions {
		suggestions[i] = fmt.Sprintf("Generated prompt %d?", i)
	}

	var job int64
	generate := func() (target, inserted int, err error) {
		job++
		run := &domain.GenerationRun{
			BrandID:       brand.ID,
			State:         domain.GenerationQueued,
			RiverJobID:    &job,
			ModelSlug:     domain.GenerationModelSlug,
			BrandSnapshot: brand.Snapshot(),
		}
		require.NoError(t, db.Create(run).Error)

		_, input, err := store.BeginGenerationAttempt(t.Context(), run.ID, job, 1)
		if err != nil {
			require.NoError(t, db.Delete(run).Error)

			return 0, 0, err
		}

		inserted, _, err = store.FinishGenerationAttempt(t.Context(), run.ID, 1, domain.GenerationResult{Prompts: suggestions})

		return input.PromptTarget, inserted, err
	}

	target, inserted, err := generate()
	require.NoError(t, err)
	require.Equal(t, domain.InitialDraftPromptLimit, target)
	require.Equal(t, domain.InitialDraftPromptLimit, inserted)

	// Only the budget left by the allowance is stored, here the rest of the trial's 10 prompts.
	target, inserted, err = generate()
	require.NoError(t, err)
	require.Equal(t, 5, target)
	require.Equal(t, 5, inserted)

	target, inserted, err = generate()
	require.NoError(t, err)
	require.Zero(t, target)
	require.Zero(t, inserted)

	require.NoError(t, db.Where("brand_id = ?", brand.ID).Delete(&domain.Prompt{}).Error)
	require.NoError(t, db.Model(subscription).Updates(map[string]any{"status": "active", "quantity": 15}).Error)

	target, inserted, err = generate()
	require.NoError(t, err)
	require.Equal(t, domain.InitialDraftPromptLimit, target, "a larger allowance gets the same draft")
	require.Equal(t, domain.InitialDraftPromptLimit, inserted)
}

func TestCloudWorkerStopsQueuedPromptAfterDeactivation(t *testing.T) {
	db, ordinaryStore, ctx, brand := liveVisibilityFixture(t, true)
	periodEnd := time.Now().Add(30 * 24 * time.Hour)
	require.NoError(t, db.Create(&domain.BillingSubscription{
		OrganizationID:       brand.OrganizationID,
		StripeSubscriptionID: "sub_worker",
		StripePriceID:        "price_worker",
		Quantity:             5,
		Status:               "active",
		CurrentPeriodEnd:     &periodEnd,
	}).Error)

	run := startVisibilityTracking(t, ordinaryStore, ctx, brand.ID)
	cloudStore := NewStore(db).WithEnqueueClient(ordinaryStore.enqueueClient).WithCloudMode(true)
	_, cells, err := cloudStore.BeginVisibilityAttempt(ctx, run.ID, *run.RiverJobID, 1)
	require.NoError(t, err)
	require.NotEmpty(t, cells)

	inactive := true
	_, err = domain.NewPromptCommander(cloudStore).Update(ctx, cells[0].PromptID, domain.UpdatePromptParams{Inactive: &inactive})
	require.NoError(t, err)
	require.ErrorIs(t,
		cloudStore.ContactVisibilityTask(ctx, run.ID, 1, &cells[0], time.Now()),
		domain.ErrPromptLimitReached,
	)
}
