package koidatabase

import (
	"time"

	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/koidomain"
)

// sessionRow is the scs/postgresstore table. Not a domain entity.
type sessionRow struct {
	Token  string    `gorm:"primaryKey;type:text"`
	Data   []byte    `gorm:"not null;type:bytea"`
	Expiry time.Time `gorm:"not null;index"`
}

func (sessionRow) TableName() string { return "sessions" }

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&koidomain.User{},
		&koidomain.Organization{},
		&koidomain.Membership{},
		&sessionRow{},
	)
}
