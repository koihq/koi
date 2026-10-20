package koidatabase

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/koihq/koi/pkg/koiauth"
	"github.com/koihq/koi/pkg/koidomain"
)

type GormBrandRepository struct {
	*Repository[koidomain.Brand]
}

func NewGormBrandRepository(db *gorm.DB) *GormBrandRepository {
	return &GormBrandRepository{Repository: newScopedRepository[koidomain.Brand](db, brandOwnerScope)}
}

func (r *GormBrandRepository) Create(ctx context.Context, brand *koidomain.Brand) error {
	identity, ok := koiauth.FromContext(ctx)
	if !ok {
		return koidomain.ErrUnauthorized
	}

	var membership koidomain.Membership
	err := r.db.WithContext(ctx).Where("user_id = ? AND organization_id = ?", identity.UserID, brand.OrganizationID).
		Take(&membership).Error
	if err == gorm.ErrRecordNotFound {
		return koidomain.ErrUnauthorized
	}
	if err != nil {
		return err
	}

	return r.Repository.Create(ctx, brand)
}

func (r *GormBrandRepository) LockForUpdate(ctx context.Context, id uuid.UUID) (*koidomain.Brand, error) {
	return r.getBy(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id)
	})
}

func (r *GormBrandRepository) List(ctx context.Context, params koidomain.PageParams) (*koidomain.Page[koidomain.Brand], error) {
	return r.listBy(ctx, func(db *gorm.DB) *gorm.DB { return db }, params, "created_at DESC, id DESC")
}

func (r *GormBrandRepository) Update(ctx context.Context, brand *koidomain.Brand) error {
	return r.updateBy(ctx, func(db *gorm.DB) *gorm.DB { return db.Where("id = ?", brand.ID) },
		map[string]any{
			"name":        brand.Name,
			"domain":      brand.Domain,
			"github_repo": brand.GitHubRepo,
			"country":     brand.Country,
			"description": brand.Description,
		}, []string{"name", "domain", "github_repo", "country", "description"}, true)
}

var _ koidomain.BrandRepository = (*GormBrandRepository)(nil)
