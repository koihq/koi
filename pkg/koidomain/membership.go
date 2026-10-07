package koidomain

import (
	"context"

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
		return Validation(FieldError{Field: "membership", Code: FieldInvalid})
	}

	if m.Role != RoleOwner {
		return Validation(FieldError{Field: "role", Code: FieldInvalid})
	}

	return nil
}

type MembershipRepository interface {
	BaseEntityRepository[Membership, uuid.UUID]
	GetByUserID(ctx context.Context, userID uuid.UUID) (*Membership, error)
}
