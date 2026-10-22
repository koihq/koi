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

func (r *BillingRepository) ProductAccess(ctx context.Context, userID uuid.UUID) (bool, error) {
	var membership koidomain.Membership
	err := r.db.WithContext(ctx).Select("organization_id").
		Where("user_id = ?", userID).Take(&membership).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, koidomain.ErrUnauthorized
	}
	if err != nil {
		return false, err
	}
	if membership.OrganizationID == uuid.Nil {
		return false, koidomain.ErrUnauthorized
	}

	limit, err := cloudPromptLimit(ctx, r.db, membership.OrganizationID)
	return limit > 0, err
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
		if err == nil && subscription.Status != "canceled" && subscription.Status != "incomplete_expired" {
			return koidomain.ErrBillingSubscriptionExists
		}

		now := time.Now().UTC()
		err = tx.Where("organization_id = ? AND NOT retired AND NOT completed AND ((url = '' AND created_at > ?) OR expires_at > ?)", orgID, now.Add(-24*time.Hour), now).
			Order("created_at DESC").Take(&checkout).Error
		if err == nil {
			if checkout.CustomerID != customerID {
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
			Trial:          organization.TrialUsedAt == nil,
		}
		return tx.Create(&checkout).Error
	})
	if err != nil {
		return nil, err
	}

	return &checkout, nil
}

func (r *BillingRepository) OrganizationByCustomer(ctx context.Context, customerID string) (*koidomain.Organization, error) {
	var organization koidomain.Organization
	err := r.db.WithContext(ctx).Where("stripe_customer_id = ?", customerID).Take(&organization).Error
	return &organization, mapDBError(err)
}

func (r *BillingRepository) LockOrganization(ctx context.Context, orgID uuid.UUID) (*koidomain.Organization, error) {
	var organization koidomain.Organization
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", orgID).Take(&organization).Error
	return &organization, mapDBError(err)
}

func (r *BillingRepository) Checkout(ctx context.Context, id uuid.UUID) (*koidomain.BillingCheckoutSession, error) {
	var checkout koidomain.BillingCheckoutSession
	err := r.db.WithContext(ctx).Where("id = ?", id).Take(&checkout).Error
	return &checkout, mapDBError(err)
}

func (r *BillingRepository) CheckoutBySession(ctx context.Context, sessionID string) (*koidomain.BillingCheckoutSession, error) {
	var checkout koidomain.BillingCheckoutSession
	err := r.db.WithContext(ctx).Where("session_id = ?", sessionID).Take(&checkout).Error
	return &checkout, mapDBError(err)
}

func (r *BillingRepository) CompletedCheckout(ctx context.Context, orgID uuid.UUID, subscriptionID string) (*koidomain.BillingCheckoutSession, error) {
	var checkout koidomain.BillingCheckoutSession
	err := r.db.WithContext(ctx).Where("organization_id = ? AND stripe_subscription_id = ? AND completed", orgID, subscriptionID).Take(&checkout).Error
	return &checkout, mapDBError(err)
}

// Finalization accepts retries for the same session, including after its webhook completes.
func (r *BillingRepository) FinalizeCheckout(ctx context.Context, reservationID uuid.UUID, sessionID, url string, expiresAt time.Time) error {
	return updated(r.db.WithContext(ctx).Model(&koidomain.BillingCheckoutSession{}).
		Where("id = ? AND NOT retired AND (session_id LIKE 'pending:%' OR session_id = ?)", reservationID, sessionID).
		Updates(map[string]any{"session_id": sessionID, "url": url, "expires_at": expiresAt}), true)
}

func (r *BillingRepository) RetireCheckout(ctx context.Context, id uuid.UUID) error {
	return updated(r.db.WithContext(ctx).Model(&koidomain.BillingCheckoutSession{}).
		Where("id = ? AND NOT completed", id).Update("retired", true), true)
}

func (r *BillingRepository) CompleteCheckout(ctx context.Context, checkout *koidomain.BillingCheckoutSession, subscriptionID string) error {
	return updated(r.db.WithContext(ctx).Model(checkout).
		Updates(map[string]any{"completed": true, "stripe_subscription_id": subscriptionID}), true)
}

func (r *BillingRepository) SaveSubscription(ctx context.Context, subscription *koidomain.BillingSubscription) error {
	return r.db.WithContext(ctx).Save(subscription).Error
}

func (r *BillingRepository) RecordEvent(ctx context.Context, eventID string) (bool, error) {
	if eventID == "" {
		return false, errors.New("missing Stripe event ID")
	}
	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).
		Create(&koidomain.StripeWebhookEvent{StripeEventID: eventID})
	return result.RowsAffected == 1, result.Error
}

func (r *BillingRepository) MarkTrialUsed(ctx context.Context, orgID uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&koidomain.Organization{}).
		Where("id = ? AND trial_used_at IS NULL", orgID).Update("trial_used_at", time.Now().UTC()).Error
}

func (r *BillingRepository) ReserveChange(ctx context.Context, orgID uuid.UUID, key string, quantity int) (*koidomain.BillingChangeRequest, error) {
	var request koidomain.BillingChangeRequest
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := NewBillingRepository(tx)
		if _, err := repo.LockOrganization(ctx, orgID); err != nil {
			return err
		}

		err := tx.Where("organization_id = ? AND idempotency_key = ?", orgID, key).Take(&request).Error
		if err == nil {
			if request.Quantity != quantity {
				return koidomain.ErrIdempotencyConflict
			}
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		subscription, err := repo.Subscription(ctx, orgID)
		if err != nil {
			return err
		}
		if subscription == nil {
			return koidomain.ErrBillingConflict
		}

		request = koidomain.BillingChangeRequest{
			OrganizationID: orgID,
			IdempotencyKey: key,
			SubscriptionID: subscription.StripeSubscriptionID,
			Quantity:       quantity,
		}
		return tx.Create(&request).Error
	})
	return &request, err
}

func (r *BillingRepository) PendingChanges(ctx context.Context, orgID uuid.UUID) ([]koidomain.BillingChangeRequest, error) {
	var requests []koidomain.BillingChangeRequest
	err := r.db.WithContext(ctx).Where("organization_id = ? AND result IS NULL AND error_code = ''", orgID).
		Order("created_at ASC, id ASC").Find(&requests).Error
	return requests, err
}

func (r *BillingRepository) SaveChange(ctx context.Context, request *koidomain.BillingChangeRequest) error {
	return r.db.WithContext(ctx).Save(request).Error
}

var _ koidomain.BillingRepository = (*BillingRepository)(nil)
