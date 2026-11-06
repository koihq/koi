package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const (
	TrialPromptLimit        = 10
	InitialDraftPromptLimit = 5
	FirstLookPromptLimit    = InitialDraftPromptLimit

	// DefaultFirstLookDailyLimit bounds what unpaid First Looks can spend on collection in a day.
	DefaultFirstLookDailyLimit = 100
)

// Reasons a First Look organization cannot collect, reported through tracking status.
const (
	TrackingBlockedFirstLookUsed     = "first_look_used"
	TrackingBlockedFirstLookSiteUsed = "first_look_site_used"
	TrackingBlockedFirstLookPaused   = "first_look_paused"
)

var (
	ErrBillingSubscriptionExists = &Error{Code: CodeBillingSubscriptionExists, Message: "organization already has a subscription"}
	ErrBillingCheckoutPending    = &Error{Code: CodeBillingCheckoutPending, Message: "another checkout is pending"}
	ErrBillingPriceMismatch      = &Error{Code: CodeBillingPriceMismatch, Message: "subscription uses a different Stripe price"}
)

type BillingSubscription struct {
	BaseEntity
	OrganizationID          uuid.UUID     `json:"organization_id" gorm:"type:uuid;not null;uniqueIndex"`
	Organization            *Organization `json:"-" gorm:"foreignKey:OrganizationID;constraint:OnDelete:RESTRICT"`
	StripeSubscriptionID    string        `json:"-" gorm:"not null;uniqueIndex"`
	StripePriceID           string        `json:"-" gorm:"not null"`
	Quantity                int           `json:"quantity" gorm:"not null"`
	Status                  string        `json:"status" gorm:"not null"`
	CurrentPeriodEnd        *time.Time    `json:"current_period_end"`
	TrialEnd                *time.Time    `json:"trial_end"`
	CancelAtPeriodEnd       bool          `json:"cancel_at_period_end" gorm:"not null;default:false"`
	PendingQuantity         *int          `json:"pending_quantity"`
	StripeScheduleID        *string       `json:"-"`
	PaymentURL              string        `json:"payment_url" gorm:"not null;default:''"`
	PendingUpgradeQuantity  *int          `json:"pending_upgrade_quantity"`
	PendingUpgradeExpiresAt *time.Time    `json:"pending_upgrade_expires_at"`
}

func (BillingSubscription) TableName() string { return "billing_subscriptions" }

type BillingCheckoutSession struct {
	BaseEntity
	OrganizationID       uuid.UUID     `gorm:"type:uuid;not null;index"`
	Organization         *Organization `json:"-" gorm:"foreignKey:OrganizationID;constraint:OnDelete:RESTRICT"`
	SessionID            string        `gorm:"not null;uniqueIndex"`
	URL                  string        `gorm:"not null;default:''"`
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
	Organization   *Organization  `json:"-" gorm:"foreignKey:OrganizationID;constraint:OnDelete:RESTRICT"`
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
	UsedPrompts    int64                `json:"used_prompts"`
	TrialAvailable bool                 `json:"trial_available"`
	FirstLook      bool                 `json:"first_look"`
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
	RetrieveCheckout(context.Context, string) (BillingCheckout, error)
	ExpireCheckout(context.Context, string) (BillingCheckout, error)
	CreatePortal(context.Context, string, string) (string, error)
	GetSubscription(context.Context, string) (BillingSnapshot, error)
	Upgrade(context.Context, BillingSnapshot, int, string) (BillingSnapshot, error)
	ChangeTrial(context.Context, BillingSnapshot, int, string) (BillingSnapshot, error)
	EndTrial(context.Context, BillingSnapshot, string) (BillingSnapshot, error)
	ScheduleDowngrade(context.Context, BillingSnapshot, int, string) (BillingSnapshot, error)
	ReleaseSchedule(context.Context, BillingSnapshot, string) (BillingSnapshot, error)
}

type BillingRepository interface {
	OrganizationForUser(context.Context) (*Organization, error)
	OrganizationByCustomer(context.Context, string) (*Organization, error)
	LockOrganization(context.Context, uuid.UUID) (*Organization, error)
	// ReservePromptCapacity holds the organization until the transaction ends and fails if
	// activating more prompts would exceed its allowance. Only Koi Cloud has an allowance.
	ReservePromptCapacity(ctx context.Context, organizationID uuid.UUID, additional int64) error
	ProductAccess(context.Context, uuid.UUID) (bool, error)
	// UsedPromptCount is what ReservePromptCapacity measures: every prompt that is not stopped,
	// drafts included, across all brands.
	UsedPromptCount(context.Context, uuid.UUID) (int64, error)
	SaveCustomer(context.Context, uuid.UUID, string) (string, error)
	Subscription(context.Context, uuid.UUID) (*BillingSubscription, error)
	ReserveCheckout(context.Context, uuid.UUID, string, string, int) (*BillingCheckoutSession, error)
	Checkout(context.Context, uuid.UUID) (*BillingCheckoutSession, error)
	CheckoutBySession(context.Context, string) (*BillingCheckoutSession, error)
	PendingCheckouts(context.Context, time.Time, int) ([]BillingCheckoutSession, error)
	TouchCheckout(context.Context, uuid.UUID) error
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
	ConfirmCheckout(context.Context, string) (BillingStatus, error)
	ReconcilePendingCheckouts(context.Context, time.Time) error
	ChangeAllowance(context.Context, int, string) (BillingChange, error)
	StartPlan(context.Context, string) (BillingChange, error)
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
	if s.inTrial(now) {
		return TrialPromptLimit
	}

	if s.Status == "active" && s.CurrentPeriodEnd != nil && now.Before(*s.CurrentPeriodEnd) {
		return s.Quantity
	}

	return 0
}

// IsFirstLook reports an organization that never had a subscription or started a trial. It may run
// one collection and keep reading it; further collections need the trial.
func IsFirstLook(s BillingSubscription, trialUsedAt *time.Time) bool {
	return s.Status == "" && trialUsedAt == nil
}

// EffectivePromptLimit is the allowance an organization has now, including the First Look.
func EffectivePromptLimit(s BillingSubscription, trialUsedAt *time.Time, now time.Time) int {
	if IsFirstLook(s, trialUsedAt) {
		return FirstLookPromptLimit
	}

	return s.PromptLimit(now)
}

// GenerationBudget returns how many drafted prompts the initial draft may store, given the
// organization's allowance and stored prompts. It leaves room for the customer's own prompts.
func GenerationBudget(limit, stored int) int {
	return min(max(0, limit-stored), InitialDraftPromptLimit)
}

func (s BillingSubscription) inTrial(now time.Time) bool {
	return s.Status == "trialing" && s.TrialEnd != nil && now.Before(*s.TrialEnd)
}
