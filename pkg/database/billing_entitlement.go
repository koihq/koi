package database

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/koihq/koi/pkg/domain"
)

// lockOrganization serializes prompt activation across all brands in an organization.
func lockOrganization(ctx context.Context, db *gorm.DB, organizationID uuid.UUID) error {
	var organization domain.Organization
	return mapDBError(db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", organizationID).Take(&organization).Error)
}

// cloudSubscription returns a zero subscription, which grants nothing, when none exists.
func cloudSubscription(ctx context.Context, db *gorm.DB, organizationID uuid.UUID) (domain.BillingSubscription, error) {
	var subscription domain.BillingSubscription
	err := db.WithContext(ctx).Where("organization_id = ?", organizationID).Take(&subscription).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.BillingSubscription{}, nil
	}

	return subscription, err
}

func cloudEntitlement(ctx context.Context, db *gorm.DB, organizationID uuid.UUID) (domain.BillingSubscription, *time.Time, error) {
	subscription, err := cloudSubscription(ctx, db, organizationID)
	if err != nil {
		return domain.BillingSubscription{}, nil, err
	}

	var organization domain.Organization
	if err := db.WithContext(ctx).Select("trial_used_at").Where("id = ?", organizationID).Take(&organization).Error; err != nil {
		return domain.BillingSubscription{}, nil, mapDBError(err)
	}

	return subscription, organization.TrialUsedAt, nil
}

func cloudPromptLimit(ctx context.Context, db *gorm.DB, organizationID uuid.UUID) (int, error) {
	subscription, trialUsedAt, err := cloudEntitlement(ctx, db, organizationID)
	if err != nil {
		return 0, err
	}

	return domain.EffectivePromptLimit(subscription, trialUsedAt, time.Now().UTC()), nil
}

func cloudFirstLook(ctx context.Context, db *gorm.DB, organizationID uuid.UUID) (bool, error) {
	subscription, trialUsedAt, err := cloudEntitlement(ctx, db, organizationID)
	if err != nil {
		return false, err
	}

	return domain.IsFirstLook(subscription, trialUsedAt), nil
}

func activePromptCount(ctx context.Context, db *gorm.DB, organizationID uuid.UUID) (int64, error) {
	var count int64
	err := db.WithContext(ctx).Model(&domain.Prompt{}).
		Joins("JOIN brands ON brands.id = prompts.brand_id").
		Where("brands.organization_id = ? AND prompts.confirmed AND NOT prompts.inactive", organizationID).
		Count(&count).Error

	return count, err
}

func storedPromptCount(ctx context.Context, db *gorm.DB, organizationID uuid.UUID) (int64, error) {
	var count int64
	err := db.WithContext(ctx).Model(&domain.Prompt{}).
		Joins("JOIN brands ON brands.id = prompts.brand_id").
		Where("brands.organization_id = ? AND NOT prompts.inactive", organizationID).
		Count(&count).Error
	return count, err
}

func requirePromptCapacity(ctx context.Context, db *gorm.DB, organizationID uuid.UUID, additional int64) error {
	limit, err := cloudPromptLimit(ctx, db, organizationID)
	if err != nil {
		return err
	}
	count, err := storedPromptCount(ctx, db, organizationID)
	if err != nil {
		return err
	}
	if limit == 0 || count+additional > int64(limit) {
		return domain.ErrPromptLimitReached
	}

	return nil
}

// cloudCollectionAllowed gates a collection. A First Look organization may only run its first.
func cloudCollectionAllowed(ctx context.Context, db *gorm.DB, organizationID uuid.UUID, firstRun bool) (bool, error) {
	firstLook, err := cloudFirstLook(ctx, db, organizationID)
	if err != nil || (firstLook && !firstRun) {
		return false, err
	}
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
