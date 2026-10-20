package koidatabase

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/internal/testdb"
	"github.com/koihq/koi/pkg/koidomain"
)

func TestBillingWebhookIdentityAndReplay(t *testing.T) {
	db, err := Open(testdb.URL(t))
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	require.NoError(t, Migrate(db))

	repo := NewBillingRepository(db)
	organization := &koidomain.Organization{Name: "Koi"}
	require.NoError(t, db.Create(organization).Error)
	customerID, err := repo.SaveCustomer(t.Context(), organization.ID, "cus_123")
	require.NoError(t, err)
	require.Equal(t, "cus_123", customerID)
	reservation, err := repo.ReserveCheckout(t.Context(), organization.ID, "cus_123", "price_123", 10)
	require.NoError(t, err)
	_, err = repo.ReserveCheckout(t.Context(), organization.ID, "cus_123", "price_123", 15)
	require.ErrorIs(t, err, koidomain.ErrBillingCheckoutPending)
	require.NoError(t, repo.FinalizeCheckout(t.Context(), reservation.ID, "cs_123", "https://checkout.stripe.com/example", time.Now().Add(24*time.Hour)))
	again, err := repo.ReserveCheckout(t.Context(), organization.ID, "cus_123", "price_123", 10)
	require.NoError(t, err)
	require.Equal(t, reservation.ID, again.ID)

	require.Error(t, repo.CompleteCheckout(t.Context(), "evt_bad", "cs_123", "cus_other", organization.ID.String(), "sub_123"))
	require.NoError(t, repo.CompleteCheckout(t.Context(), "evt_checkout", "cs_123", "cus_123", organization.ID.String(), "sub_123"))
	require.NoError(t, repo.CompleteCheckout(t.Context(), "evt_checkout", "cs_123", "cus_123", organization.ID.String(), "sub_123"))

	periodEnd := time.Now().Add(30 * 24 * time.Hour)
	require.Error(t, repo.SyncSubscription(t.Context(), "evt_bad_quantity", organization.ID.String(), "cus_123", "sub_123", "price_123", "active", 6, &periodEnd, nil, false))
	require.Error(t, repo.SyncSubscription(t.Context(), "evt_bad_sub", organization.ID.String(), "cus_other", "sub_123", "price_123", "active", 10, &periodEnd, nil, false))
	require.NoError(t, repo.SyncSubscription(t.Context(), "evt_sub", organization.ID.String(), "cus_123", "sub_123", "price_123", "active", 10, &periodEnd, nil, false))
	require.NoError(t, repo.SyncSubscription(t.Context(), "evt_sub", organization.ID.String(), "cus_123", "sub_123", "price_123", "active", 10, &periodEnd, nil, false))

	subscription, err := repo.Subscription(t.Context(), organization.ID)
	require.NoError(t, err)
	require.Equal(t, 10, subscription.PromptLimit(time.Now()))

	var eventCount int64
	require.NoError(t, db.Model(&koidomain.StripeWebhookEvent{}).Count(&eventCount).Error)
	require.EqualValues(t, 2, eventCount)
}

func TestBillingConcurrentCheckoutReservationReusesOneSession(t *testing.T) {
	db, err := Open(testdb.URL(t))
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	require.NoError(t, Migrate(db))

	repo := NewBillingRepository(db)
	organization := &koidomain.Organization{Name: "Koi"}
	require.NoError(t, db.Create(organization).Error)
	_, err = repo.SaveCustomer(t.Context(), organization.ID, "cus_parallel")
	require.NoError(t, err)

	type result struct {
		checkout *koidomain.BillingCheckoutSession
		err      error
	}
	results := make(chan result, 2)
	var workers sync.WaitGroup
	for range 2 {
		workers.Go(func() {
			checkout, err := repo.ReserveCheckout(t.Context(), organization.ID, "cus_parallel", "price_123", 10)
			results <- result{checkout: checkout, err: err}
		})
	}
	workers.Wait()
	close(results)

	var firstID string
	for item := range results {
		require.NoError(t, item.err)
		if firstID == "" {
			firstID = item.checkout.ID.String()
		} else {
			require.Equal(t, firstID, item.checkout.ID.String())
		}
	}
	var count int64
	require.NoError(t, db.Model(&koidomain.BillingCheckoutSession{}).Count(&count).Error)
	require.EqualValues(t, 1, count)
}
