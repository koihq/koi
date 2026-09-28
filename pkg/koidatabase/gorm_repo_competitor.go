package koidatabase

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/koidomain"
)

type GormCompetitorRepository struct {
	*Repository[koidomain.Competitor]
}

func NewGormCompetitorRepository(db *gorm.DB) *GormCompetitorRepository {
	return &GormCompetitorRepository{Repository: newScopedRepository[koidomain.Competitor](db, childOwnerScope("competitors"))}
}

func (r *GormCompetitorRepository) Create(ctx context.Context, competitor *koidomain.Competitor) error {
	if _, err := NewGormBrandRepository(r.db).Get(ctx, competitor.BrandID); err != nil {
		return err
	}
	return r.Repository.Create(ctx, competitor)
}

func applyCompetitorFilter(db *gorm.DB, filter koidomain.CompetitorFilter) *gorm.DB {
	if filter.ID != uuid.Nil {
		db = db.Where("id = ?", filter.ID)
	}
	if filter.BrandID != uuid.Nil {
		db = db.Where("brand_id = ?", filter.BrandID)
	}
	if filter.Domain != "" {
		db = db.Where("domain = ?", filter.Domain)
	}
	if filter.Confirmed != nil {
		db = db.Where("confirmed = ?", *filter.Confirmed)
	}
	if filter.Query != "" {
		pattern := likePattern(filter.Query)
		db = db.Where("(name ILIKE ? ESCAPE '!' OR domain ILIKE ? ESCAPE '!')", pattern, pattern)
	}
	return db
}

func (r *GormCompetitorRepository) Get(ctx context.Context, filter koidomain.CompetitorFilter) (*koidomain.Competitor, error) {
	return r.getBy(ctx, func(db *gorm.DB) *gorm.DB { return applyCompetitorFilter(db, filter) })
}

func (r *GormCompetitorRepository) List(ctx context.Context, filter koidomain.CompetitorFilter) (*koidomain.Page[koidomain.Competitor], error) {
	return r.listBy(ctx, func(db *gorm.DB) *gorm.DB { return applyCompetitorFilter(db, filter) }, filter.Page, "created_at ASC, id ASC")
}

func (r *GormCompetitorRepository) Update(ctx context.Context, filter koidomain.CompetitorFilter, update koidomain.CompetitorUpdate) error {
	values := &koidomain.Competitor{}
	columns := make([]string, 0, 1)
	if update.Confirmed != nil {
		values.Confirmed = *update.Confirmed
		columns = append(columns, "confirmed")
	}
	return r.updateBy(ctx, func(db *gorm.DB) *gorm.DB { return applyCompetitorFilter(db, filter) }, values, columns, filter.ID != uuid.Nil)
}

func (r *GormCompetitorRepository) Delete(ctx context.Context, filter koidomain.CompetitorFilter) error {
	return r.deleteBy(ctx, func(db *gorm.DB) *gorm.DB { return applyCompetitorFilter(db, filter) }, filter.ID != uuid.Nil)
}
