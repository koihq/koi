package database

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
	"github.com/koihq/koi/pkg/domain"
)

type billingTestGateway struct {
	domain.BillingGateway
	mu               sync.Mutex
	snapshot         domain.BillingSnapshot
	sessions         map[string]domain.BillingCheckout
	createCalls      int
	upgradeCalls     int
	endTrialCalls    int
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
func (g *billingTestGateway) CreateCheckout(_ context.Context, input domain.BillingCheckoutInput) (domain.BillingCheckout, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if session, ok := g.sessions[input.ReservationID]; ok {
		return session, nil
	}
	g.createCalls++
	session := domain.BillingCheckout{
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
func (g *billingTestGateway) ExpireCheckout(_ context.Context, id string) (domain.BillingCheckout, error) {
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
	return domain.BillingCheckout{}, errors.New("missing session")
}

func (g *billingTestGateway) RetrieveCheckout(_ context.Context, id string) (domain.BillingCheckout, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, session := range g.sessions {
		if session.ID == id {
			return session, nil
		}
	}
	return domain.BillingCheckout{}, errors.New("missing session")
}
func (g *billingTestGateway) GetSubscription(context.Context, string) (domain.BillingSnapshot, error) {
	g.mu.Lock()
	snapshot, hook := g.snapshot, g.beforeRead
	g.mu.Unlock()
	if hook != nil {
		hook()
	}
	return snapshot, nil
}
func (g *billingTestGateway) Upgrade(_ context.Context, current domain.BillingSnapshot, quantity int, _ string) (domain.BillingSnapshot, error) {
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
		return domain.BillingSnapshot{}, errors.New("lost Stripe response")
	}
	return g.snapshot, nil
}
func (g *billingTestGateway) ChangeTrial(_ context.Context, _ domain.BillingSnapshot, quantity int, _ string) (domain.BillingSnapshot, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.snapshot.Quantity = quantity
	return g.snapshot, nil
}
func (g *billingTestGateway) EndTrial(_ context.Context, _ domain.BillingSnapshot, _ string) (domain.BillingSnapshot, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.endTrialCalls++
	if g.pendingPayment {
		expires := time.Now().Add(23 * time.Hour)
		quantity := g.snapshot.Quantity
		g.snapshot.PendingUpgradeQuantity = &quantity
		g.snapshot.PendingUpgradeExpiresAt = &expires
		g.snapshot.PaymentURL = "https://invoice.stripe.com/payment"
	} else {
		g.snapshot.Status, g.snapshot.TrialEnd = "active", nil
	}
	return g.snapshot, nil
}
func (g *billingTestGateway) ScheduleDowngrade(_ context.Context, _ domain.BillingSnapshot, quantity int, _ string) (domain.BillingSnapshot, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	id := "sub_sched_test"
	g.snapshot.ScheduleID, g.snapshot.PendingQuantity = &id, &quantity
	return g.snapshot, nil
}
func (g *billingTestGateway) ReleaseSchedule(_ context.Context, _ domain.BillingSnapshot, _ string) (domain.BillingSnapshot, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.releaseCalls++
	g.snapshot.ScheduleID, g.snapshot.PendingQuantity = nil, nil
	return g.snapshot, nil
}

type billingFixture struct {
	db           *gorm.DB
	store        *Store
	repo         *GormBillingRepository
	gateway      *billingTestGateway
	commander    domain.BillingCommander
	organization *domain.Organization
	user         *domain.User
	ctx          context.Context
	analytics    *recordedAnalytics
}

type recordedEvent struct {
	userID uuid.UUID
	name   string
	props  map[string]any
}

type recordedAnalytics struct {
	mu     sync.Mutex
	events []recordedEvent
}

func (a *recordedAnalytics) Track(userID, _ uuid.UUID, event string, props map[string]any) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.events = append(a.events, recordedEvent{userID: userID, name: event, props: props})
}

func (a *recordedAnalytics) names() []string {
	a.mu.Lock()
	defer a.mu.Unlock()

	names := make([]string, 0, len(a.events))
	for _, event := range a.events {
		names = append(names, event.name)
	}

	return names
}

func newBillingFixture(t *testing.T) *billingFixture {
	t.Helper()
	db, err := Open(testdb.URL(t))
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	require.NoError(t, Migrate(db))
	organization := &domain.Organization{Name: "Koi"}
	require.NoError(t, db.Create(organization).Error)
	user := &domain.User{Email: uuid.NewString() + "@example.com", PasswordHash: "hash"}
	require.NoError(t, db.Create(user).Error)
	require.NoError(t, db.Create(&domain.Membership{
		UserID:         user.ID,
		OrganizationID: organization.ID,
		Role:           domain.RoleOwner,
	}).Error)
	repo := NewBillingRepository(db, false)
	_, err = repo.SaveCustomer(t.Context(), organization.ID, "cus_test")
	require.NoError(t, err)
	end := time.Now().Add(30 * 24 * time.Hour)
	gateway := &billingTestGateway{
		snapshot: domain.BillingSnapshot{
			ID: "sub_test", CustomerID: "cus_test", OrganizationID: organization.ID.String(),
			ItemID: "si_test", PriceID: "price_test", Quantity: 25, Status: "active", CurrentPeriodEnd: &end,
		},
		sessions: make(map[string]domain.BillingCheckout),
	}
	store := NewStore(db)
	analytics := &recordedAnalytics{}
	return &billingFixture{
		db: db, store: store, repo: repo, gateway: gateway, organization: organization, user: user,
		commander: domain.NewBillingCommander(store, gateway, analytics, "price_test", "https://koi.example"),
		ctx:       testIdentity(t, store, user.ID),
		analytics: analytics,
	}
}

func (f *billingFixture) complete(t *testing.T) {
	t.Helper()
	_, err := f.commander.Checkout(f.ctx, 25)
	require.NoError(t, err)
	var checkout domain.BillingCheckoutSession
	require.NoError(t, f.db.First(&checkout).Error)
	require.NoError(t, f.commander.ProcessEvent(f.ctx, domain.BillingEvent{
		ID: "evt_checkout", CheckoutID: checkout.SessionID, CustomerID: "cus_test",
		OrganizationID: f.organization.ID.String(), SubscriptionID: "sub_test",
	}))
}

func (f *billingFixture) event(t *testing.T, id string) {
	t.Helper()
	require.NoError(t, f.commander.ProcessEvent(f.ctx, domain.BillingEvent{
		ID: id, CustomerID: "cus_test", SubscriptionID: "sub_test",
	}))
}

func TestBillingWebhookIdentityReplayAndEntitlement(t *testing.T) {
	f := newBillingFixture(t)
	// A new organization can read its First Look before any subscription exists.
	allowed, err := f.repo.ProductAccess(f.ctx, f.organization.ID)
	require.NoError(t, err)
	require.True(t, allowed)
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
	require.Error(t, f.commander.ProcessEvent(f.ctx, domain.BillingEvent{ID: "evt_bad", CustomerID: "cus_test", SubscriptionID: "sub_test"}))
	f.gateway.snapshot.CustomerID = "cus_test"
	f.gateway.snapshot.Quantity = 6
	require.Error(t, f.commander.ProcessEvent(f.ctx, domain.BillingEvent{ID: "evt_bad", CustomerID: "cus_test", SubscriptionID: "sub_test"}))
	f.gateway.snapshot.Quantity = 100
	f.event(t, "evt_bad") // Failed transactions leave the event retryable.
	subscription, err = f.repo.Subscription(f.ctx, f.organization.ID)
	require.NoError(t, err)
	require.Equal(t, 100, subscription.Quantity)
	require.NoError(t, f.db.First(f.organization, "id = ?", f.organization.ID).Error)
	require.NotNil(t, f.organization.TrialUsedAt)

	// Analytics count each committed change once: not the duplicate, not the failed attempts.
	require.Equal(t, []string{"subscription_activated", "allowance_changed", "allowance_changed"}, f.analytics.names())
	for _, event := range f.analytics.events {
		require.Equal(t, f.user.ID, event.userID, "billing is attributed to the owner")
	}
	require.Equal(t, int64(4500), f.analytics.events[0].props["amount_cents"])
}

func TestBillingCheckoutConfirmationWithoutWebhook(t *testing.T) {
	f := newBillingFixture(t)
	_, err := f.commander.Checkout(f.ctx, 25)
	require.NoError(t, err)
	var checkout domain.BillingCheckoutSession
	require.NoError(t, f.db.First(&checkout).Error)

	status, err := f.commander.ConfirmCheckout(f.ctx, checkout.SessionID)
	require.NoError(t, err)
	require.True(t, status.FirstLook)
	require.Equal(t, domain.FirstLookPromptLimit, status.PromptLimit)
	otherOrg := &domain.Organization{Name: "Other"}
	require.NoError(t, f.db.Create(otherOrg).Error)
	otherUser := &domain.User{Email: uuid.NewString() + "@example.com", PasswordHash: "hash"}
	require.NoError(t, f.db.Create(otherUser).Error)
	require.NoError(t, f.db.Create(&domain.Membership{
		UserID: otherUser.ID, OrganizationID: otherOrg.ID, Role: domain.RoleOwner,
	}).Error)
	otherCtx := testIdentity(t, f.store, otherUser.ID)
	_, err = f.commander.ConfirmCheckout(otherCtx, checkout.SessionID)
	require.ErrorIs(t, err, domain.ErrNotFound)
	trialEnd := time.Now().Add(7 * 24 * time.Hour)
	f.gateway.snapshot.Status = "trialing"
	f.gateway.snapshot.TrialEnd = &trialEnd
	for key, session := range f.gateway.sessions {
		session.Status = "complete"
		session.SubscriptionID = f.gateway.snapshot.ID
		f.gateway.sessions[key] = session
	}
	status, err = f.commander.ConfirmCheckout(f.ctx, checkout.SessionID)
	require.NoError(t, err)
	require.Equal(t, domain.TrialPromptLimit, status.PromptLimit)
	require.NoError(t, f.commander.ProcessEvent(f.ctx, domain.BillingEvent{
		ID: "evt_late", CheckoutID: checkout.SessionID, CustomerID: "cus_test",
		OrganizationID: f.organization.ID.String(), SubscriptionID: "sub_test",
	}))
	var count int64
	require.NoError(t, f.db.Model(&domain.BillingSubscription{}).Count(&count).Error)
	require.EqualValues(t, 1, count)
}

func TestBillingMaintenanceRecoversCompletedCheckout(t *testing.T) {
	f := newBillingFixture(t)
	_, err := f.commander.Checkout(f.ctx, 25)
	require.NoError(t, err)
	var checkout domain.BillingCheckoutSession
	require.NoError(t, f.db.First(&checkout).Error)
	for key, session := range f.gateway.sessions {
		session.Status = "complete"
		session.SubscriptionID = f.gateway.snapshot.ID
		f.gateway.sessions[key] = session
	}
	require.NoError(t, f.db.Model(&checkout).Update("updated_at", time.Now().Add(-2*time.Minute)).Error)
	require.NoError(t, f.commander.ReconcilePendingCheckouts(f.ctx, time.Now()))
	allowed, err := f.repo.ProductAccess(f.ctx, f.organization.ID)
	require.NoError(t, err)
	require.True(t, allowed)
}

func TestBillingWebhookAndConfirmationRace(t *testing.T) {
	f := newBillingFixture(t)
	_, err := f.commander.Checkout(f.ctx, 25)
	require.NoError(t, err)
	var checkout domain.BillingCheckoutSession
	require.NoError(t, f.db.First(&checkout).Error)
	for key, session := range f.gateway.sessions {
		session.Status = "complete"
		session.SubscriptionID = f.gateway.snapshot.ID
		f.gateway.sessions[key] = session
	}

	var group sync.WaitGroup
	errors := make(chan error, 2)
	group.Go(func() {
		_, err := f.commander.ConfirmCheckout(f.ctx, checkout.SessionID)
		errors <- err
	})
	group.Go(func() {
		errors <- f.commander.ProcessEvent(f.ctx, domain.BillingEvent{
			ID: "evt_race", CheckoutID: checkout.SessionID, CustomerID: "cus_test",
			OrganizationID: f.organization.ID.String(), SubscriptionID: "sub_test",
		})
	})
	group.Wait()
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}
	var count int64
	require.NoError(t, f.db.Model(&domain.BillingSubscription{}).Count(&count).Error)
	require.EqualValues(t, 1, count)
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
	var checkout domain.BillingCheckoutSession
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
				require.ErrorIs(t, err, domain.ErrBillingSubscriptionExists)
				require.Equal(t, 1, f.gateway.createCalls)
				subscription, err := f.repo.Subscription(f.ctx, f.organization.ID)
				require.NoError(t, err)
				require.Equal(t, 25, subscription.Quantity)
			} else {
				require.NoError(t, err)
				require.NotEqual(t, first, second)
				require.Equal(t, 2, f.gateway.createCalls)
				var retired int64
				require.NoError(t, f.db.Model(&domain.BillingCheckoutSession{}).Where("retired").Count(&retired).Error)
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
	result, err := f.commander.ChangeAllowance(f.ctx, 15, "trial-change")
	require.NoError(t, err)
	require.Equal(t, "trial_end", result.Effective)
	subscription, err := f.repo.Subscription(f.ctx, f.organization.ID)
	require.NoError(t, err)
	require.Equal(t, domain.TrialPromptLimit, subscription.PromptLimit(time.Now()))
	require.WithinDuration(t, trialEnd, *subscription.TrialEnd, time.Microsecond)
	require.Equal(t, 0, f.gateway.upgradeCalls)

	f.gateway.snapshot.Status, f.gateway.snapshot.TrialEnd = "active", nil
	f.gateway.snapshot.Quantity = 50
	f.event(t, "evt_trial_ended")
	for index, quantity := range []int{25, 15, 50, 25, 100} {
		_, err := f.commander.ChangeAllowance(f.ctx, quantity, fmt.Sprintf("change-%d", index))
		require.NoError(t, err)
	}
	subscription, err = f.repo.Subscription(f.ctx, f.organization.ID)
	require.NoError(t, err)
	require.Equal(t, 100, subscription.Quantity)
	require.Nil(t, subscription.PendingQuantity)
	require.Equal(t, 2, f.gateway.releaseCalls)
	require.Equal(t, 1, f.gateway.upgradeCalls)
}

