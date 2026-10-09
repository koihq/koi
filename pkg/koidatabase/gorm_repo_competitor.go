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
	return &GormCompetitorRepository{
		Repository: newScopedRepository[koidomain.Competitor](db, childOwnerScope("competitors")),
	}
}

func (r *GormCompetitorRepository) Create(ctx context.Context, competitor *koidomain.Competitor) error {
	if _, err := r.Repository.query(ctx); err != nil {
		return err
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := NewGormBrandRepository(tx).LockForUpdate(ctx, competitor.BrandID); err != nil {
			return err
		}
		if err := NewGormCompetitorRepository(tx).Repository.Create(ctx, competitor); err != nil {
			return err
		}
		return invalidateReview(tx, competitor.BrandID, "competitors")
	})
}

func applyCompetitorFilter(db *gorm.DB, filter koidomain.CompetitorFilter) *gorm.DB {
	db = applyChildListFilter(db, filter.ID, filter.BrandID, filter.Confirmed)
	if filter.Domain != "" {
		db = db.Where("domain = ?", filter.Domain)
	}

	if filter.Query != "" {
		pattern := likePattern(filter.Query)
		db = db.Where("(name ILIKE ? ESCAPE '!' OR domain ILIKE ? ESCAPE '!' OR github_repo ILIKE ? ESCAPE '!')", pattern, pattern, pattern)
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
	if _, err := r.Repository.query(ctx); err != nil {
		return err
	}

	values := &koidomain.Competitor{}
	columns := make([]string, 0, 1)
	if update.Confirmed != nil {
		values.Confirmed = *update.Confirmed
		columns = append(columns, "confirmed")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := NewGormBrandRepository(tx).LockForUpdate(ctx, filter.BrandID); err != nil {
			return err
		}
		if err := NewGormCompetitorRepository(tx).updateBy(ctx, func(db *gorm.DB) *gorm.DB { return applyCompetitorFilter(db, filter) }, values, columns, filter.ID != uuid.Nil); err != nil {
			return err
		}
		if update.Confirmed != nil && !*update.Confirmed {
			return invalidateReview(tx, filter.BrandID, "competitors")
		}
		return nil
	})
}

func (r *GormCompetitorRepository) Delete(ctx context.Context, filter koidomain.CompetitorFilter) error {
	if _, err := r.Repository.query(ctx); err != nil {
		return err
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := NewGormBrandRepository(tx).LockForUpdate(ctx, filter.BrandID); err != nil {
			return err
		}
		if err := NewGormCompetitorRepository(tx).deleteBy(ctx, func(db *gorm.DB) *gorm.DB { return applyCompetitorFilter(db, filter) }, filter.ID != uuid.Nil); err != nil {
			return err
		}
		return invalidateReview(tx, filter.BrandID, "competitors")
	})
}
