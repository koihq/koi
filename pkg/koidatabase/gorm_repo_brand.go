package koidatabase

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/koidomain"
)

type GormBrandRepository struct {
	db *gorm.DB
}

func NewGormBrandRepository(db *gorm.DB) *GormBrandRepository {
	return &GormBrandRepository{db: db}
}

func brandOwner(userID uuid.UUID) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB { return db.Where("user_id = ?", userID) }
}

func (r *GormBrandRepository) Create(ctx context.Context, brand *koidomain.Brand) error {
	return mapDBError(r.db.WithContext(ctx).Create(brand).Error)
}

func (r *GormBrandRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&koidomain.Brand{}).Scopes(brandOwner(userID)).Count(&count).Error
	return count, mapDBError(err)
}

func (r *GormBrandRepository) ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]koidomain.Brand, error) {
	brands := make([]koidomain.Brand, 0)
	err := r.db.WithContext(ctx).Scopes(brandOwner(userID)).Order("created_at DESC, id DESC").Limit(limit).Offset(offset).Find(&brands).Error
	return brands, mapDBError(err)
}

func (r *GormBrandRepository) GetByIDAndUserID(ctx context.Context, id, userID uuid.UUID) (*koidomain.Brand, error) {
	var brand koidomain.Brand
	err := r.db.WithContext(ctx).Scopes(brandOwner(userID)).Where("id = ?", id).Take(&brand).Error
	if err != nil {
		return nil, mapDBError(err)
	}
	return &brand, nil
}

func (r *GormBrandRepository) UpdateByIDAndUserID(ctx context.Context, actorID uuid.UUID, brand *koidomain.Brand) error {
	result := r.db.WithContext(ctx).Model(&koidomain.Brand{}).Scopes(brandOwner(actorID)).
		Where("id = ?", brand.ID).
		Updates(map[string]any{
			"name": brand.Name, "domain": brand.Domain, "github_repo": brand.GitHubRepo,
			"country": brand.Country, "description": brand.Description,
		})
	if err := mapDBError(result.Error); err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return koidomain.ErrNotFound
	}
	return nil
}
