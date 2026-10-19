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
		if err := NewGormPromptRepository(tx).Repository.Create(ctx, prompt); err != nil {
			return err
		}
		return invalidateReview(tx, prompt.BrandID, "prompts")
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
	order := promptSortOrder(filter.Sort, filter.Descending, "prompts.text", "prompts.created_at ASC, prompts.id ASC")

	return r.listBy(ctx, func(db *gorm.DB) *gorm.DB {
		db = applyPromptFilter(db, filter)
		if filter.Sort == koidomain.PromptSortRate || filter.Sort == koidomain.PromptSortChange {
			join, args := visibilitySortJoin("prompts.id", filter.BrandID, filter.Days, filter.Engines)
			db = db.Joins(join, args...)
		}

		return db
	}, filter.Page, order)
}

func (r *GormPromptRepository) Update(ctx context.Context, filter koidomain.PromptFilter, update koidomain.PromptUpdate) error {
	if _, err := r.Repository.query(ctx); err != nil {
		return err
	}

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
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := NewGormBrandRepository(tx).LockForUpdate(ctx, filter.BrandID); err != nil {
			return err
		}
		if err := NewGormPromptRepository(tx).updateBy(ctx, func(db *gorm.DB) *gorm.DB { return applyPromptFilter(db, filter) }, values, columns, filter.ID != uuid.Nil); err != nil {
			return err
		}
		if update.Text != nil || update.Confirmed != nil && !*update.Confirmed {
			return invalidateReview(tx, filter.BrandID, "prompts")
		}
		return nil
	})
}

func (r *GormPromptRepository) Delete(ctx context.Context, filter koidomain.PromptFilter) error {
	if _, err := r.Repository.query(ctx); err != nil {
		return err
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := NewGormBrandRepository(tx).LockForUpdate(ctx, filter.BrandID); err != nil {
			return err
		}
		if err := NewGormPromptRepository(tx).deleteBy(ctx, func(db *gorm.DB) *gorm.DB { return applyPromptFilter(db, filter) }, filter.ID != uuid.Nil); err != nil {
			return err
		}
		return invalidateReview(tx, filter.BrandID, "prompts")
	})
}
