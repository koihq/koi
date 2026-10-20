package koidatabase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/koihq/koi/pkg/koiauth"
	"github.com/koihq/koi/pkg/koidomain"
	"github.com/koihq/koi/pkg/koipricing"
)

type BillingRepository struct{ db *gorm.DB }

func NewBillingRepository(db *gorm.DB) *BillingRepository { return &BillingRepository{db: db} }

func (r *BillingRepository) OrganizationForUser(ctx context.Context) (*koidomain.Organization, error) {
	identity, ok := koiauth.FromContext(ctx)
	if !ok {
		return nil, koidomain.ErrUnauthorized
	}

	var organization koidomain.Organization
	err := r.db.WithContext(ctx).Joins("JOIN memberships ON memberships.organization_id = organizations.id").
		Where("memberships.user_id = ?", identity.UserID).Take(&organization).Error
	if err != nil {
		return nil, mapDBError(err)
	}

	return &organization, nil
}

func (r *BillingRepository) SaveCustomer(ctx context.Context, orgID uuid.UUID, customerID string) (string, error) {
	result := r.db.WithContext(ctx).Model(&koidomain.Organization{}).
		Where("id = ? AND stripe_customer_id IS NULL", orgID).
		Update("stripe_customer_id", customerID)
	if result.Error != nil {
		return "", result.Error
	}
	if result.RowsAffected == 1 {
		return customerID, nil
	}

	var organization koidomain.Organization
	if err := r.db.WithContext(ctx).Where("id = ?", orgID).Take(&organization).Error; err != nil {
		return "", mapDBError(err)
	}
	if organization.StripeCustomerID == nil {
		return "", errors.New("Stripe customer not saved")
	}

	return *organization.StripeCustomerID, nil
}

func (r *BillingRepository) Subscription(ctx context.Context, orgID uuid.UUID) (*koidomain.BillingSubscription, error) {
	var subscription koidomain.BillingSubscription
	err := r.db.WithContext(ctx).Where("organization_id = ?", orgID).Take(&subscription).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &subscription, nil
}

func (r *BillingRepository) ReserveCheckout(ctx context.Context, orgID uuid.UUID, customerID, priceID string, quantity int) (*koidomain.BillingCheckoutSession, error) {
	var checkout koidomain.BillingCheckoutSession
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var organization koidomain.Organization
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", orgID).Take(&organization).Error; err != nil {
			return mapDBError(err)
		}
		if organization.StripeCustomerID == nil || *organization.StripeCustomerID != customerID {
			return errors.New("Stripe customer mismatch")
		}

		var subscription koidomain.BillingSubscription
		err := tx.Where("organization_id = ?", orgID).Take(&subscription).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err == nil && subscription.Status != "canceled" {
			return koidomain.ErrBillingSubscriptionExists
		}

		now := time.Now().UTC()
		err = tx.Where("organization_id = ? AND ((url = '' AND created_at > ?) OR expires_at > ?)", orgID, now.Add(-24*time.Hour), now).
			Order("created_at DESC").Take(&checkout).Error
		if err == nil {
			if checkout.CustomerID != customerID || checkout.PriceID != priceID || checkout.Quantity != quantity || checkout.Completed {
				return koidomain.ErrBillingCheckoutPending
			}
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		checkout = koidomain.BillingCheckoutSession{
			OrganizationID: orgID,
			SessionID:      "pending:" + uuid.NewString(),
			CustomerID:     customerID,
			PriceID:        priceID,
			Quantity:       quantity,
		}
		return tx.Create(&checkout).Error
	})
	if err != nil {
		return nil, err
	}

	return &checkout, nil
}

func (r *BillingRepository) FinalizeCheckout(ctx context.Context, reservationID uuid.UUID, sessionID, url string, expiresAt time.Time) error {
	return updated(r.db.WithContext(ctx).Model(&koidomain.BillingCheckoutSession{}).
		Where("id = ? AND session_id LIKE 'pending:%' AND NOT completed", reservationID).
		Updates(map[string]any{"session_id": sessionID, "url": url, "expires_at": expiresAt}), true)
}

