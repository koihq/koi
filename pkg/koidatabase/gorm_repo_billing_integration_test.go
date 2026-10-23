package koidatabase

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/koihq/koi/internal/testdb"
	"github.com/koihq/koi/pkg/koidomain"
)

type billingTestGateway struct {
	koidomain.BillingGateway
	mu               sync.Mutex
	snapshot         koidomain.BillingSnapshot
	sessions         map[string]koidomain.BillingCheckout
	createCalls      int
	upgradeCalls     int
	releaseCalls     int
	failAfterUpgrade bool
	pendingPayment   bool
	expireCompletes  bool
	beforeRead       func()
	priceError       error
}

func (g *billingTestGateway) ValidatePrice(context.Context, string) error { return g.priceError }
func (g *billingTestGateway) CreateCustomer(context.Context, string) (string, error) {
	return "cus_test", nil
}
func (g *billingTestGateway) CreateCheckout(_ context.Context, input koidomain.BillingCheckoutInput) (koidomain.BillingCheckout, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if session, ok := g.sessions[input.ReservationID]; ok {
		return session, nil
	}
	g.createCalls++
	session := koidomain.BillingCheckout{
		ID:             "cs_" + input.ReservationID,
		URL:            "https://checkout.stripe.com/" + input.ReservationID,
		CustomerID:     input.CustomerID,
		OrganizationID: input.OrganizationID,
		Status:         "open",
		ExpiresAt:      time.Now().Add(24 * time.Hour),
	}
	g.sessions[input.ReservationID] = session
	return session, nil
}
func (g *billingTestGateway) ExpireCheckout(_ context.Context, id string) (koidomain.BillingCheckout, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for key, session := range g.sessions {
		if session.ID != id {
			continue
		}
		if g.expireCompletes {
			session.Status, session.SubscriptionID = "complete", g.snapshot.ID
		} else {
			session.Status = "expired"
		}
		g.sessions[key] = session
		return session, nil
	}
	return koidomain.BillingCheckout{}, errors.New("missing session")
}
func (g *billingTestGateway) GetSubscription(context.Context, string) (koidomain.BillingSnapshot, error) {
	g.mu.Lock()
	snapshot, hook := g.snapshot, g.beforeRead
	g.mu.Unlock()
	if hook != nil {
		hook()
	}
	return snapshot, nil
}
func (g *billingTestGateway) Upgrade(_ context.Context, current koidomain.BillingSnapshot, quantity int, _ string) (koidomain.BillingSnapshot, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.upgradeCalls++
	if g.pendingPayment {
		expires := time.Now().Add(time.Hour)
		g.snapshot.PendingUpgradeQuantity = &quantity
		g.snapshot.PendingUpgradeExpiresAt = &expires
		g.snapshot.PaymentURL = "https://invoice.stripe.com/payment"
	} else {
		g.snapshot.Quantity = quantity
	}
	if g.failAfterUpgrade {
		g.failAfterUpgrade = false
		return koidomain.BillingSnapshot{}, errors.New("lost Stripe response")
	}
	return g.snapshot, nil
}
func (g *billingTestGateway) ChangeTrial(_ context.Context, _ koidomain.BillingSnapshot, quantity int, _ string) (koidomain.BillingSnapshot, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.snapshot.Quantity = quantity
	return g.snapshot, nil
}
func (g *billingTestGateway) ScheduleDowngrade(_ context.Context, _ koidomain.BillingSnapshot, quantity int, _ string) (koidomain.BillingSnapshot, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	id := "sub_sched_test"
	g.snapshot.ScheduleID, g.snapshot.PendingQuantity = &id, &quantity
	return g.snapshot, nil
}
func (g *billingTestGateway) ReleaseSchedule(_ context.Context, _ koidomain.BillingSnapshot, _ string) (koidomain.BillingSnapshot, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.releaseCalls++
	g.snapshot.ScheduleID, g.snapshot.PendingQuantity = nil, nil
	return g.snapshot, nil
}

type billingFixture struct {
	db           *gorm.DB
	store        *Store
	repo         *BillingRepository
	gateway      *billingTestGateway
	commander    koidomain.BillingCommander
	organization *koidomain.Organization
	user         *koidomain.User
	ctx          context.Context
}

