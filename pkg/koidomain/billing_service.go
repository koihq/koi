package koidomain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/koihq/koi/pkg/koipricing"
)

type billingService struct {
	store     Store
	gateway   BillingGateway
	priceID   string
	publicURL string
}

func NewBillingCommander(store Store, gateway BillingGateway, priceID, publicURL string) BillingCommander {
	return &billingService{store: store, gateway: gateway, priceID: priceID, publicURL: publicURL}
}

func NewBillingQuerier(store Store, gateway BillingGateway, priceID string) BillingQuerier {
	return &billingService{store: store, gateway: gateway, priceID: priceID}
}

func (s *billingService) Catalog(ctx context.Context) (BillingCatalog, error) {
	if err := s.gateway.ValidatePrice(ctx, s.priceID); err != nil {
		return BillingCatalog{}, err
	}

	catalog := koipricing.Current()
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
	if _, err := koipricing.Current().Quote(quantity); err != nil {
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
	status := BillingStatus{Subscription: subscription, TrialAvailable: organization.TrialUsedAt == nil}
	if subscription != nil {
		status.PromptLimit = subscription.PromptLimit(time.Now().UTC())
	}

	return status, nil
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
		err = s.store.Atomic(ctx, func(store Store) error {
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
				if err := repo.CompleteCheckout(ctx, checkout, remote.SubscriptionID); err != nil {
					return err
				}
				completed = true
				return s.refresh(ctx, repo, locked, remote.SubscriptionID)
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
		err := s.store.Atomic(ctx, func(store Store) error {
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

	return s.store.Atomic(ctx, func(store Store) error {
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
			if err := repo.CompleteCheckout(ctx, checkout, event.SubscriptionID); err != nil {
				return err
			}
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

	return repo.SaveSubscription(ctx, local)
}
