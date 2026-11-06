package database

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/koihq/koi/pkg/auth"
	"github.com/koihq/koi/pkg/domain"
)

type GormBillingRepository struct {
	db        *gorm.DB
	cloudMode bool
}

func NewBillingRepository(db *gorm.DB, cloudMode bool) *GormBillingRepository {
	return &GormBillingRepository{db: db, cloudMode: cloudMode}
}

func (r *GormBillingRepository) ReservePromptCapacity(ctx context.Context, organizationID uuid.UUID, additional int64) error {
	if !r.cloudMode {
		return nil
	}

	if err := lockOrganization(ctx, r.db, organizationID); err != nil {
		return err
	}

	return requirePromptCapacity(ctx, r.db, organizationID, additional)
}

func (r *GormBillingRepository) OrganizationForUser(ctx context.Context) (*domain.Organization, error) {
	identity, ok := auth.FromContext(ctx)
	if !ok || identity.OrganizationID == uuid.Nil {
		return nil, domain.ErrUnauthorized
	}

	var organization domain.Organization
	err := r.db.WithContext(ctx).Where("id = ?", identity.OrganizationID).Take(&organization).Error
	if err != nil {
		return nil, mapDBError(err)
	}

	return &organization, nil
}

func (r *GormBillingRepository) ProductAccess(ctx context.Context, organizationID uuid.UUID) (bool, error) {
	identity, ok := auth.FromContext(ctx)
	if !ok || identity.OrganizationID == uuid.Nil || identity.OrganizationID != organizationID {
		return false, domain.ErrUnauthorized
	}
	limit, err := cloudPromptLimit(ctx, r.db, organizationID)
	return limit > 0, err
}

func (r *GormBillingRepository) UsedPromptCount(ctx context.Context, organizationID uuid.UUID) (int64, error) {
	identity, ok := auth.FromContext(ctx)
	if !ok || identity.OrganizationID == uuid.Nil || identity.OrganizationID != organizationID {
		return 0, domain.ErrUnauthorized
	}

	return storedPromptCount(ctx, r.db, organizationID)
}

func (r *GormBillingRepository) SaveCustomer(ctx context.Context, orgID uuid.UUID, customerID string) (string, error) {
	result := r.db.WithContext(ctx).Model(&domain.Organization{}).
		Where("id = ? AND stripe_customer_id IS NULL", orgID).
		Update("stripe_customer_id", customerID)
	if result.Error != nil {
		return "", result.Error
	}
	if result.RowsAffected == 1 {
		return customerID, nil
	}

	var organization domain.Organization
	if err := r.db.WithContext(ctx).Where("id = ?", orgID).Take(&organization).Error; err != nil {
		return "", mapDBError(err)
	}
	if organization.StripeCustomerID == nil {
		return "", errors.New("Stripe customer not saved")
	}

	return *organization.StripeCustomerID, nil
}

func (r *GormBillingRepository) Subscription(ctx context.Context, orgID uuid.UUID) (*domain.BillingSubscription, error) {
	var subscription domain.BillingSubscription
	err := r.db.WithContext(ctx).Where("organization_id = ?", orgID).Take(&subscription).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &subscription, nil
}

