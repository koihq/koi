package koidatabase

import (
	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/koidomain"
)

type GormOrganizationRepository struct {
	*Repository[koidomain.Organization]
}

func NewGormOrganizationRepository(db *gorm.DB) *GormOrganizationRepository {
	return &GormOrganizationRepository{Repository: NewRepository[koidomain.Organization](db)}
}
