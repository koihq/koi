package koidomain

import (
	"context"

	"github.com/google/uuid"
)

const RoleOwner = "owner"

type Membership struct {
	BaseEntity
	UserID         uuid.UUID     `gorm:"type:uuid;uniqueIndex;not null"`
	OrganizationID uuid.UUID     `gorm:"type:uuid;index;not null"`
	User           *User         `json:"-" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Organization   *Organization `json:"-" gorm:"foreignKey:OrganizationID;constraint:OnDelete:CASCADE"`
	Role           string        `gorm:"not null"`
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
	BaseEntityRepository[Membership]
	GetByUserID(ctx context.Context, userID uuid.UUID) (*Membership, error)
}