func newBillingFixture(t *testing.T) *billingFixture {
	t.Helper()
	db, err := Open(testdb.URL(t))
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	require.NoError(t, Migrate(db))
	organization := &koidomain.Organization{Name: "Koi"}
	require.NoError(t, db.Create(organization).Error)
	user := &koidomain.User{Email: uuid.NewString() + "@example.com", PasswordHash: "hash"}
	require.NoError(t, db.Create(user).Error)
	require.NoError(t, db.Create(&koidomain.Membership{
		UserID:         user.ID,
		OrganizationID: organization.ID,
		Role:           koidomain.RoleOwner,
	}).Error)
	repo := NewBillingRepository(db)
	_, err = repo.SaveCustomer(t.Context(), organization.ID, "cus_test")
	require.NoError(t, err)
	end := time.Now().Add(30 * 24 * time.Hour)
	gateway := &billingTestGateway{
		snapshot: koidomain.BillingSnapshot{
			ID: "sub_test", CustomerID: "cus_test", OrganizationID: organization.ID.String(),
			ItemID: "si_test", PriceID: "price_test", Quantity: 25, Status: "active", CurrentPeriodEnd: &end,
		},
		sessions: make(map[string]koidomain.BillingCheckout),
	}
	store := NewStore(db)
	return &billingFixture{
		db: db, store: store, repo: repo, gateway: gateway, organization: organization, user: user,
		commander: koidomain.NewBillingCommander(store, gateway, "price_test", "https://koi.example"),
		ctx:       testIdentity(t, store, user.ID),
	}
}

func (f *billingFixture) complete(t *testing.T) {
	t.Helper()
	_, err := f.commander.Checkout(f.ctx, 25)
	require.NoError(t, err)
	var checkout koidomain.BillingCheckoutSession
	require.NoError(t, f.db.First(&checkout).Error)
	require.NoError(t, f.commander.ProcessEvent(f.ctx, koidomain.BillingEvent{
		ID: "evt_checkout", CheckoutID: checkout.SessionID, CustomerID: "cus_test",
		OrganizationID: f.organization.ID.String(), SubscriptionID: "sub_test",
	}))
}

func (f *billingFixture) event(t *testing.T, id string) {
	t.Helper()
	require.NoError(t, f.commander.ProcessEvent(f.ctx, koidomain.BillingEvent{
		ID: id, CustomerID: "cus_test", SubscriptionID: "sub_test",
	}))
}

func TestBillingWebhookIdentityReplayAndEntitlement(t *testing.T) {
	f := newBillingFixture(t)
	allowed, err := f.repo.ProductAccess(f.ctx, f.organization.ID)
	require.NoError(t, err)
	require.False(t, allowed)
	// A premature subscription event cannot grant access without a completed checkout.
	f.event(t, "evt_early")
	f.complete(t)
	allowed, err = f.repo.ProductAccess(f.ctx, f.organization.ID)
	require.NoError(t, err)
	require.True(t, allowed)

	f.gateway.snapshot.Quantity = 50
	f.event(t, "evt_upgrade")
	f.gateway.snapshot.Quantity = 100
	f.event(t, "evt_upgrade") // Duplicate IDs cannot reapply changed content.
	subscription, err := f.repo.Subscription(f.ctx, f.organization.ID)
	require.NoError(t, err)
	require.Equal(t, 50, subscription.Quantity)

	f.gateway.snapshot.CustomerID = "cus_wrong"
	require.Error(t, f.commander.ProcessEvent(f.ctx, koidomain.BillingEvent{ID: "evt_bad", CustomerID: "cus_test", SubscriptionID: "sub_test"}))
	f.gateway.snapshot.CustomerID = "cus_test"
	f.gateway.snapshot.Quantity = 6
	require.Error(t, f.commander.ProcessEvent(f.ctx, koidomain.BillingEvent{ID: "evt_bad", CustomerID: "cus_test", SubscriptionID: "sub_test"}))
	f.gateway.snapshot.Quantity = 100
	f.event(t, "evt_bad") // Failed transactions leave the event retryable.
	subscription, err = f.repo.Subscription(f.ctx, f.organization.ID)
	require.NoError(t, err)
	require.Equal(t, 100, subscription.Quantity)
	require.NoError(t, f.db.First(f.organization, "id = ?", f.organization.ID).Error)
	require.NotNil(t, f.organization.TrialUsedAt)
}

