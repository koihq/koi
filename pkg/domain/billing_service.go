package domain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/koihq/koi/pkg/pricing"
)

type billingService struct {
	store     Store
	gateway   BillingGateway
	analytics Analytics
	priceID   string
	publicURL string
}

func NewBillingCommander(store Store, gateway BillingGateway, analytics Analytics, priceID, publicURL string) BillingCommander {
	return &billingService{
		store:     store,
		gateway:   gateway,
		analytics: orNop(analytics),
		priceID:   priceID,
		publicURL: publicURL,
	}
}

func NewBillingQuerier(store Store, gateway BillingGateway, priceID string) BillingQuerier {
	return &billingService{store: store, gateway: gateway, analytics: NopAnalytics{}, priceID: priceID}
}

func (s *billingService) Catalog(ctx context.Context) (BillingCatalog, error) {
	if err := s.gateway.ValidatePrice(ctx, s.priceID); err != nil {
		return BillingCatalog{}, err
	}

	catalog := pricing.Current()
	result := BillingCatalog{Currency: catalog.Currency, Interval: "month", Options: make([]BillingQuote, 0, len(catalog.Options))}
	for _, option := range catalog.Options {
		result.Options = append(result.Options, BillingQuote{
			Prompts:     option.Prompts,
			AmountCents: int64(option.Euros) * 100,
			Currency:    catalog.Currency,
			Interval:    "month",
		})
	}

	return result, nil
}

func validateBillingQuantity(quantity int) error {
	if _, err := pricing.Current().Quote(quantity); err != nil {
		return Validation(FieldError{Field: "prompts", Code: FieldInvalid})
	}

	return nil
}

func (s *billingService) Quote(ctx context.Context, quantity int) (BillingQuote, error) {
	if err := validateBillingQuantity(quantity); err != nil {
		return BillingQuote{}, err
	}

	catalog, err := s.Catalog(ctx)
	if err != nil {
		return BillingQuote{}, err
	}
	for _, option := range catalog.Options {
		if option.Prompts == quantity {
			return option, nil
		}
	}

	return BillingQuote{}, ErrBillingPriceMismatch
}

func (s *billingService) Subscription(ctx context.Context) (BillingStatus, error) {
	organization, err := s.store.BillingRepo().OrganizationForUser(ctx)
	if err != nil {
		return BillingStatus{}, err
	}

	subscription, err := s.store.BillingRepo().Subscription(ctx, organization.ID)
	if err != nil {
		return BillingStatus{}, err
	}

	usedPrompts, err := s.store.BillingRepo().UsedPromptCount(ctx, organization.ID)
	if err != nil {
		return BillingStatus{}, err
	}

	var current BillingSubscription
	if subscription != nil {
		current = *subscription
	}

	return BillingStatus{
		Subscription:   subscription,
		PromptLimit:    EffectivePromptLimit(current, organization.TrialUsedAt, time.Now().UTC()),
		UsedPrompts:    usedPrompts,
		TrialAvailable: organization.TrialUsedAt == nil,
		FirstLook:      IsFirstLook(current, organization.TrialUsedAt),
	}, nil
}

func (s *billingService) ConfirmCheckout(ctx context.Context, sessionID string) (BillingStatus, error) {
	organization, err := s.store.BillingRepo().OrganizationForUser(ctx)
	if err != nil {
		return BillingStatus{}, err
	}
	if sessionID == "" {
		return BillingStatus{}, Validation(FieldError{Field: "sessionId", Code: FieldInvalid})
	}
	if err := s.reconcileCheckout(ctx, organization.ID, sessionID); err != nil {
		return BillingStatus{}, err
	}
	return s.Subscription(ctx)
}

func (s *billingService) ReconcilePendingCheckouts(ctx context.Context, now time.Time) error {
	checkouts, err := s.store.BillingRepo().PendingCheckouts(ctx, now.Add(-time.Minute), 20)
	if err != nil {
		return err
	}
	var issues error
	for _, checkout := range checkouts {
		if err := s.reconcileCheckout(ctx, checkout.OrganizationID, checkout.SessionID); err != nil {
			issues = errors.Join(issues, err)
			if touchErr := s.store.BillingRepo().TouchCheckout(ctx, checkout.ID); touchErr != nil {
				issues = errors.Join(issues, touchErr)
			}
		}
	}
	return issues
}

