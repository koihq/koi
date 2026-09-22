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
	return &GormCompetitorRepository{Repository: NewRepository[koidomain.Competitor](db)}
}

func applyCompetitorFilter(db *gorm.DB, filter koidomain.CompetitorFilter) *gorm.DB {
	if filter.ID != uuid.Nil {
		db = db.Where("id = ?", filter.ID)
	}
	if filter.BrandID != uuid.Nil {
		db = db.Where("brand_id = ?", filter.BrandID)
	}
	if filter.Confirmed != nil {
		db = db.Where("confirmed = ?", *filter.Confirmed)
	}
	return db
}

func (r *GormCompetitorRepository) Get(ctx context.Context, filter koidomain.CompetitorFilter) (*koidomain.Competitor, error) {
	var competitor koidomain.Competitor
	err := applyCompetitorFilter(r.db.WithContext(ctx), filter).Take(&competitor).Error
	if err != nil {
		return nil, mapDBError(err)
	}
	return &competitor, nil
}

func (r *GormCompetitorRepository) List(ctx context.Context, filter koidomain.CompetitorFilter) (*koidomain.Page[koidomain.Competitor], error) {
	return paginate[koidomain.Competitor](
		applyCompetitorFilter(r.db.WithContext(ctx), filter),
		filter.Page,
		"created_at ASC, id ASC",
	)
}

func (r *GormCompetitorRepository) Update(ctx context.Context, filter koidomain.CompetitorFilter, update koidomain.CompetitorUpdate) error {
	values := &koidomain.Competitor{}
	columns := make([]string, 0, 4)
	if update.Name != nil {
		values.Name = *update.Name
		columns = append(columns, "name")
	}
	if update.Domain != nil {
		values.Domain = *update.Domain
		columns = append(columns, "domain")
	}
	if update.Aliases != nil {
		values.Aliases = *update.Aliases
		columns = append(columns, "aliases")
	}
	if update.Confirmed != nil {
		values.Confirmed = *update.Confirmed
		columns = append(columns, "confirmed")
	}
	result := applyCompetitorFilter(r.db.WithContext(ctx).Model(&koidomain.Competitor{}), filter).
		Select(columns).Updates(values)
	return updated(result, filter.ID != uuid.Nil)
}

func (r *GormCompetitorRepository) Delete(ctx context.Context, filter koidomain.CompetitorFilter) error {
	result := applyCompetitorFilter(r.db.WithContext(ctx), filter).Delete(&koidomain.Competitor{})
	return updated(result, filter.ID != uuid.Nil)
}
