package koidatabase

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/internal/testdb"
	"github.com/koihq/koi/pkg/koiauth"
	"github.com/koihq/koi/pkg/koidomain"
)

func TestCloudPromptAllowanceIsOrganizationWideAndIgnoresInactivePrompts(t *testing.T) {
	db, err := Open(testdb.URL(t))
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	require.NoError(t, Migrate(db))

	store := NewStore(db).WithCloudMode(true)
	user := &koidomain.User{Email: uuid.NewString() + "@example.com", PasswordHash: "hash"}
	require.NoError(t, store.UserRepo().Create(t.Context(), user))
	organization := &koidomain.Organization{Name: "Koi"}
	require.NoError(t, store.OrganizationRepo().Create(t.Context(), organization))
	require.NoError(t, store.MembershipRepo().Create(t.Context(), &koidomain.Membership{
		UserID: user.ID, OrganizationID: organization.ID, Role: koidomain.RoleOwner,
	}))
	periodEnd := time.Now().Add(30 * 24 * time.Hour)
	require.NoError(t, db.Create(&koidomain.BillingSubscription{
		OrganizationID:       organization.ID,
		StripeSubscriptionID: "sub_123",
		StripePriceID:        "price_123",
		PriceVersion:         "v1",
		Quantity:             5,
		Status:               "active",
		CurrentPeriodEnd:     &periodEnd,
	}).Error)

	ctx := koiauth.WithIdentity(t.Context(), koiauth.Identity{UserID: user.ID})
	brands := make([]*koidomain.Brand, 2)
	for i := range brands {
		brands[i] = &koidomain.Brand{OrganizationID: organization.ID, Name: fmt.Sprintf("Brand %d", i), Domain: fmt.Sprintf("brand%d.example", i), Country: "IT"}
		require.NoError(t, store.BrandRepo().Create(ctx, brands[i]))
	}
	for i := range 5 {
		prompt := &koidomain.Prompt{BrandID: brands[i%2].ID, Text: fmt.Sprintf("Prompt %d", i), Source: koidomain.PromptSourceManual, Confirmed: true}
		require.NoError(t, store.PromptRepo().Create(ctx, prompt))
	}

	extra := &koidomain.Prompt{BrandID: brands[1].ID, Text: "Extra", Source: koidomain.PromptSourceManual, Confirmed: true}
	require.ErrorIs(t, store.PromptRepo().Create(ctx, extra), koidomain.ErrPromptLimitReached)
	extra.Inactive = true
	require.NoError(t, store.PromptRepo().Create(ctx, extra))
	active := false
	require.ErrorIs(t, store.PromptRepo().Update(ctx, koidomain.PromptFilter{ID: extra.ID, BrandID: brands[1].ID}, koidomain.PromptUpdate{Inactive: &active}), koidomain.ErrPromptLimitReached)

	first, err := store.PromptRepo().Get(ctx, koidomain.PromptFilter{BrandID: brands[0].ID, Query: "Prompt 0"})
	require.NoError(t, err)
	inactive := true
	require.NoError(t, store.PromptRepo().Update(ctx, koidomain.PromptFilter{ID: first.ID, BrandID: brands[0].ID}, koidomain.PromptUpdate{Inactive: &inactive}))
	require.NoError(t, store.PromptRepo().Update(ctx, koidomain.PromptFilter{ID: extra.ID, BrandID: brands[1].ID}, koidomain.PromptUpdate{Inactive: &active}))
}

func TestCloudWorkerStopsQueuedPromptAfterDeactivation(t *testing.T) {
	db, ordinaryStore, ctx, brand := liveVisibilityFixture(t, true)
	periodEnd := time.Now().Add(30 * 24 * time.Hour)
	require.NoError(t, db.Create(&koidomain.BillingSubscription{
		OrganizationID:       brand.OrganizationID,
		StripeSubscriptionID: "sub_worker",
		StripePriceID:        "price_worker",
		PriceVersion:         "v1",
		Quantity:             5,
		Status:               "active",
		CurrentPeriodEnd:     &periodEnd,
	}).Error)

	run := finishVisibilityReviews(t, ordinaryStore, ctx, brand.ID)
	cloudStore := NewStore(db, ordinaryStore.enqueueClient).WithCloudMode(true)
	_, cells, err := cloudStore.BeginVisibilityAttempt(ctx, run.ID, *run.RiverJobID, 1)
	require.NoError(t, err)
	require.NotEmpty(t, cells)

	inactive := true
	require.NoError(t, cloudStore.PromptRepo().Update(ctx,
		koidomain.PromptFilter{ID: cells[0].PromptID, BrandID: brand.ID},
		koidomain.PromptUpdate{Inactive: &inactive},
	))
	require.ErrorIs(t,
		cloudStore.ContactVisibilityTask(ctx, run.ID, 1, &cells[0], time.Now()),
		koidomain.ErrPromptLimitReached,
	)
}
