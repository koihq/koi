package database

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/koihq/koi/pkg/auth"
	"github.com/koihq/koi/pkg/domain"
)

type GormBrandRepository struct {
	*ListRepository[domain.Brand, domain.BrandListFilter]
}

func NewBrandRepository(db *gorm.DB) *GormBrandRepository {
	base := newScopedRepository(db, tenantScope[domain.Brand]{
		read: func(identity auth.Identity, db *gorm.DB) *gorm.DB {
			return db.Where("brands.organization_id = ?", identity.OrganizationID)
		},
		authorizeCreate: func(_ *gorm.DB, identity auth.Identity, brand *domain.Brand) error {
			if identity.OrganizationID != brand.OrganizationID {
				return domain.ErrUnauthorized
			}

			return nil
		},
	}, "name", "domain", "github_repo", "country", "description")

	return &GormBrandRepository{
		ListRepository: newListRepository(base, filterBrands, fixedOrder[domain.BrandListFilter]("created_at DESC, id DESC")),
	}
}

func filterBrands(db *gorm.DB, filter domain.BrandListFilter) *gorm.DB {
	if filter.Query == "" {
		return db
	}

	pattern := likePattern(filter.Query)

	return db.Where("(name ILIKE ? ESCAPE '!' OR domain ILIKE ? ESCAPE '!' OR github_repo ILIKE ? ESCAPE '!')", pattern, pattern, pattern)
}

func (r *GormBrandRepository) LockForUpdate(ctx context.Context, id uuid.UUID) (*domain.Brand, error) {
	return r.getBy(ctx, func(db *gorm.DB) *gorm.DB {
		return r.whereID(db.Clauses(clause.Locking{Strength: "UPDATE"}), id)
	})
}

func (r *GormBrandRepository) Latest(ctx context.Context) (*domain.Brand, error) {
	return r.getBy(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Order("created_at DESC, id DESC")
	})
}

var _ domain.BrandRepository = (*GormBrandRepository)(nil)
