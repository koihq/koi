package koidatabase

import (
	"context"

	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/koidomain"
)

type GormUserRepository struct {
	*Repository[koidomain.User]
}

func NewGormUserRepository(db *gorm.DB) *GormUserRepository {
	return &GormUserRepository{Repository: NewRepository[koidomain.User](db)}
}

func (r *GormUserRepository) GetByEmail(ctx context.Context, email string) (*koidomain.User, error) {
	entity := new(koidomain.User)
	err := r.db.WithContext(ctx).Where("email = ?", email).Take(entity).Error
	if err != nil {
		return nil, mapNotFound(err)
	}
	return entity, nil
}
