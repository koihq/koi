package koidatabase

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

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
		return nil, mapDBError(err)
	}

	return entity, nil
}

func (r *GormUserRepository) LockForUpdate(ctx context.Context, id uuid.UUID) error {
	var user koidomain.User
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").Where("id = ?", id).Take(&user).Error

	return mapDBError(err)
}
