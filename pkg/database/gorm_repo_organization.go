package database

import (
	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/domain"
)

type GormOrganizationRepository struct {
	*Repository[domain.Organization]
}

func NewOrganizationRepository(db *gorm.DB) *GormOrganizationRepository {
	return &GormOrganizationRepository{Repository: newRepository[domain.Organization](db)}
}
