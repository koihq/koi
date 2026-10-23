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
	cloudMode bool
}

func NewGormPromptRepository(db *gorm.DB, cloud ...bool) *GormPromptRepository {
	return &GormPromptRepository{
		Repository: newScopedRepository[koidomain.Prompt](db, childOwnerScope("prompts")),
		cloudMode:  len(cloud) > 0 && cloud[0],
	}
}

func (r *GormPromptRepository) Create(ctx context.Context, prompt *koidomain.Prompt) error {
	if _, err := r.Repository.query(ctx); err != nil {
		return err
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		brand, err := NewGormBrandRepository(tx).LockForUpdate(ctx, prompt.BrandID)
		if err != nil {
			return err
		}
		if r.cloudMode && !prompt.Inactive {
			if err := lockOrganization(ctx, tx, brand.OrganizationID); err != nil {
				return err
			}
			if err := requirePromptCapacity(ctx, tx, brand.OrganizationID, 1); err != nil {
				return err
			}
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

func (r *GormPromptRepository) HasConfirmed(ctx context.Context, brandID uuid.UUID) (bool, error) {
	query, err := r.query(ctx)
	if err != nil {
		return false, err
	}

	subquery := query.Model(&koidomain.Prompt{}).
		Select("1").
		Where("brand_id = ? AND confirmed = true", brandID)
	var exists bool
	err = r.db.WithContext(ctx).Raw("SELECT EXISTS (?)", subquery).Scan(&exists).Error

	return exists, mapDBError(err)
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

func (r *GormPromptRepository) Update(ctx context.Context, id uuid.UUID, update koidomain.PromptUpdate) error {
	currentEntity, err := r.Get(ctx, id)
	if err != nil {
		return err
	}
	filter := koidomain.PromptFilter{ID: id, BrandID: currentEntity.BrandID}

	values := &koidomain.Prompt{}
	columns := make([]string, 0, 3)
	if update.Text != nil {
		values.Text = *update.Text
		columns = append(columns, "text")
	}

	if update.Confirmed != nil {
		values.Confirmed = *update.Confirmed
		columns = append(columns, "confirmed")
	}
	if update.Inactive != nil {
		values.Inactive = *update.Inactive
		columns = append(columns, "inactive")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		brand, err := NewGormBrandRepository(tx).LockForUpdate(ctx, filter.BrandID)
		if err != nil {
			return err
		}
		if r.cloudMode && filter.ID != uuid.Nil && (update.Confirmed != nil || update.Inactive != nil) {
			if err := lockOrganization(ctx, tx, brand.OrganizationID); err != nil {
				return err
			}
			current, err := NewGormPromptRepository(tx).Get(ctx, filter.ID)
			if err != nil {
				return err
			}
			wasInactive := current.Inactive
			if update.Inactive != nil {
				current.Inactive = *update.Inactive
			}
			if wasInactive && !current.Inactive {
				if err := requirePromptCapacity(ctx, tx, brand.OrganizationID, 1); err != nil {
					return err
				}
			}
		}
		if err := NewGormPromptRepository(tx).updateBy(ctx, func(db *gorm.DB) *gorm.DB { return applyPromptFilter(db, filter) }, values, columns, filter.ID != uuid.Nil); err != nil {
			return err
		}
		if update.Text != nil || update.Confirmed != nil && !*update.Confirmed || update.Inactive != nil {
			return invalidateReview(tx, filter.BrandID, "prompts")
		}
		return nil
	})
}

func (r *GormPromptRepository) Delete(ctx context.Context, id uuid.UUID) error {
	currentEntity, err := r.Get(ctx, id)
	if err != nil {
		return err
	}
	filter := koidomain.PromptFilter{ID: id, BrandID: currentEntity.BrandID}

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
