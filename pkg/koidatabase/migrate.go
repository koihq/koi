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
		&koidomain.GenerationRun{},
		&sessionRow{},
	); err != nil {
		return err
	}
	for _, statement := range []string{
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_prompts_brand_text ON prompts (brand_id, lower(text))",
		"CREATE INDEX IF NOT EXISTS idx_generation_runs_recent ON generation_runs (brand_id, created_at DESC, id DESC)",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_generation_runs_key ON generation_runs (brand_id, idempotency_key) WHERE idempotency_key IS NOT NULL",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_generation_runs_active ON generation_runs (brand_id) WHERE state IN ('queued', 'running')",
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	if db.Migrator().HasColumn("competitors", "aliases") {
		return db.Migrator().DropColumn("competitors", "aliases")
	}
	return nil
}
