package database

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/domain"
)

type GormMembershipRepository struct {
	*Repository[domain.Membership]
}

func NewMembershipRepository(db *gorm.DB) *GormMembershipRepository {
	return &GormMembershipRepository{Repository: newRepository[domain.Membership](db)}
}

func (r *GormMembershipRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Membership, error) {
	entity := new(domain.Membership)
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Take(entity).Error
	if err != nil {
		return nil, mapDBError(err)
	}

	return entity, nil
}
