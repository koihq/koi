package koidatabase

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/koidomain"
)

type GormMembershipRepository struct {
	*Repository[koidomain.Membership]
}

func NewGormMembershipRepository(db *gorm.DB) *GormMembershipRepository {
	return &GormMembershipRepository{Repository: NewRepository[koidomain.Membership](db)}
}

func (r *GormMembershipRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*koidomain.Membership, error) {
	entity := new(koidomain.Membership)
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Take(entity).Error
	if err != nil {
		return nil, mapNotFound(err)
	}
	return entity, nil
}
