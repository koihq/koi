package koidatabase

import (
	"context"
	"github.com/google/uuid"
	"strings"

	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/koidomain"
)

type GormPromptRepository struct {
	*Repository[koidomain.Prompt]
}

func NewGormPromptRepository(db *gorm.DB) *GormPromptRepository {
	return &GormPromptRepository{
		Repository: newScopedRepository[koidomain.Prompt](db, childOwnerScope("prompts")),
	}
}

func (r *GormPromptRepository) Create(ctx context.Context, prompt *koidomain.Prompt) error {
	if _, err := r.Repository.query(ctx); err != nil {
		return err
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := NewGormBrandRepository(tx).LockForUpdate(ctx, prompt.BrandID); err != nil {
			return err
		}

		return NewGormPromptRepository(tx).Repository.Create(ctx, prompt)
	})
}

func applyPromptFilter(db *gorm.DB, filter koidomain.PromptFilter) *gorm.DB {
	db = applyChildListFilter(db, filter.ID, filter.BrandID, filter.Confirmed)
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