func TestBillingConcurrentCheckoutAndFinalization(t *testing.T) {
	f := newBillingFixture(t)
	var workers sync.WaitGroup
	urls := make(chan string, 2)
	errs := make(chan error, 2)
	for range 2 {
		workers.Go(func() {
			url, err := f.commander.Checkout(f.ctx, 25)
			urls <- url
			errs <- err
		})
	}
	workers.Wait()
	close(urls)
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	first := <-urls
	require.Equal(t, first, <-urls)
	require.Equal(t, 1, f.gateway.createCalls)
	var checkout koidomain.BillingCheckoutSession
	require.NoError(t, f.db.First(&checkout).Error)
	require.NoError(t, f.repo.FinalizeCheckout(f.ctx, checkout.ID, checkout.SessionID, checkout.URL, *checkout.ExpiresAt))
	require.Error(t, f.repo.FinalizeCheckout(f.ctx, checkout.ID, "cs_other", checkout.URL, *checkout.ExpiresAt))
}

func TestBillingCheckoutReplacementAndCompletionRace(t *testing.T) {
	for _, complete := range []bool{false, true} {
		t.Run(fmt.Sprint(complete), func(t *testing.T) {
			f := newBillingFixture(t)
			first, err := f.commander.Checkout(f.ctx, 25)
			require.NoError(t, err)
			f.gateway.expireCompletes = complete
			second, err := f.commander.Checkout(f.ctx, 50)
			if complete {
				require.ErrorIs(t, err, koidomain.ErrBillingSubscriptionExists)
				require.Equal(t, 1, f.gateway.createCalls)
				subscription, err := f.repo.Subscription(f.ctx, f.organization.ID)
				require.NoError(t, err)
				require.Equal(t, 25, subscription.Quantity)
			} else {
				require.NoError(t, err)
				require.NotEqual(t, first, second)
				require.Equal(t, 2, f.gateway.createCalls)
				var retired int64
				require.NoError(t, f.db.Model(&koidomain.BillingCheckoutSession{}).Where("retired").Count(&retired).Error)
				require.EqualValues(t, 1, retired)
			}
		})
	}
}

func TestBillingChangesPreserveTrialAndManageDowngrades(t *testing.T) {
	f := newBillingFixture(t)
	trialEnd := time.Now().Add(7 * 24 * time.Hour)
	f.gateway.snapshot.Status, f.gateway.snapshot.TrialEnd = "trialing", &trialEnd
	f.complete(t)
	result, err := f.commander.ChangeAllowance(f.ctx, 10, "trial-change")
	require.NoError(t, err)
	require.Equal(t, "trial_end", result.Effective)
	subscription, err := f.repo.Subscription(f.ctx, f.organization.ID)
	require.NoError(t, err)
	require.Equal(t, 5, subscription.PromptLimit(time.Now()))
	require.WithinDuration(t, trialEnd, *subscription.TrialEnd, time.Microsecond)
	require.Equal(t, 0, f.gateway.upgradeCalls)

	f.gateway.snapshot.Status, f.gateway.snapshot.TrialEnd = "active", nil
	f.gateway.snapshot.Quantity = 25
	f.event(t, "evt_trial_ended")
	for index, quantity := range []int{15, 10, 25, 5, 50} {
		_, err := f.commander.ChangeAllowance(f.ctx, quantity, fmt.Sprintf("change-%d", index))
		require.NoError(t, err)
	}
	subscription, err = f.repo.Subscription(f.ctx, f.organization.ID)
	require.NoError(t, err)
	require.Equal(t, 50, subscription.Quantity)
	require.Nil(t, subscription.PendingQuantity)
	require.Equal(t, 2, f.gateway.releaseCalls)
	require.Equal(t, 1, f.gateway.upgradeCalls)
}

func TestBillingUpgradeRetryAfterLostResponseAndConflictingKey(t *testing.T) {
	f := newBillingFixture(t)
	f.complete(t)
	f.gateway.failAfterUpgrade = true
	_, err := f.commander.ChangeAllowance(f.ctx, 50, "upgrade")
	require.ErrorContains(t, err, "lost Stripe response")
	result, err := f.commander.ChangeAllowance(f.ctx, 50, "upgrade")
	require.NoError(t, err)
	require.Equal(t, 50, result.Quantity)
	f.gateway.priceError = errors.New("Stripe is unavailable")
	replayed, err := f.commander.ChangeAllowance(f.ctx, 50, "upgrade")
	require.NoError(t, err)
	require.Equal(t, result, replayed)
	_, err = f.commander.ChangeAllowance(f.ctx, 100, "upgrade")
	require.ErrorIs(t, err, koidomain.ErrIdempotencyConflict)
	require.Equal(t, 1, f.gateway.upgradeCalls)
	subscription, err := f.repo.Subscription(f.ctx, f.organization.ID)
	require.NoError(t, err)
	require.Equal(t, 50, subscription.Quantity)
}

