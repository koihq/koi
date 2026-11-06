package database

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/koihq/koi/pkg/auth"
	"github.com/koihq/koi/pkg/domain"
)

type GormOnboardingRepository struct {
	db *gorm.DB
}

func NewOnboardingRepository(db *gorm.DB) *GormOnboardingRepository {
	return &GormOnboardingRepository{db: db}
}

// RecordStep keeps the first visit only, so a reload or going back does not change the funnel.
func (r *GormOnboardingRepository) RecordStep(ctx context.Context, step string) error {
	identity, ok := auth.FromContext(ctx)
	if !ok {
		return domain.ErrUnauthorized
	}

	return r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&domain.OnboardingEvent{
		OrganizationID: identity.OrganizationID,
		Step:           step,
	}).Error
}
