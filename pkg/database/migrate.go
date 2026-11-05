package database

import (
	"time"

	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/domain"
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
		&domain.User{},
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
	)
}
