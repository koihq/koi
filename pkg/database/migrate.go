package database

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/domain"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// sessionRow is the scs/postgresstore table. Not a domain entity.
type sessionRow struct {
	Token  string    `gorm:"primaryKey;type:text"`
	Data   []byte    `gorm:"not null;type:bytea"`
	Expiry time.Time `gorm:"not null;index"`
}

func (sessionRow) TableName() string { return "sessions" }

// models are the structs the migrations must match. A test applies both the migrations and
// AutoMigrate of these structs and fails when the schemas differ.
func models() []any {
	return []any{
		&domain.User{},
		&domain.PasswordResetToken{},
		&domain.Credential{},
		&domain.OAuthClient{},
		&domain.OAuthCode{},
		&domain.Organization{},
		&domain.BillingSubscription{},
		&domain.BillingCheckoutSession{},
		&domain.StripeWebhookEvent{},
		&domain.BillingChangeRequest{},
		&domain.Membership{},
		&domain.Brand{},
		&domain.Prompt{},
		&domain.Competitor{},
		&domain.CompetitorObservation{},
		&domain.GenerationRun{},
		&domain.VisibilityRun{},
		&domain.VisibilityResult{},
		&sessionRow{},
	}
}

// Migrate applies pending goose migrations. A Postgres advisory lock lets several instances start
// at once; only one applies the migrations.
func Migrate(db *gorm.DB) error {
	ctx := db.Statement.Context
	if ctx == nil {
		ctx = context.Background()
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	migrations, err := fs.Sub(migrationFiles, "migrations")
	if err != nil {
		return err
	}

	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return err
	}

	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		sqlDB,
		migrations,
		goose.WithSessionLocker(locker),
		goose.WithDisableGlobalRegistry(true),
	)
	if err != nil {
		return err
	}

	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}
