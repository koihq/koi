package koidatabase

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/koidomain"
)

type GormPromptRepository struct{ db *gorm.DB }

func NewGormPromptRepository(db *gorm.DB) *GormPromptRepository {
	return &GormPromptRepository{db: db}
}

func promptBrand(brandID uuid.UUID) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB { return db.Where("brand_id = ?", brandID) }
}

func (r *GormPromptRepository) Create(ctx context.Context, prompt *koidomain.Prompt) error {
	return mapDBError(r.db.WithContext(ctx).Create(prompt).Error)
}

func (r *GormPromptRepository) ListByBrandID(ctx context.Context, brandID uuid.UUID) ([]koidomain.Prompt, error) {
	prompts := make([]koidomain.Prompt, 0)
	err := r.db.WithContext(ctx).Scopes(promptBrand(brandID)).Order("created_at ASC, id ASC").Find(&prompts).Error
	return prompts, mapDBError(err)
}

func (r *GormPromptRepository) GetByIDAndBrandID(ctx context.Context, id, brandID uuid.UUID) (*koidomain.Prompt, error) {
	var prompt koidomain.Prompt
	err := r.db.WithContext(ctx).Scopes(promptBrand(brandID)).Where("id = ?", id).Take(&prompt).Error
	if err != nil {
		return nil, mapDBError(err)
	}
	return &prompt, nil
}

func (r *GormPromptRepository) UpdateByIDAndBrandID(ctx context.Context, brandID uuid.UUID, prompt *koidomain.Prompt) error {
	result := r.db.WithContext(ctx).Model(&koidomain.Prompt{}).Scopes(promptBrand(brandID)).
		Where("id = ?", prompt.ID).Update("text", prompt.Text)
	return affected(result)
}

func (r *GormPromptRepository) DeleteByIDAndBrandID(ctx context.Context, id, brandID uuid.UUID) error {
	return affected(r.db.WithContext(ctx).Scopes(promptBrand(brandID)).Where("id = ?", id).Delete(&koidomain.Prompt{}))
}

func (r *GormPromptRepository) ConfirmByIDAndBrandID(ctx context.Context, id, brandID uuid.UUID) error {
	result := r.db.WithContext(ctx).Model(&koidomain.Prompt{}).Scopes(promptBrand(brandID)).
		Where("id = ?", id).Update("confirmed", true)
	return affected(result)
}

func (r *GormPromptRepository) ConfirmAllByBrandID(ctx context.Context, brandID uuid.UUID) error {
	return mapDBError(r.db.WithContext(ctx).Model(&koidomain.Prompt{}).Scopes(promptBrand(brandID)).
		Where("confirmed = ?", false).Update("confirmed", true).Error)
}

func affected(result *gorm.DB) error {
	if err := mapDBError(result.Error); err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return koidomain.ErrNotFound
	}
	return nil
}
