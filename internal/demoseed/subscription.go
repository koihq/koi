package demoseed

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/koihq/koi/pkg/domain"
)

// DemoQuantity is a catalog option above the prompts the scenarios store, with room to add more.
const DemoQuantity = 50

// EnsureCloudSubscription gives the demo organization an active plan, so a cloud-mode staging
// instance shows the seeded brands instead of the paywall. Stripe does not know this
// subscription: billing actions on the demo account fail. Each run moves the period end 30 days
// past now. Run must have created the account first.
func EnsureCloudSubscription(ctx context.Context, db *gorm.DB, now time.Time) error {
	now = now.UTC()
	periodEnd := now.AddDate(0, 0, 30)

	subscription := domain.BillingSubscription{
		BaseEntity:           entity("subscription", now),
		OrganizationID:       demoID("organization"),
		StripeSubscriptionID: "sub_demo_koi",
		StripePriceID:        "price_demo_koi",
		Quantity:             DemoQuantity,
		Status:               "active",
		CurrentPeriodEnd:     &periodEnd,
	}

	return db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "organization_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"status":               "active",
			"quantity":             DemoQuantity,
			"current_period_end":   periodEnd,
			"cancel_at_period_end": false,
			"updated_at":           now,
		}),
	}).Create(&subscription).Error
}
