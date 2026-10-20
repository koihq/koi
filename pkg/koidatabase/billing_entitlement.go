package koidatabase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/koihq/koi/pkg/koidomain"
)

// lockOrganization serializes prompt activation across all brands in an organization.
func lockOrganization(ctx context.Context, db *gorm.DB, organizationID uuid.UUID) error {
	var organization koidomain.Organization
	return mapDBError(db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", organizationID).Take(&organization).Error)
}

func cloudPromptLimit(ctx context.Context, db *gorm.DB, organizationID uuid.UUID) (int, error) {
	var subscription koidomain.BillingSubscription
	err := db.WithContext(ctx).Where("organization_id = ?", organizationID).Take(&subscription).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	return subscription.PromptLimit(time.Now().UTC()), nil
}

func activePromptCount(ctx context.Context, db *gorm.DB, organizationID uuid.UUID) (int64, error) {
	var count int64
	err := db.WithContext(ctx).Model(&koidomain.Prompt{}).
		Joins("JOIN brands ON brands.id = prompts.brand_id").
		Where("brands.organization_id = ? AND prompts.confirmed AND NOT prompts.inactive", organizationID).
		Count(&count).Error

	return count, err
}

func requirePromptCapacity(ctx context.Context, db *gorm.DB, organizationID uuid.UUID, additional int64) error {
	limit, err := cloudPromptLimit(ctx, db, organizationID)
	if err != nil {
		return err
	}
	count, err := activePromptCount(ctx, db, organizationID)
	if err != nil {
		return err
	}
	if limit == 0 || count+additional > int64(limit) {
		return koidomain.ErrPromptLimitReached
	}

	return nil
}

func cloudTrackingAllowed(ctx context.Context, db *gorm.DB, organizationID uuid.UUID) (bool, error) {
	limit, err := cloudPromptLimit(ctx, db, organizationID)
	if err != nil || limit == 0 {
		return false, err
	}
	count, err := activePromptCount(ctx, db, organizationID)
	if err != nil {
		return false, err
	}

	return count > 0 && count <= int64(limit), nil
}
