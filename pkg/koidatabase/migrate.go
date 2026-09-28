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
	// AutoMigrate adds columns but does not rename them; preserve existing brand descriptions.
	if db.Migrator().HasTable(&koidomain.Brand{}) &&
		db.Migrator().HasColumn(&koidomain.Brand{}, "product_line") &&
		!db.Migrator().HasColumn(&koidomain.Brand{}, "description") {
		if err := db.Migrator().RenameColumn(&koidomain.Brand{}, "product_line", "description"); err != nil {
			return err
		}
	}
	if err := db.AutoMigrate(
		&koidomain.User{},
		&koidomain.Organization{},
		&koidomain.Membership{},
		&koidomain.Brand{},
		&koidomain.Prompt{},
		&koidomain.Competitor{},
		&sessionRow{},
	); err != nil {
		return err
	}
	if db.Migrator().HasColumn("competitors", "aliases") {
		return db.Migrator().DropColumn("competitors", "aliases")
	}
	return nil
}
