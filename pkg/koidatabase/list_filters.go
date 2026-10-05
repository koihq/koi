package koidatabase

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func applyChildListFilter(db *gorm.DB, id, brandID uuid.UUID, confirmed *bool) *gorm.DB {
	if id != uuid.Nil {
		db = db.Where("id = ?", id)
	}
	if brandID != uuid.Nil {
		db = db.Where("brand_id = ?", brandID)
	}
	if confirmed != nil {
		db = db.Where("confirmed = ?", *confirmed)
	}
	return db
}