func (s *billingService) reconcileCheckout(ctx context.Context, organizationID uuid.UUID, sessionID string) error {
	return s.atomic(ctx, func(ctx context.Context, store Store) error {
		repo := store.BillingRepo()
		organization, err := repo.LockOrganization(ctx, organizationID)
		if err != nil {
			return err
		}
		checkout, err := repo.CheckoutBySession(ctx, sessionID)
		if err != nil {
			return err
		}
		if checkout.OrganizationID != organizationID {
			return ErrNotFound
		}
		if checkout.Completed || checkout.Retired {
			return nil
		}
		remote, err := s.gateway.RetrieveCheckout(ctx, sessionID)
		if err != nil {
			return err
		}
		switch remote.Status {
		case "complete":
			if err := validateCheckout(organization, checkout, remote.CustomerID, remote.OrganizationID, remote.SubscriptionID); err != nil {
				return err
			}
			return s.completeCheckout(ctx, repo, organization, checkout, remote.SubscriptionID)
		case "expired":
			return repo.RetireCheckout(ctx, checkout.ID)
		default:
			return repo.TouchCheckout(ctx, checkout.ID)
		}
	})
}

func (s *billingService) completeCheckout(ctx context.Context, repo BillingRepository, organization *Organization, checkout *BillingCheckoutSession, subscriptionID string) error {
	if err := repo.CompleteCheckout(ctx, checkout, subscriptionID); err != nil {
		return err
	}
	return s.refresh(ctx, repo, organization, subscriptionID)
}

func (s *billingService) ProductAccess(ctx context.Context, organizationID uuid.UUID) (bool, error) {
	return s.store.BillingRepo().ProductAccess(ctx, organizationID)
}

func (s *billingService) Portal(ctx context.Context) (string, error) {
	organization, err := s.store.BillingRepo().OrganizationForUser(ctx)
	if err != nil {
		return "", err
	}
	if organization.StripeCustomerID == nil {
		return "", ErrBillingConflict
	}

	return s.gateway.CreatePortal(ctx, *organization.StripeCustomerID, s.publicURL)
}

func (s *billingService) Checkout(ctx context.Context, quantity int) (string, error) {
	if _, err := s.Quote(ctx, quantity); err != nil {
		return "", err
	}
	organization, err := s.store.BillingRepo().OrganizationForUser(ctx)
	if err != nil {
		return "", err
	}

	var customerID string
	err = s.store.Atomic(ctx, func(store Store) error {
		repo := store.BillingRepo()
		locked, err := repo.LockOrganization(ctx, organization.ID)
		if err != nil {
			return err
		}
		if locked.StripeCustomerID != nil {
			customerID = *locked.StripeCustomerID
			return nil
		}

		customerID, err = s.gateway.CreateCustomer(ctx, organization.ID.String())
		if err != nil {
			return err
		}
		customerID, err = repo.SaveCustomer(ctx, organization.ID, customerID)
		return err
	})
	if err != nil {
		return "", err
	}

	// A competing tab may replace a reservation before this request acquires its lock.
	for range 3 {
		reservation, err := s.store.BillingRepo().ReserveCheckout(ctx, organization.ID, customerID, s.priceID, quantity)
		if err != nil {
			return "", err
		}

		var checkoutURL string
		var completed bool
		err = s.atomic(ctx, func(ctx context.Context, store Store) error {
			repo := store.BillingRepo()
			locked, err := repo.LockOrganization(ctx, organization.ID)
			if err != nil {
				return err
			}
			checkout, err := repo.Checkout(ctx, reservation.ID)
			if err != nil || checkout.Retired {
				return err
			}
			local, err := repo.Subscription(ctx, organization.ID)
			if err != nil {
				return err
			}
			if local != nil && local.Status != "canceled" && local.Status != "incomplete_expired" {
				return ErrBillingSubscriptionExists
			}

			if checkout.URL == "" {
				session, err := s.gateway.CreateCheckout(ctx, BillingCheckoutInput{
					CustomerID:     customerID,
					OrganizationID: organization.ID.String(),
					PriceID:        checkout.PriceID,
					ReservationID:  checkout.ID.String(),
					Quantity:       checkout.Quantity,
					PublicURL:      s.publicURL,
					Trial:          checkout.Trial,
				})
				if err != nil {
					return err
				}
				if err := repo.FinalizeCheckout(ctx, checkout.ID, session.ID, session.URL, session.ExpiresAt); err != nil {
					return err
				}
				checkout.SessionID, checkout.URL = session.ID, session.URL
			}
			if checkout.Quantity == quantity && checkout.PriceID == s.priceID {
				checkoutURL = checkout.URL
				return nil
			}

			remote, err := s.gateway.ExpireCheckout(ctx, checkout.SessionID)
			if err != nil {
				return err
			}
			if remote.Status == "complete" {
				if err := validateCheckout(locked, checkout, remote.CustomerID, remote.OrganizationID, remote.SubscriptionID); err != nil {
					return err
				}
				completed = true
				return s.completeCheckout(ctx, repo, locked, checkout, remote.SubscriptionID)
			}
			if remote.Status != "expired" {
				return ErrBillingCheckoutPending
			}

			return repo.RetireCheckout(ctx, checkout.ID)
		})
		if err != nil {
			return "", err
		}
		if completed {
			return "", ErrBillingSubscriptionExists
		}
		if checkoutURL != "" {
			return checkoutURL, nil
		}
	}

	return "", ErrBillingCheckoutPending
}

