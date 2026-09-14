package koidomain

import (
	"fmt"
	"strings"
)

type Organization struct {
	BaseEntity
	Name string `json:"name" gorm:"not null"`
}

func (Organization) TableName() string { return "organizations" }

func (o Organization) Validate() error {
	if strings.TrimSpace(o.Name) == "" {
		return fmt.Errorf("%w: name", ErrValidation)
	}
	return nil
}

type OrganizationRepository interface {
	BaseEntityRepository[Organization]
}