func TestBillingPendingPaymentDoesNotGrantUpgrade(t *testing.T) {
	f := newBillingFixture(t)
	f.complete(t)
	f.gateway.pendingPayment = true
	result, err := f.commander.ChangeAllowance(f.ctx, 50, "upgrade")
	require.NoError(t, err)
	require.Equal(t, "after_payment", result.Effective)
	require.NotEmpty(t, result.PaymentURL)
	require.NotNil(t, result.ExpiresAt)
	subscription, err := f.repo.Subscription(f.ctx, f.organization.ID)
	require.NoError(t, err)
	require.Equal(t, 25, subscription.PromptLimit(time.Now()))
	require.Equal(t, 50, *subscription.PendingUpgradeQuantity)
	_, err = f.commander.ChangeAllowance(f.ctx, 100, "other-change")
	require.ErrorIs(t, err, koidomain.ErrBillingConflict)
	f.gateway.snapshot.Quantity = 50
	f.gateway.snapshot.PendingUpgradeQuantity, f.gateway.snapshot.PendingUpgradeExpiresAt = nil, nil
	f.gateway.snapshot.PaymentURL = ""
	f.event(t, "evt_paid")
	subscription, err = f.repo.Subscription(f.ctx, f.organization.ID)
	require.NoError(t, err)
	require.Equal(t, 50, subscription.PromptLimit(time.Now()))
	require.Nil(t, subscription.PendingUpgradeQuantity)
	require.Empty(t, subscription.PaymentURL)
}

func TestBillingExpiredUpgradeClearsPendingState(t *testing.T) {
	f := newBillingFixture(t)
	f.complete(t)
	f.gateway.pendingPayment = true
	_, err := f.commander.ChangeAllowance(f.ctx, 50, "unpaid-upgrade")
	require.NoError(t, err)
	f.gateway.snapshot.PendingUpgradeQuantity = nil
	f.gateway.snapshot.PendingUpgradeExpiresAt = nil
	f.gateway.snapshot.PaymentURL = ""
	f.event(t, "evt_pending_update_expired")
	subscription, err := f.repo.Subscription(f.ctx, f.organization.ID)
	require.NoError(t, err)
	require.Equal(t, 25, subscription.PromptLimit(time.Now()))
	require.Nil(t, subscription.PendingUpgradeQuantity)
	require.Empty(t, subscription.PaymentURL)
}

func TestBillingOldSubscriptionEventCannotReplaceSuccessor(t *testing.T) {
	f := newBillingFixture(t)
	f.complete(t)
	f.gateway.snapshot.Status = "canceled"
	f.event(t, "evt_old_cancelled")
	f.gateway.snapshot.ID = "sub_successor"
	f.gateway.snapshot.Status = "active"
	_, err := f.commander.Checkout(f.ctx, 25)
	require.NoError(t, err)
	var checkout koidomain.BillingCheckoutSession
	require.NoError(t, f.db.Where("NOT completed").Take(&checkout).Error)
	require.NoError(t, f.commander.ProcessEvent(f.ctx, koidomain.BillingEvent{
		ID: "evt_successor", CheckoutID: checkout.SessionID, CustomerID: "cus_test",
		OrganizationID: f.organization.ID.String(), SubscriptionID: "sub_successor",
	}))
	f.event(t, "evt_delayed_old_subscription")
	subscription, err := f.repo.Subscription(f.ctx, f.organization.ID)
	require.NoError(t, err)
	require.Equal(t, "sub_successor", subscription.StripeSubscriptionID)
	require.Equal(t, "active", subscription.Status)
}

