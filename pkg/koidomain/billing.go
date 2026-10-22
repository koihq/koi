package koidomain

import (
	"time"

	"github.com/google/uuid"
)

var (
	ErrBillingSubscriptionExists = &Error{Code: CodeBillingSubscriptionExists, Message: "organization already has a subscription"}
	ErrBillingCheckoutPending    = &Error{Code: CodeBillingCheckoutPending, Message: "another checkout is pending"}
	ErrBillingPriceMismatch      = &Error{Code: CodeBillingPriceMismatch, Message: "subscription uses a different Stripe price"}
)

type BillingSubscription struct {
	BaseEntity
	OrganizationID       uuid.UUID  `json:"organization_id" gorm:"type:uuid;not null;uniqueIndex"`
	StripeSubscriptionID string     `json:"-" gorm:"not null;uniqueIndex"`
	StripePriceID        string     `json:"-" gorm:"not null"`
	Quantity             int        `json:"quantity" gorm:"not null"`
	Status               string     `json:"status" gorm:"not null"`
	CurrentPeriodEnd     *time.Time `json:"current_period_end"`
	TrialEnd             *time.Time `json:"trial_end"`
	CancelAtPeriodEnd    bool       `json:"cancel_at_period_end" gorm:"not null;default:false"`
	PendingQuantity      *int       `json:"pending_quantity"`
	StripeScheduleID     *string    `json:"-"`
}

func (BillingSubscription) TableName() string { return "billing_subscriptions" }

type BillingCheckoutSession struct {
	BaseEntity
	OrganizationID       uuid.UUID `gorm:"type:uuid;not null;index"`
	SessionID            string    `gorm:"not null;uniqueIndex"`
	URL                  string    `gorm:"not null;default:''"`
	ExpiresAt            *time.Time
	CustomerID           string  `gorm:"not null"`
	PriceID              string  `gorm:"not null"`
	Quantity             int     `gorm:"not null"`
	Trial                bool    `gorm:"not null;default:false"`
	Completed            bool    `gorm:"not null;default:false"`
	StripeSubscriptionID *string `gorm:"uniqueIndex"`
}

func (BillingCheckoutSession) TableName() string { return "billing_checkout_sessions" }

type StripeWebhookEvent struct {
	BaseEntity
	StripeEventID string `gorm:"not null;uniqueIndex"`
}

func (StripeWebhookEvent) TableName() string { return "stripe_webhook_events" }

func (s BillingSubscription) PromptLimit(now time.Time) int {
	switch s.Status {
	case "trialing":
		if s.TrialEnd != nil && now.Before(*s.TrialEnd) {
			return 5
		}
	case "active":
		if s.CurrentPeriodEnd != nil && now.Before(*s.CurrentPeriodEnd) {
			return s.Quantity
		}
	}

	return 0
}
