package database

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/domain"
)

type GormPromptRepository struct {
	*ListRepository[domain.Prompt, domain.PromptFilter]
}

func NewPromptRepository(db *gorm.DB) *GormPromptRepository {
	base := newBrandChildRepository[domain.Prompt](db, "text", "confirmed", "inactive")

	return &GormPromptRepository{
		ListRepository: newListRepository(base, applyPromptFilter, promptOrder),
	}
}

func applyPromptFilter(db *gorm.DB, filter domain.PromptFilter) *gorm.DB {
	db = filterByBrand(db, filter.BrandFilter)

	if filter.Confirmed != nil {
		db = db.Where("confirmed = ?", *filter.Confirmed)
	}

	if filter.Inactive != nil {
		db = db.Where("inactive = ?", *filter.Inactive)
	}

	if filter.Query != "" {
		db = db.Where("text ILIKE ? ESCAPE '!'", likePattern(filter.Query))
	}

	if filter.Sort == domain.PromptSortRate || filter.Sort == domain.PromptSortChange {
		join, args := visibilitySortJoin("prompts.id", filter.BrandID, filter.Days, filter.Engines)
		db = db.Joins(join, args...)
	}

	return db
}

func promptOrder(filter domain.PromptFilter) string {
	return promptSortOrder(filter.Sort, filter.Descending, "prompts.text", "prompts.created_at ASC, prompts.id ASC")
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

	subquery := query.Model(&domain.Prompt{}).
		Select("1").
		Where("brand_id = ? AND confirmed = true", brandID)

	var exists bool
	err = r.db.WithContext(ctx).Raw("SELECT EXISTS (?)", subquery).Scan(&exists).Error

	return exists, mapDBError(err)
}

var _ domain.PromptRepository = (*GormPromptRepository)(nil)
