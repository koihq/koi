package koidomain

import (
	"strings"

	"github.com/google/uuid"
)

type Organization struct {
	BaseEntity
	Name string `json:"name" gorm:"not null"`
}

func (Organization) TableName() string { return "organizations" }

func (o Organization) Validate() error {
	if strings.TrimSpace(o.Name) == "" {
		return Validation(FieldError{Field: "name", Code: FieldInvalid})
	}

	return nil
}

type OrganizationRepository interface {
	BaseEntityRepository[Organization, uuid.UUID]
}
