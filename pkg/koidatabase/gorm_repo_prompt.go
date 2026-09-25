package koidatabase

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/koidomain"
)

type GormPromptRepository struct {
	*Repository[koidomain.Prompt]
}

func NewGormPromptRepository(db *gorm.DB) *GormPromptRepository {
	return &GormPromptRepository{Repository: newScopedRepository[koidomain.Prompt](db, childOwnerScope("prompts"))}
}

func (r *GormPromptRepository) Create(ctx context.Context, prompt *koidomain.Prompt) error {
	if _, err := NewGormBrandRepository(r.db).Get(ctx, prompt.BrandID); err != nil {
		return err
	}
	return r.Repository.Create(ctx, prompt)
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
	if filter.Query != "" {
		db = db.Where("text ILIKE ? ESCAPE '!'", likePattern(filter.Query))
	}
	return db
}

func likePattern(query string) string {
	replacer := strings.NewReplacer(`!`, `!!`, `%`, `!%`, `_`, `!_`)
	return "%" + replacer.Replace(query) + "%"
}

func (r *GormPromptRepository) Get(ctx context.Context, filter koidomain.PromptFilter) (*koidomain.Prompt, error) {
	return r.getBy(ctx, func(db *gorm.DB) *gorm.DB { return applyPromptFilter(db, filter) })
}

func (r *GormPromptRepository) List(ctx context.Context, filter koidomain.PromptFilter) (*koidomain.Page[koidomain.Prompt], error) {
	return r.listBy(ctx, func(db *gorm.DB) *gorm.DB { return applyPromptFilter(db, filter) }, filter.Page, "created_at ASC, id ASC")
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
	return r.updateBy(ctx, func(db *gorm.DB) *gorm.DB { return applyPromptFilter(db, filter) }, values, columns, filter.ID != uuid.Nil)
}

func (r *GormPromptRepository) Delete(ctx context.Context, filter koidomain.PromptFilter) error {
	return r.deleteBy(ctx, func(db *gorm.DB) *gorm.DB { return applyPromptFilter(db, filter) }, filter.ID != uuid.Nil)
}
