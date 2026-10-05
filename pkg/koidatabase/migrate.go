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
		&koidomain.VisibilityRun{},
		&koidomain.VisibilityResult{},
		&sessionRow{},
	); err != nil {
		return err
	}
	if err := db.Exec("DROP INDEX IF EXISTS idx_competitors_brand_domain").Error; err != nil {
		return err
	}
	for _, statement := range []string{
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_competitors_brand_domain ON competitors (brand_id, domain) WHERE domain <> ''",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_competitors_brand_github_repo ON competitors (brand_id, github_repo) WHERE github_repo <> ''",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_prompts_brand_text ON prompts (brand_id, lower(text))",
		"CREATE INDEX IF NOT EXISTS idx_generation_runs_recent ON generation_runs (brand_id, created_at DESC, id DESC)",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_generation_runs_key ON generation_runs (brand_id, idempotency_key) WHERE idempotency_key IS NOT NULL",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_generation_runs_active ON generation_runs (brand_id) WHERE state IN ('queued', 'running')",
		"CREATE INDEX IF NOT EXISTS idx_visibility_runs_recent ON visibility_runs (brand_id, created_at DESC, id DESC)",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_visibility_runs_active ON visibility_runs (brand_id) WHERE state IN ('queued', 'running')",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_visibility_results_cell ON visibility_results (run_id, prompt_id, engine)",
		"CREATE INDEX IF NOT EXISTS idx_visibility_results_order ON visibility_results (run_id, prompt_position, engine)",
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
