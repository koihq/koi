package koidatabase

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/koidomain"
)

type GormPromptRepository struct {
	*Repository[koidomain.Prompt]
}

func NewGormPromptRepository(db *gorm.DB) *GormPromptRepository {
	return &GormPromptRepository{Repository: NewRepository[koidomain.Prompt](db)}
}

func applyPromptFilter(db *gorm.DB, filter koidomain.PromptFilter) *gorm.DB {
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

func (r *GormPromptRepository) Get(ctx context.Context, filter koidomain.PromptFilter) (*koidomain.Prompt, error) {
	var prompt koidomain.Prompt
	err := applyPromptFilter(r.db.WithContext(ctx), filter).Take(&prompt).Error
	if err != nil {
		return nil, mapDBError(err)
	}
	return &prompt, nil
}

func (r *GormPromptRepository) List(ctx context.Context, filter koidomain.PromptFilter) (*koidomain.Page[koidomain.Prompt], error) {
	return paginate[koidomain.Prompt](
		applyPromptFilter(r.db.WithContext(ctx), filter),
		filter.Page,
		"created_at ASC, id ASC",
	)
}

func (r *GormPromptRepository) Update(ctx context.Context, filter koidomain.PromptFilter, update koidomain.PromptUpdate) error {
	values := &koidomain.Prompt{}
	columns := make([]string, 0, 2)
	if update.Text != nil {
		values.Text = *update.Text
		columns = append(columns, "text")
	}
	if update.Confirmed != nil {
		values.Confirmed = *update.Confirmed
		columns = append(columns, "confirmed")
	}
	result := applyPromptFilter(r.db.WithContext(ctx).Model(&koidomain.Prompt{}), filter).
		Select(columns).Updates(values)
	return updated(result, filter.ID != uuid.Nil)
}

func (r *GormPromptRepository) Delete(ctx context.Context, filter koidomain.PromptFilter) error {
	result := applyPromptFilter(r.db.WithContext(ctx), filter).Delete(&koidomain.Prompt{})
	return updated(result, filter.ID != uuid.Nil)
}

func updated(result *gorm.DB, requireMatch bool) error {
	if err := mapDBError(result.Error); err != nil {
		return err
	}
	if requireMatch && result.RowsAffected == 0 {
		return koidomain.ErrNotFound
	}
	return nil
}