func TestBillingStartPlanEndsTrialAfterPayment(t *testing.T) {
	f := newBillingFixture(t)
	trialEnd := time.Now().Add(7 * 24 * time.Hour)
	f.gateway.snapshot.Status, f.gateway.snapshot.TrialEnd = "trialing", &trialEnd
	f.complete(t)

	result, err := f.commander.StartPlan(f.ctx, "start")
	require.NoError(t, err)
	require.Equal(t, "started", result.Effective)
	require.Equal(t, 25, result.Quantity)
	require.Empty(t, result.PaymentURL)

	subscription, err := f.repo.Subscription(f.ctx, f.organization.ID)
	require.NoError(t, err)
	require.Equal(t, "active", subscription.Status)
	require.Equal(t, 25, subscription.PromptLimit(time.Now()))

	replayed, err := f.commander.StartPlan(f.ctx, "start")
	require.NoError(t, err)
	require.Equal(t, "started", replayed.Effective)
	require.Equal(t, 1, f.gateway.endTrialCalls)

	require.Equal(t, []string{"trial_started", "subscription_activated"}, f.analytics.names())
	require.Equal(t, true, f.analytics.events[1].props["from_trial"])
}

func TestBillingStartPlanKeepsTrialWhenPaymentFails(t *testing.T) {
	f := newBillingFixture(t)
	trialEnd := time.Now().Add(7 * 24 * time.Hour)
	f.gateway.snapshot.Status, f.gateway.snapshot.TrialEnd = "trialing", &trialEnd
	f.gateway.pendingPayment = true
	f.complete(t)

	result, err := f.commander.StartPlan(f.ctx, "start")
	require.NoError(t, err)
	require.Equal(t, "after_payment", result.Effective)
	require.Equal(t, "https://invoice.stripe.com/payment", result.PaymentURL)
	require.NotNil(t, result.ExpiresAt)

	subscription, err := f.repo.Subscription(f.ctx, f.organization.ID)
	require.NoError(t, err)
	require.Equal(t, domain.TrialPromptLimit, subscription.PromptLimit(time.Now()))

	// A new attempt would void the open invoice, so it returns the same payment link.
	again, err := f.commander.StartPlan(f.ctx, "start-again")
	require.NoError(t, err)
	require.Equal(t, result.PaymentURL, again.PaymentURL)
	require.Equal(t, 1, f.gateway.endTrialCalls)

	// The open invoice is for the current plan, so its quantity cannot change.
	_, err = f.commander.ChangeAllowance(f.ctx, 50, "change-while-open")
	require.ErrorIs(t, err, domain.ErrBillingConflict)

	f.gateway.snapshot.PendingUpgradeQuantity, f.gateway.snapshot.PendingUpgradeExpiresAt = nil, nil
	f.gateway.snapshot.PaymentURL = ""
	f.event(t, "evt_pending_expired")
	subscription, err = f.repo.Subscription(f.ctx, f.organization.ID)
	require.NoError(t, err)
	require.Equal(t, "trialing", subscription.Status)
	require.Nil(t, subscription.PendingUpgradeQuantity)
}

