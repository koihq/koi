package koidatabase

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/koidomain"
)

type GormCompetitorRepository struct{ db *gorm.DB }

func NewGormCompetitorRepository(db *gorm.DB) *GormCompetitorRepository {
	return &GormCompetitorRepository{db: db}
}

func competitorBrand(brandID uuid.UUID) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB { return db.Where("brand_id = ?", brandID) }
}

func (r *GormCompetitorRepository) Create(ctx context.Context, competitor *koidomain.Competitor) error {
	return mapDBError(r.db.WithContext(ctx).Create(competitor).Error)
}

func (r *GormCompetitorRepository) ListByBrandID(ctx context.Context, brandID uuid.UUID) ([]koidomain.Competitor, error) {
	competitors := make([]koidomain.Competitor, 0)
	err := r.db.WithContext(ctx).Scopes(competitorBrand(brandID)).Order("created_at ASC, id ASC").Find(&competitors).Error
	return competitors, mapDBError(err)
}

func (r *GormCompetitorRepository) GetByIDAndBrandID(ctx context.Context, id, brandID uuid.UUID) (*koidomain.Competitor, error) {
	var competitor koidomain.Competitor
	err := r.db.WithContext(ctx).Scopes(competitorBrand(brandID)).Where("id = ?", id).Take(&competitor).Error
	if err != nil {
		return nil, mapDBError(err)
	}
	return &competitor, nil
}

func (r *GormCompetitorRepository) UpdateByIDAndBrandID(ctx context.Context, brandID uuid.UUID, competitor *koidomain.Competitor) error {
	result := r.db.WithContext(ctx).Model(&koidomain.Competitor{}).Scopes(competitorBrand(brandID)).
		Where("id = ?", competitor.ID).Select("name", "domain", "aliases").Updates(competitor)
	return affected(result)
}

func (r *GormCompetitorRepository) DeleteByIDAndBrandID(ctx context.Context, id, brandID uuid.UUID) error {
	return affected(r.db.WithContext(ctx).Scopes(competitorBrand(brandID)).Where("id = ?", id).Delete(&koidomain.Competitor{}))
}

func (r *GormCompetitorRepository) ConfirmByIDAndBrandID(ctx context.Context, id, brandID uuid.UUID) error {
	result := r.db.WithContext(ctx).Model(&koidomain.Competitor{}).Scopes(competitorBrand(brandID)).
		Where("id = ?", id).Update("confirmed", true)
	return affected(result)
}

func (r *GormCompetitorRepository) ConfirmAllByBrandID(ctx context.Context, brandID uuid.UUID) error {
	return mapDBError(r.db.WithContext(ctx).Model(&koidomain.Competitor{}).Scopes(competitorBrand(brandID)).
		Where("confirmed = ?", false).Update("confirmed", true).Error)
}