func (s *billingService) ChangeAllowance(ctx context.Context, quantity int, key string) (BillingChange, error) {
	if err := validateBillingQuantity(quantity); err != nil {
		return BillingChange{}, err
	}
	if key == "" {
		return BillingChange{}, Validation(FieldError{Field: "idempotency_key", Code: FieldInvalid})
	}
	organization, err := s.store.BillingRepo().OrganizationForUser(ctx)
	if err != nil {
		return BillingChange{}, err
	}
	request, err := s.store.BillingRepo().ReserveChange(ctx, organization.ID, key, quantity)
	if err != nil {
		return BillingChange{}, err
	}
	if request.Result != nil {
		return *request.Result, nil
	}
	if request.ErrorCode != "" {
		return BillingChange{}, &Error{Code: request.ErrorCode, Message: request.ErrorCode}
	}
	if err := s.gateway.ValidatePrice(ctx, s.priceID); err != nil {
		return BillingChange{}, err
	}

	for {
		var result BillingChange
		var rejected error
		var done bool
		err := s.atomic(ctx, func(ctx context.Context, store Store) error {
			repo := store.BillingRepo()
			locked, err := repo.LockOrganization(ctx, organization.ID)
			if err != nil {
				return err
			}
			pending, err := repo.PendingChanges(ctx, organization.ID)
			if err != nil {
				return err
			}
			if len(pending) == 0 {
				stored, err := repo.ReserveChange(ctx, organization.ID, key, quantity)
				if err != nil {
					return err
				}
				done = true
				if stored.Result != nil {
					result = *stored.Result
				} else {
					rejected = &Error{Code: stored.ErrorCode, Message: stored.ErrorCode}
				}
				return nil
			}

			operation := pending[0]
			change, err := s.applyChange(ctx, repo, locked, operation)
			if err != nil {
				var domainError *Error
				if !errors.As(err, &domainError) {
					return err
				}
				operation.ErrorCode = domainError.Code
				rejected = err
			} else {
				operation.Result = &change
				result = change
			}
			done = operation.ID == request.ID
			return repo.SaveChange(ctx, &operation)
		})
		if err != nil {
			return BillingChange{}, err
		}
		if done {
			return result, rejected
		}
	}
}

// StartPlan ends a trial and charges the selected plan now. A failed payment keeps the trial,
// and an open payment is returned as is because a new attempt would void its invoice.
func (s *billingService) StartPlan(ctx context.Context, key string) (BillingChange, error) {
	organization, err := s.store.BillingRepo().OrganizationForUser(ctx)
	if err != nil {
		return BillingChange{}, err
	}

	var result BillingChange
	err = s.atomic(ctx, func(ctx context.Context, store Store) error {
		repo := store.BillingRepo()

		locked, err := repo.LockOrganization(ctx, organization.ID)
		if err != nil {
			return err
		}

		pending, err := repo.PendingChanges(ctx, organization.ID)
		if err != nil {
			return err
		}
		local, err := repo.Subscription(ctx, organization.ID)
		if err != nil {
			return err
		}
		if len(pending) > 0 || local == nil {
			return ErrBillingConflict
		}

		current, err := s.gateway.GetSubscription(ctx, local.StripeSubscriptionID)
		if err != nil {
			return err
		}
		if err := s.validateSnapshot(locked, current); err != nil {
			return err
		}

		current, err = s.endTrial(ctx, current, key)
		if err != nil {
			return err
		}
		if err := s.saveSnapshot(ctx, repo, locked, current); err != nil {
			return err
		}

		result = BillingChange{Quantity: current.Quantity, Effective: "started"}
		if current.Status == "trialing" {
			result.Effective = "after_payment"
			result.PaymentURL = current.PaymentURL
			result.ExpiresAt = current.PendingUpgradeExpiresAt
		}

		return nil
	})
	if err != nil {
		return BillingChange{}, err
	}

	return result, nil
}