func TestBillingStartPlanRequiresAnOpenTrial(t *testing.T) {
	f := newBillingFixture(t)
	_, err := f.commander.StartPlan(f.ctx, "no-subscription")
	require.ErrorIs(t, err, domain.ErrBillingConflict)

	trialEnd := time.Now().Add(7 * 24 * time.Hour)
	f.gateway.snapshot.Status, f.gateway.snapshot.TrialEnd = "trialing", &trialEnd
	f.complete(t)

	for _, change := range []func(){
		func() { f.gateway.snapshot.CancelAtPeriodEnd = true },
		func() { f.gateway.snapshot.CancelAtPeriodEnd, f.gateway.snapshot.Status = false, "past_due" },
	} {
		change()
		_, err = f.commander.StartPlan(f.ctx, "closed")
		require.ErrorIs(t, err, domain.ErrBillingConflict)
	}
	require.Equal(t, 0, f.gateway.endTrialCalls)
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
	require.ErrorIs(t, err, domain.ErrIdempotencyConflict)
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
	require.ErrorIs(t, err, domain.ErrBillingConflict)
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
	var checkout domain.BillingCheckoutSession
	require.NoError(t, f.db.Where("NOT completed").Take(&checkout).Error)
	require.NoError(t, f.commander.ProcessEvent(f.ctx, domain.BillingEvent{
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
		errs <- f.commander.ProcessEvent(f.ctx, domain.BillingEvent{ID: "evt_first", CustomerID: "cus_test", SubscriptionID: "sub_test"})
	}()
	<-entered
	go func() {
		errs <- f.commander.ProcessEvent(f.ctx, domain.BillingEvent{ID: "evt_second", CustomerID: "cus_test", SubscriptionID: "sub_test"})
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
	querier := domain.NewBillingQuerier(f.store, f.gateway, "price_test")
	catalog, err := querier.Catalog(f.ctx)
	require.NoError(t, err)
	require.Len(t, catalog.Options, 10)
	for _, quote := range catalog.Options {
		got, err := querier.Quote(f.ctx, quote.Prompts)
		require.NoError(t, err)
		require.Equal(t, quote, got)
	}
	for _, quantity := range []int{0, 5, 6, 501} {
		_, err := querier.Quote(f.ctx, quantity)
		require.ErrorIs(t, err, domain.ErrValidation)
	}
	f.complete(t)
	f.gateway.snapshot.CancelAtPeriodEnd = true
	f.event(t, "evt_cancel_scheduled")
	allowed, err := f.repo.ProductAccess(f.ctx, f.organization.ID)
	require.NoError(t, err)
	require.True(t, allowed)
	_, err = f.commander.ChangeAllowance(f.ctx, 50, "cancelled-upgrade")
	require.ErrorIs(t, err, domain.ErrBillingConflict)
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
	brand := &domain.Brand{OrganizationID: f.organization.ID, Name: "Koi", Country: "IT"}
	require.NoError(t, f.db.Create(brand).Error)
	for index := range 16 {
		require.NoError(t, f.db.Create(&domain.Prompt{
			BrandID:   brand.ID,
			Text:      fmt.Sprintf("Prompt %d?", index),
			Source:    domain.PromptSourceManual,
			Confirmed: true,
		}).Error)
	}
	allowed, err := cloudCollectionAllowed(f.ctx, f.db, f.organization.ID, false)
	require.NoError(t, err)
	require.True(t, allowed)
	f.gateway.snapshot.Quantity = 15
	f.event(t, "evt_downgrade_applied")
	allowed, err = cloudCollectionAllowed(f.ctx, f.db, f.organization.ID, false)
	require.NoError(t, err)
	require.False(t, allowed)
	var prompt domain.Prompt
	require.NoError(t, f.db.First(&prompt).Error)
	inactive := true
	_, err = domain.NewPromptCommander(f.store.WithCloudMode(true)).Update(f.ctx, prompt.ID, domain.UpdatePromptParams{
		Inactive: &inactive,
	})
	require.NoError(t, err)
	allowed, err = cloudCollectionAllowed(f.ctx, f.db, f.organization.ID, false)
	require.NoError(t, err)
	require.True(t, allowed)
}
