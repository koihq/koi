package koidatabase

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/koidomain"
)

type GormBrandRepository struct {
	*Repository[koidomain.Brand]
}

func NewGormBrandRepository(db *gorm.DB) *GormBrandRepository {
	return &GormBrandRepository{Repository: NewRepository[koidomain.Brand](db)}
}

func (r *GormBrandRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]koidomain.Brand, error) {
	var brands []koidomain.Brand
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&brands).Error
	return brands, mapDBError(err)
}