func (s *billingService) endTrial(ctx context.Context, current BillingSnapshot, key string) (BillingSnapshot, error) {
	if current.CancelAtPeriodEnd {
		return BillingSnapshot{}, ErrBillingConflict
	}

	// An active plan means a repeated request already succeeded.
	if current.Status == "active" {
		return current, nil
	}
	if current.Status != "trialing" {
		return BillingSnapshot{}, ErrBillingConflict
	}
	if current.PendingUpgradeQuantity != nil {
		return current, nil
	}

	stripeKey := "koi-start-plan-" + key
	if current.ScheduleID != nil {
		released, err := s.gateway.ReleaseSchedule(ctx, current, stripeKey+"-release")
		if err != nil {
			return BillingSnapshot{}, err
		}

		current = released
	}

	return s.gateway.EndTrial(ctx, current, stripeKey)
}

func (s *billingService) applyChange(ctx context.Context, repo BillingRepository, organization *Organization, operation BillingChangeRequest) (BillingChange, error) {
	local, err := repo.Subscription(ctx, organization.ID)
	if err != nil {
		return BillingChange{}, err
	}
	if local == nil || local.StripeSubscriptionID != operation.SubscriptionID {
		return BillingChange{}, ErrBillingConflict
	}
	current, err := s.gateway.GetSubscription(ctx, operation.SubscriptionID)
	if err != nil {
		return BillingChange{}, err
	}
	if err := s.validateSnapshot(organization, current); err != nil {
		return BillingChange{}, err
	}
	if current.CancelAtPeriodEnd || (current.Status != "active" && current.Status != "trialing") {
		return BillingChange{}, ErrBillingConflict
	}
	if current.PendingUpgradeQuantity != nil && *current.PendingUpgradeQuantity != operation.Quantity {
		return BillingChange{}, ErrBillingConflict
	}

	key := "koi-change-" + operation.ID.String()
	effective := "current"
	quantity := operation.Quantity
	if current.Status == "trialing" {
		if current.ScheduleID != nil {
			current, err = s.gateway.ReleaseSchedule(ctx, current, key+"-release")
			if err != nil {
				return BillingChange{}, err
			}
		}
		if quantity != current.Quantity {
			current, err = s.gateway.ChangeTrial(ctx, current, quantity, key+"-trial")
		}
		effective = "trial_end"
	} else if quantity < current.Quantity {
		if current.PendingQuantity == nil || *current.PendingQuantity != quantity {
			current, err = s.gateway.ScheduleDowngrade(ctx, current, quantity, key)
		}
		effective = "next_cycle"
	} else {
		if current.ScheduleID != nil {
			current, err = s.gateway.ReleaseSchedule(ctx, current, key+"-release")
			if err != nil {
				return BillingChange{}, err
			}
		}
		if quantity > current.Quantity {
			if current.PendingUpgradeQuantity == nil {
				current, err = s.gateway.Upgrade(ctx, current, quantity, key+"-upgrade")
			}
			effective = "after_payment"
		}
	}
	if err != nil {
		return BillingChange{}, err
	}
	if err := s.saveSnapshot(ctx, repo, organization, current); err != nil {
		return BillingChange{}, err
	}

	return BillingChange{Quantity: quantity, Effective: effective, PaymentURL: current.PaymentURL, ExpiresAt: current.PendingUpgradeExpiresAt}, nil
}

func validateCheckout(organization *Organization, checkout *BillingCheckoutSession, customerID, organizationID, subscriptionID string) error {
	if organization.StripeCustomerID == nil || *organization.StripeCustomerID != customerID ||
		checkout.OrganizationID != organization.ID || checkout.OrganizationID.String() != organizationID ||
		checkout.CustomerID != customerID || subscriptionID == "" ||
		(checkout.StripeSubscriptionID != nil && *checkout.StripeSubscriptionID != subscriptionID) {
		return errors.New("Stripe checkout identity mismatch")
	}

	return nil
}