func TestBillingWebhookSerializesRemoteReads(t *testing.T) {
	f := newBillingFixture(t)
	f.complete(t)
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	f.gateway.beforeRead = func() { entered <- struct{}{}; <-release }
	errs := make(chan error, 2)
	go func() {
		errs <- f.commander.ProcessEvent(f.ctx, koidomain.BillingEvent{ID: "evt_first", CustomerID: "cus_test", SubscriptionID: "sub_test"})
	}()
	<-entered
	go func() {
		errs <- f.commander.ProcessEvent(f.ctx, koidomain.BillingEvent{ID: "evt_second", CustomerID: "cus_test", SubscriptionID: "sub_test"})
	}()
	select {
	case <-entered:
		t.Fatal("concurrent webhook fetched Stripe before acquiring the organization lock")
	case <-time.After(50 * time.Millisecond):
	}
	f.gateway.mu.Lock()
	f.gateway.snapshot.Quantity = 50
	f.gateway.mu.Unlock()
	release <- struct{}{}
	<-entered
	release <- struct{}{}
	require.NoError(t, <-errs)
	require.NoError(t, <-errs)
	subscription, err := f.repo.Subscription(f.ctx, f.organization.ID)
	require.NoError(t, err)
	require.Equal(t, 50, subscription.Quantity)
}

func TestBillingCatalogAndCancellation(t *testing.T) {
	f := newBillingFixture(t)
	querier := koidomain.NewBillingQuerier(f.store, f.gateway, "price_test")
	catalog, err := querier.Catalog(f.ctx)
	require.NoError(t, err)
	require.Len(t, catalog.Options, 12)
	for _, quote := range catalog.Options {
		got, err := querier.Quote(f.ctx, quote.Prompts)
		require.NoError(t, err)
		require.Equal(t, quote, got)
	}
	for _, quantity := range []int{0, 6, 501} {
		_, err := querier.Quote(f.ctx, quantity)
		require.ErrorIs(t, err, koidomain.ErrValidation)
	}
	f.complete(t)
	f.gateway.snapshot.CancelAtPeriodEnd = true
	f.event(t, "evt_cancel_scheduled")
	allowed, err := f.repo.ProductAccess(f.ctx, f.organization.ID)
	require.NoError(t, err)
	require.True(t, allowed)
	_, err = f.commander.ChangeAllowance(f.ctx, 50, "cancelled-upgrade")
	require.ErrorIs(t, err, koidomain.ErrBillingConflict)
	f.gateway.snapshot.Status = "canceled"
	f.event(t, "evt_cancelled")
	allowed, err = f.repo.ProductAccess(f.ctx, f.organization.ID)
	require.NoError(t, err)
	require.False(t, allowed)
}

func TestBillingConcurrentChangeReusesOneOperation(t *testing.T) {
	f := newBillingFixture(t)
	f.complete(t)
	errs := make(chan error, 2)
	var workers sync.WaitGroup
	for range 2 {
		workers.Go(func() {
			_, err := f.commander.ChangeAllowance(f.ctx, 50, "concurrent-upgrade")
			errs <- err
		})
	}
	workers.Wait()
	require.NoError(t, <-errs)
	require.NoError(t, <-errs)
	require.Equal(t, 1, f.gateway.upgradeCalls)
}

func TestBillingDowngradePausesTrackingUntilPromptsFit(t *testing.T) {
	f := newBillingFixture(t)
	f.complete(t)
	brand := &koidomain.Brand{OrganizationID: f.organization.ID, Name: "Koi", Country: "IT"}
	require.NoError(t, f.db.Create(brand).Error)
	for index := range 6 {
		require.NoError(t, f.db.Create(&koidomain.Prompt{
			BrandID:   brand.ID,
			Text:      fmt.Sprintf("Prompt %d?", index),
			Confirmed: true,
		}).Error)
	}
	allowed, err := cloudTrackingAllowed(f.ctx, f.db, f.organization.ID)
	require.NoError(t, err)
	require.True(t, allowed)
	f.gateway.snapshot.Quantity = 5
	f.event(t, "evt_downgrade_applied")
	allowed, err = cloudTrackingAllowed(f.ctx, f.db, f.organization.ID)
	require.NoError(t, err)
	require.False(t, allowed)
	var prompt koidomain.Prompt
	require.NoError(t, f.db.First(&prompt).Error)
	inactive := true
	require.NoError(t, f.store.WithCloudMode(true).PromptRepo().Update(f.ctx,
		prompt.ID,
		koidomain.PromptUpdate{Inactive: &inactive},
	))
	allowed, err = cloudTrackingAllowed(f.ctx, f.db, f.organization.ID)
	require.NoError(t, err)
	require.True(t, allowed)
}