func (r *GormBillingRepository) ReserveCheckout(ctx context.Context, orgID uuid.UUID, customerID, priceID string, quantity int) (*domain.BillingCheckoutSession, error) {
	var checkout domain.BillingCheckoutSession
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var organization domain.Organization
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", orgID).Take(&organization).Error; err != nil {
			return mapDBError(err)
		}
		if organization.StripeCustomerID == nil || *organization.StripeCustomerID != customerID {
			return errors.New("Stripe customer mismatch")
		}

		var subscription domain.BillingSubscription
		err := tx.Where("organization_id = ?", orgID).Take(&subscription).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err == nil && subscription.Status != "canceled" && subscription.Status != "incomplete_expired" {
			return domain.ErrBillingSubscriptionExists
		}

		now := time.Now().UTC()
		err = tx.Where("organization_id = ? AND NOT retired AND NOT completed AND ((url = '' AND created_at > ?) OR expires_at > ?)", orgID, now.Add(-24*time.Hour), now).
			Order("created_at DESC").Take(&checkout).Error
		if err == nil {
			if checkout.CustomerID != customerID {
				return domain.ErrBillingCheckoutPending
			}
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		checkout = domain.BillingCheckoutSession{
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

func (r *GormBillingRepository) OrganizationByCustomer(ctx context.Context, customerID string) (*domain.Organization, error) {
	var organization domain.Organization
	err := r.db.WithContext(ctx).Where("stripe_customer_id = ?", customerID).Take(&organization).Error
	return &organization, mapDBError(err)
}

func (r *GormBillingRepository) LockOrganization(ctx context.Context, orgID uuid.UUID) (*domain.Organization, error) {
	var organization domain.Organization
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", orgID).Take(&organization).Error
	return &organization, mapDBError(err)
}

func (r *GormBillingRepository) Checkout(ctx context.Context, id uuid.UUID) (*domain.BillingCheckoutSession, error) {
	var checkout domain.BillingCheckoutSession
	err := r.db.WithContext(ctx).Where("id = ?", id).Take(&checkout).Error
	return &checkout, mapDBError(err)
}

func (r *GormBillingRepository) CheckoutBySession(ctx context.Context, sessionID string) (*domain.BillingCheckoutSession, error) {
	var checkout domain.BillingCheckoutSession
	err := r.db.WithContext(ctx).Where("session_id = ?", sessionID).Take(&checkout).Error
	return &checkout, mapDBError(err)
}

func (r *GormBillingRepository) PendingCheckouts(ctx context.Context, before time.Time, limit int) ([]domain.BillingCheckoutSession, error) {
	var checkouts []domain.BillingCheckoutSession
	err := r.db.WithContext(ctx).Where("NOT completed AND NOT retired AND url <> '' AND updated_at < ?", before).
		Order("updated_at ASC, id ASC").Limit(limit).Find(&checkouts).Error
	return checkouts, err
}

func (r *GormBillingRepository) TouchCheckout(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&domain.BillingCheckoutSession{}).
		Where("id = ? AND NOT completed AND NOT retired", id).Update("updated_at", time.Now().UTC()).Error
}

func (r *GormBillingRepository) CompletedCheckout(ctx context.Context, orgID uuid.UUID, subscriptionID string) (*domain.BillingCheckoutSession, error) {
	var checkout domain.BillingCheckoutSession
	err := r.db.WithContext(ctx).Where("organization_id = ? AND stripe_subscription_id = ? AND completed", orgID, subscriptionID).Take(&checkout).Error
	return &checkout, mapDBError(err)
}

// Finalization accepts retries for the same session, including after its webhook completes.
func (r *GormBillingRepository) FinalizeCheckout(ctx context.Context, reservationID uuid.UUID, sessionID, url string, expiresAt time.Time) error {
	return requireRow(r.db.WithContext(ctx).Model(&domain.BillingCheckoutSession{}).
		Where("id = ? AND NOT retired AND (session_id LIKE 'pending:%' OR session_id = ?)", reservationID, sessionID).
		Updates(map[string]any{"session_id": sessionID, "url": url, "expires_at": expiresAt}))
}

func (r *GormBillingRepository) RetireCheckout(ctx context.Context, id uuid.UUID) error {
	return requireRow(r.db.WithContext(ctx).Model(&domain.BillingCheckoutSession{}).
		Where("id = ? AND NOT completed", id).Update("retired", true))
}

func (r *GormBillingRepository) CompleteCheckout(ctx context.Context, checkout *domain.BillingCheckoutSession, subscriptionID string) error {
	return requireRow(r.db.WithContext(ctx).Model(checkout).
		Updates(map[string]any{"completed": true, "stripe_subscription_id": subscriptionID}))
}

func (r *GormBillingRepository) SaveSubscription(ctx context.Context, subscription *domain.BillingSubscription) error {
	return r.db.WithContext(ctx).Save(subscription).Error
}

func (r *GormBillingRepository) RecordEvent(ctx context.Context, eventID string) (bool, error) {
	if eventID == "" {
		return false, errors.New("missing Stripe event ID")
	}
	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).
		Create(&domain.StripeWebhookEvent{StripeEventID: eventID})
	return result.RowsAffected == 1, result.Error
}

func (r *GormBillingRepository) MarkTrialUsed(ctx context.Context, orgID uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&domain.Organization{}).
		Where("id = ? AND trial_used_at IS NULL", orgID).Update("trial_used_at", time.Now().UTC()).Error
}

func (r *GormBillingRepository) ReserveChange(ctx context.Context, orgID uuid.UUID, key string, quantity int) (*domain.BillingChangeRequest, error) {
	var request domain.BillingChangeRequest
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := NewBillingRepository(tx, r.cloudMode)
		if _, err := repo.LockOrganization(ctx, orgID); err != nil {
			return err
		}

		err := tx.Where("organization_id = ? AND idempotency_key = ?", orgID, key).Take(&request).Error
		if err == nil {
			if request.Quantity != quantity {
				return domain.ErrIdempotencyConflict
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
			return domain.ErrBillingConflict
		}

		request = domain.BillingChangeRequest{
			OrganizationID: orgID,
			IdempotencyKey: key,
			SubscriptionID: subscription.StripeSubscriptionID,
			Quantity:       quantity,
		}
		return tx.Create(&request).Error
	})
	return &request, err
}

func (r *GormBillingRepository) PendingChanges(ctx context.Context, orgID uuid.UUID) ([]domain.BillingChangeRequest, error) {
	var requests []domain.BillingChangeRequest
	err := r.db.WithContext(ctx).Where("organization_id = ? AND result IS NULL AND error_code = ''", orgID).
		Order("created_at ASC, id ASC").Find(&requests).Error
	return requests, err
}

func (r *GormBillingRepository) SaveChange(ctx context.Context, request *domain.BillingChangeRequest) error {
	return r.db.WithContext(ctx).Save(request).Error
}

var _ domain.BillingRepository = (*GormBillingRepository)(nil)