func (r *BillingRepository) PendingDowngrade(ctx context.Context, orgID uuid.UUID, scheduleID string, quantity int) error {
	return r.db.WithContext(ctx).Model(&koidomain.BillingSubscription{}).
		Where("organization_id = ?", orgID).
		Updates(map[string]any{"stripe_schedule_id": scheduleID, "pending_quantity": quantity}).Error
}

func (r *BillingRepository) CompleteCheckout(ctx context.Context, eventID, sessionID, customerID, organizationID, subscriptionID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		fresh, err := recordStripeEvent(tx, eventID)
		if err != nil || !fresh {
			return err
		}

		var checkout koidomain.BillingCheckoutSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("session_id = ?", sessionID).Take(&checkout).Error; err != nil {
			return mapDBError(err)
		}
		if checkout.OrganizationID.String() != organizationID || checkout.CustomerID != customerID ||
			(checkout.StripeSubscriptionID != nil && *checkout.StripeSubscriptionID != subscriptionID) || subscriptionID == "" {
			return errors.New("Stripe checkout identity mismatch")
		}

		var organization koidomain.Organization
		if err := tx.Where("id = ?", checkout.OrganizationID).Take(&organization).Error; err != nil {
			return err
		}
		if organization.StripeCustomerID == nil || *organization.StripeCustomerID != customerID {
			return errors.New("Stripe customer does not belong to organization")
		}

		return tx.Model(&checkout).Updates(map[string]any{
			"completed":              true,
			"stripe_subscription_id": subscriptionID,
		}).Error
	})
}

func (r *BillingRepository) SyncSubscription(ctx context.Context, eventID, organizationID, customerID, subscriptionID, priceID, status string, quantity int, currentPeriodEnd, trialEnd *time.Time, cancelAtPeriodEnd bool) error {
	orgID, err := uuid.Parse(organizationID)
	_, priceErr := koipricing.Current().Quote(quantity)
	if err != nil || priceErr != nil || subscriptionID == "" || priceID == "" {
		return errors.New("invalid Stripe subscription identity or quantity")
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		fresh, err := recordStripeEvent(tx, eventID)
		if err != nil || !fresh {
			return err
		}

		var organization koidomain.Organization
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", orgID).Take(&organization).Error; err != nil {
			return mapDBError(err)
		}
		if organization.StripeCustomerID == nil || *organization.StripeCustomerID != customerID {
			return errors.New("Stripe subscription customer mismatch")
		}

		var checkout koidomain.BillingCheckoutSession
		if err := tx.Where("organization_id = ? AND customer_id = ? AND stripe_subscription_id = ? AND completed", orgID, customerID, subscriptionID).
			Take(&checkout).Error; err != nil {
			return mapDBError(err)
		}
		if checkout.PriceID != priceID {
			return errors.New("Stripe subscription price mismatch")
		}

		var subscription koidomain.BillingSubscription
		err = tx.Where("organization_id = ?", orgID).Take(&subscription).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err == nil && subscription.StripeSubscriptionID != subscriptionID && subscription.Status != "canceled" {
			return errors.New("organization already has a subscription")
		}
		if (errors.Is(err, gorm.ErrRecordNotFound) || subscription.StripeSubscriptionID != subscriptionID) && checkout.Quantity != quantity {
			return errors.New("Stripe subscription quantity differs from Checkout")
		}

		subscription.OrganizationID = orgID
		subscription.StripeSubscriptionID = subscriptionID
		subscription.StripePriceID = priceID
		subscription.Quantity = quantity
		subscription.Status = status
		subscription.CurrentPeriodEnd = currentPeriodEnd
		subscription.TrialEnd = trialEnd
		subscription.CancelAtPeriodEnd = cancelAtPeriodEnd
		if subscription.PendingQuantity != nil && quantity == *subscription.PendingQuantity {
			subscription.PendingQuantity = nil
			subscription.StripeScheduleID = nil
		}
		if err == gorm.ErrRecordNotFound {
			return tx.Create(&subscription).Error
		}

		return tx.Save(&subscription).Error
	})
}

func recordStripeEvent(tx *gorm.DB, eventID string) (bool, error) {
	if eventID == "" {
		return false, errors.New("missing Stripe event ID")
	}
	result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&koidomain.StripeWebhookEvent{StripeEventID: eventID})
	return result.RowsAffected == 1, result.Error
}