func (s *billingService) ProcessEvent(ctx context.Context, event BillingEvent) error {
	organization, err := s.store.BillingRepo().OrganizationByCustomer(ctx, event.CustomerID)
	if errors.Is(err, ErrNotFound) {
		return nil // Other products in the Stripe account do not belong to Koi.
	}
	if err != nil {
		return err
	}

	return s.atomic(ctx, func(ctx context.Context, store Store) error {
		repo := store.BillingRepo()
		locked, err := repo.LockOrganization(ctx, organization.ID)
		if err != nil {
			return err
		}
		fresh, err := repo.RecordEvent(ctx, event.ID)
		if err != nil || !fresh {
			return err
		}

		if event.CheckoutID != "" {
			checkout, err := repo.CheckoutBySession(ctx, event.CheckoutID)
			if err != nil {
				return err
			}
			if err := validateCheckout(locked, checkout, event.CustomerID, event.OrganizationID, event.SubscriptionID); err != nil {
				return err
			}
			return s.completeCheckout(ctx, repo, locked, checkout, event.SubscriptionID)
		} else {
			if _, err := repo.CompletedCheckout(ctx, organization.ID, event.SubscriptionID); errors.Is(err, ErrNotFound) {
				return nil // Checkout's event will fetch the current state after association.
			} else if err != nil {
				return err
			}
			local, err := repo.Subscription(ctx, organization.ID)
			if err != nil {
				return err
			}
			if local != nil && local.StripeSubscriptionID != event.SubscriptionID {
				return nil // A delayed event from a previous subscription cannot replace its successor.
			}
		}

		return s.refresh(ctx, repo, locked, event.SubscriptionID)
	})
}

func (s *billingService) refresh(ctx context.Context, repo BillingRepository, organization *Organization, subscriptionID string) error {
	current, err := s.gateway.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return err
	}

	return s.saveSnapshot(ctx, repo, organization, current)
}

func (s *billingService) validateSnapshot(organization *Organization, snapshot BillingSnapshot) error {
	if snapshot.ID == "" || snapshot.ItemID == "" || organization.StripeCustomerID == nil ||
		snapshot.CustomerID != *organization.StripeCustomerID || snapshot.OrganizationID != organization.ID.String() {
		return errors.New("Stripe subscription identity mismatch")
	}
	if snapshot.PriceID != s.priceID {
		return ErrBillingPriceMismatch
	}
	if err := validateBillingQuantity(snapshot.Quantity); err != nil {
		return fmt.Errorf("invalid Stripe subscription quantity: %w", err)
	}

	return nil
}

func (s *billingService) saveSnapshot(ctx context.Context, repo BillingRepository, organization *Organization, snapshot BillingSnapshot) error {
	if err := s.validateSnapshot(organization, snapshot); err != nil {
		return err
	}
	checkout, err := repo.CompletedCheckout(ctx, organization.ID, snapshot.ID)
	if err != nil {
		return err
	}
	local, err := repo.Subscription(ctx, organization.ID)
	if err != nil {
		return err
	}
	events := billingTransitions(local, snapshot, time.Now().UTC())

	if local == nil {
		local = &BillingSubscription{OrganizationID: organization.ID}
	}
	if local.StripeSubscriptionID != snapshot.ID {
		if local.StripeSubscriptionID != "" && local.Status != "canceled" && local.Status != "incomplete_expired" {
			return ErrBillingSubscriptionExists
		}
		if checkout.PriceID != snapshot.PriceID || checkout.Quantity != snapshot.Quantity {
			return errors.New("Stripe subscription differs from Checkout")
		}
	}

	local.StripeSubscriptionID = snapshot.ID
	local.StripePriceID = snapshot.PriceID
	local.Quantity = snapshot.Quantity
	local.Status = snapshot.Status
	local.CurrentPeriodEnd = snapshot.CurrentPeriodEnd
	local.TrialEnd = snapshot.TrialEnd
	local.CancelAtPeriodEnd = snapshot.CancelAtPeriodEnd
	local.StripeScheduleID = snapshot.ScheduleID
	local.PendingQuantity = snapshot.PendingQuantity
	local.PendingUpgradeQuantity = snapshot.PendingUpgradeQuantity
	local.PendingUpgradeExpiresAt = snapshot.PendingUpgradeExpiresAt
	local.PaymentURL = snapshot.PaymentURL
	if checkout.Trial {
		if err := repo.MarkTrialUsed(ctx, organization.ID); err != nil {
			return err
		}
	}

	if err := repo.SaveSubscription(ctx, local); err != nil {
		return err
	}

	recordBillingEvents(ctx, organization.ID, events)

	return nil
}
