package koidomain

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

const RoleOwner = "owner"

type Membership struct {
	BaseEntity
	UserID         uuid.UUID `gorm:"uniqueIndex:idx_memberships_user_org;not null"`
	OrganizationID uuid.UUID `gorm:"uniqueIndex:idx_memberships_user_org;not null"`
	Role           string    `gorm:"not null"`
}

func (Membership) TableName() string { return "memberships" }

func (m Membership) Validate() error {
	if m.UserID == uuid.Nil || m.OrganizationID == uuid.Nil {
		return fmt.Errorf("%w: membership", ErrValidation)
	}
	if m.Role != RoleOwner {
		return fmt.Errorf("%w: role", ErrValidation)
	}
	return nil
}

type MembershipRepository interface {
	BaseEntityRepository[Membership]
	GetByUserID(ctx context.Context, userID uuid.UUID) (*Membership, error)
}
