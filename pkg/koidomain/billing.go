package koidomain

import (
	"context"
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
	OrganizationID          uuid.UUID  `json:"organization_id" gorm:"type:uuid;not null;uniqueIndex"`
	StripeSubscriptionID    string     `json:"-" gorm:"not null;uniqueIndex"`
	StripePriceID           string     `json:"-" gorm:"not null"`
	Quantity                int        `json:"quantity" gorm:"not null"`
	Status                  string     `json:"status" gorm:"not null"`
	CurrentPeriodEnd        *time.Time `json:"current_period_end"`
	TrialEnd                *time.Time `json:"trial_end"`
	CancelAtPeriodEnd       bool       `json:"cancel_at_period_end" gorm:"not null;default:false"`
	PendingQuantity         *int       `json:"pending_quantity"`
	StripeScheduleID        *string    `json:"-"`
	PaymentURL              string     `json:"payment_url" gorm:"not null;default:''"`
	PendingUpgradeQuantity  *int       `json:"pending_upgrade_quantity"`
	PendingUpgradeExpiresAt *time.Time `json:"pending_upgrade_expires_at"`
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
	Retired              bool    `gorm:"not null;default:false"`
	StripeSubscriptionID *string `gorm:"uniqueIndex"`
}

func (BillingCheckoutSession) TableName() string { return "billing_checkout_sessions" }

type StripeWebhookEvent struct {
	BaseEntity
	StripeEventID string `gorm:"not null;uniqueIndex"`
}

func (StripeWebhookEvent) TableName() string { return "stripe_webhook_events" }

type BillingChange struct {
	Quantity   int        `json:"quantity"`
	Effective  string     `json:"effective"`
	PaymentURL string     `json:"payment_url,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

type BillingChangeRequest struct {
	BaseEntity
	OrganizationID uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex:idx_billing_change_key,priority:1"`
	IdempotencyKey string         `gorm:"type:varchar(255);not null;uniqueIndex:idx_billing_change_key,priority:2"`
	SubscriptionID string         `gorm:"not null"`
	Quantity       int            `gorm:"not null"`
	Result         *BillingChange `gorm:"type:jsonb;serializer:json"`
	ErrorCode      string         `gorm:"not null;default:''"`
}

func (BillingChangeRequest) TableName() string { return "billing_change_requests" }

// BillingSnapshot is the validated remote state of Koi's single subscription item.
type BillingSnapshot struct {
	ID                      string
	CustomerID              string
	OrganizationID          string
	ItemID                  string
	PriceID                 string
	Quantity                int
	Status                  string
	CurrentPeriodEnd        *time.Time
	TrialEnd                *time.Time
	CancelAtPeriodEnd       bool
	ScheduleID              *string
	PendingQuantity         *int
	PendingUpgradeQuantity  *int
	PendingUpgradeExpiresAt *time.Time
	PaymentURL              string
}

type BillingCheckout struct {
	ID             string
	URL            string
	CustomerID     string
	OrganizationID string
	SubscriptionID string
	Status         string
	ExpiresAt      time.Time
}

type BillingCheckoutInput struct {
	CustomerID     string
	OrganizationID string
	PriceID        string
	ReservationID  string
	Quantity       int
	PublicURL      string
	Trial          bool
}

type BillingEvent struct {
	ID             string
	Type           string
	CustomerID     string
	OrganizationID string
	CheckoutID     string
	SubscriptionID string
}

type BillingStatus struct {
	Subscription   *BillingSubscription `json:"subscription"`
	PromptLimit    int                  `json:"prompt_limit"`
	TrialAvailable bool                 `json:"trial_available"`
}

type BillingQuote struct {
	Prompts     int    `json:"prompts"`
	AmountCents int64  `json:"amount_cents"`
	Currency    string `json:"currency"`
	Interval    string `json:"interval"`
}

type BillingCatalog struct {
	Currency string         `json:"currency"`
	Interval string         `json:"interval"`
	Options  []BillingQuote `json:"options"`
}

type BillingGateway interface {
	ValidatePrice(context.Context, string) error
	CreateCustomer(context.Context, string) (string, error)
	CreateCheckout(context.Context, BillingCheckoutInput) (BillingCheckout, error)
	ExpireCheckout(context.Context, string) (BillingCheckout, error)
	CreatePortal(context.Context, string, string) (string, error)
	GetSubscription(context.Context, string) (BillingSnapshot, error)
	Upgrade(context.Context, BillingSnapshot, int, string) (BillingSnapshot, error)
	ChangeTrial(context.Context, BillingSnapshot, int, string) (BillingSnapshot, error)
	ScheduleDowngrade(context.Context, BillingSnapshot, int, string) (BillingSnapshot, error)
	ReleaseSchedule(context.Context, BillingSnapshot, string) (BillingSnapshot, error)
}

type BillingRepository interface {
	OrganizationForUser(context.Context) (*Organization, error)
	OrganizationByCustomer(context.Context, string) (*Organization, error)
	LockOrganization(context.Context, uuid.UUID) (*Organization, error)
	ProductAccess(context.Context, uuid.UUID) (bool, error)
	SaveCustomer(context.Context, uuid.UUID, string) (string, error)
	Subscription(context.Context, uuid.UUID) (*BillingSubscription, error)
	ReserveCheckout(context.Context, uuid.UUID, string, string, int) (*BillingCheckoutSession, error)
	Checkout(context.Context, uuid.UUID) (*BillingCheckoutSession, error)
	CheckoutBySession(context.Context, string) (*BillingCheckoutSession, error)
	CompletedCheckout(context.Context, uuid.UUID, string) (*BillingCheckoutSession, error)
	FinalizeCheckout(context.Context, uuid.UUID, string, string, time.Time) error
	RetireCheckout(context.Context, uuid.UUID) error
	CompleteCheckout(context.Context, *BillingCheckoutSession, string) error
	SaveSubscription(context.Context, *BillingSubscription) error
	RecordEvent(context.Context, string) (bool, error)
	MarkTrialUsed(context.Context, uuid.UUID) error
	ReserveChange(context.Context, uuid.UUID, string, int) (*BillingChangeRequest, error)
	PendingChanges(context.Context, uuid.UUID) ([]BillingChangeRequest, error)
	SaveChange(context.Context, *BillingChangeRequest) error
}

type BillingCommander interface {
	Checkout(context.Context, int) (string, error)
	ChangeAllowance(context.Context, int, string) (BillingChange, error)
	Portal(context.Context) (string, error)
	ProcessEvent(context.Context, BillingEvent) error
}

type BillingQuerier interface {
	Catalog(context.Context) (BillingCatalog, error)
	Quote(context.Context, int) (BillingQuote, error)
	Subscription(context.Context) (BillingStatus, error)
	ProductAccess(context.Context, uuid.UUID) (bool, error)
}

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
