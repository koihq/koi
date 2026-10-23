package koiapi

import (
	"time"

	"github.com/google/uuid"

	"github.com/koihq/koi/pkg/koidomain"
)

type BrandRes struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organizationId"`
	Name           string    `json:"name"`
	Domain         string    `json:"domain"`
	GitHubRepo     string    `json:"githubRepo"`
	Country        string    `json:"country"`
	Description    string    `json:"description"`
}

func BrandToResponse(v *koidomain.Brand) BrandRes {
	return BrandRes{
		ID:             v.ID,
		OrganizationID: v.OrganizationID,
		Name:           v.Name,
		Domain:         v.Domain,
		GitHubRepo:     v.GitHubRepo,
		Country:        v.Country,
		Description:    v.Description,
	}
}

type PromptRes struct {
	ID         uuid.UUID                   `json:"id"`
	BrandID    uuid.UUID                   `json:"brandId"`
	Text       string                      `json:"text"`
	Source     koidomain.PromptSource      `json:"source"`
	Confirmed  bool                        `json:"confirmed"`
	Inactive   bool                        `json:"inactive"`
	Visibility *koidomain.VisibilityReport `json:"visibility,omitempty"`
}

func PromptToResponse(v *koidomain.Prompt) PromptRes {
	return PromptRes{
		ID:         v.ID,
		BrandID:    v.BrandID,
		Text:       v.Text,
		Source:     v.Source,
		Confirmed:  v.Confirmed,
		Inactive:   v.Inactive,
		Visibility: v.Visibility,
	}
}

type CompetitorRes struct {
	ID             uuid.UUID                  `json:"id"`
	BrandID        uuid.UUID                  `json:"brandId"`
	Name           string                     `json:"name"`
	Domain         string                     `json:"domain"`
	GitHubRepo     string                     `json:"githubRepo"`
	Aliases        []string                   `json:"aliases"`
	AdditionalURLs []string                   `json:"additionalUrls"`
	Source         koidomain.CompetitorSource `json:"source"`
	Confirmed      bool                       `json:"confirmed"`
}

func CompetitorToResponse(v *koidomain.Competitor) CompetitorRes {
	return CompetitorRes{
		ID:             v.ID,
		BrandID:        v.BrandID,
		Name:           v.Name,
		Domain:         v.Domain,
		GitHubRepo:     v.GitHubRepo,
		Aliases:        v.Aliases,
		AdditionalURLs: v.AdditionalURLs,
		Source:         v.Source,
		Confirmed:      v.Confirmed,
	}
}

type VisibilityResultRes struct {
	ID             uuid.UUID                 `json:"id"`
	RunID          uuid.UUID                 `json:"runId"`
	PromptID       uuid.UUID                 `json:"promptId"`
	PromptText     string                    `json:"promptText"`
	PromptPosition int                       `json:"promptPosition"`
	Engine         koidomain.EngineID        `json:"engine"`
	Collector      koidomain.CollectorID     `json:"collector"`
	FetchedAt      *time.Time                `json:"fetchedAt"`
	Outcome        *koidomain.Outcome        `json:"outcome"`
	RawAnswer      string                    `json:"rawAnswer"`
	Appearances    []CompetitorAppearanceRes `json:"appearances"`
	Sources        []string                  `json:"sources"`
	ErrorCode      *string                   `json:"errorCode"`
}

func VisibilityResultToResponse(v *koidomain.VisibilityResult) VisibilityResultRes {
	return VisibilityResultRes{
		ID:             v.ID,
		RunID:          v.RunID,
		PromptID:       v.PromptID,
		PromptText:     v.PromptText,
		PromptPosition: v.PromptPosition,
		Engine:         v.Engine,
		Collector:      v.Collector,
		FetchedAt:      v.FetchedAt,
		Outcome:        v.Outcome,
		RawAnswer:      v.RawAnswer,
		Appearances:    mapItems(v.Appearances, CompetitorAppearanceToResponse),
		Sources:        v.Sources,
		ErrorCode:      v.ErrorCode,
	}
}

type HistoricalPromptRes struct {
	PromptID   uuid.UUID                   `json:"promptId"`
	PromptText string                      `json:"promptText"`
	LastSeenAt time.Time                   `json:"lastSeenAt"`
	Visibility *koidomain.VisibilityReport `json:"visibility,omitempty"`
}

func HistoricalPromptToResponse(v *koidomain.HistoricalPrompt) HistoricalPromptRes {
	return HistoricalPromptRes{
		PromptID:   v.PromptID,
		PromptText: v.PromptText,
		LastSeenAt: v.LastSeenAt,
		Visibility: v.Visibility,
	}
}

type TrackingStatusRes struct {
	PromptsReviewed     bool       `json:"promptsReviewed"`
	CompetitorsReviewed bool       `json:"competitorsReviewed"`
	StartedAt           *time.Time `json:"startedAt"`
	NextRunAt           *time.Time `json:"nextRunAt"`
	BlockedReason       *string    `json:"blockedReason"`
}

func TrackingStatusToResponse(v *koidomain.TrackingStatus) TrackingStatusRes {
	return TrackingStatusRes{
		PromptsReviewed:     v.PromptsReviewed,
		CompetitorsReviewed: v.CompetitorsReviewed,
		StartedAt:           v.StartedAt,
		NextRunAt:           v.NextRunAt,
		BlockedReason:       v.BlockedReason,
	}
}

type BillingSubscriptionRes struct {
	ID                      uuid.UUID  `json:"id"`
	OrganizationID          uuid.UUID  `json:"organizationId"`
	Quantity                int        `json:"quantity"`
	Status                  string     `json:"status"`
	CurrentPeriodEnd        *time.Time `json:"currentPeriodEnd"`
	TrialEnd                *time.Time `json:"trialEnd"`
	CancelAtPeriodEnd       bool       `json:"cancelAtPeriodEnd"`
	PendingQuantity         *int       `json:"pendingQuantity"`
	PaymentURL              string     `json:"paymentUrl"`
	PendingUpgradeQuantity  *int       `json:"pendingUpgradeQuantity"`
	PendingUpgradeExpiresAt *time.Time `json:"pendingUpgradeExpiresAt"`
}

func BillingSubscriptionToResponse(v *koidomain.BillingSubscription) BillingSubscriptionRes {
	return BillingSubscriptionRes{
		ID:                      v.ID,
		OrganizationID:          v.OrganizationID,
		Quantity:                v.Quantity,
		Status:                  v.Status,
		CurrentPeriodEnd:        v.CurrentPeriodEnd,
		TrialEnd:                v.TrialEnd,
		CancelAtPeriodEnd:       v.CancelAtPeriodEnd,
		PendingQuantity:         v.PendingQuantity,
		PaymentURL:              v.PaymentURL,
		PendingUpgradeQuantity:  v.PendingUpgradeQuantity,
		PendingUpgradeExpiresAt: v.PendingUpgradeExpiresAt,
	}
}

type BillingStatusRes struct {
	Subscription   *BillingSubscriptionRes `json:"subscription"`
	PromptLimit    int                     `json:"promptLimit"`
	TrialAvailable bool                    `json:"trialAvailable"`
}

func BillingStatusToResponse(v *koidomain.BillingStatus) BillingStatusRes {
	return BillingStatusRes{
		Subscription:   optionalResponse(v.Subscription, BillingSubscriptionToResponse),
		PromptLimit:    v.PromptLimit,
		TrialAvailable: v.TrialAvailable,
	}
}

type BillingQuoteRes struct {
	Prompts     int    `json:"prompts"`
	AmountCents int64  `json:"amountCents"`
	Currency    string `json:"currency"`
	Interval    string `json:"interval"`
}

func BillingQuoteToResponse(v *koidomain.BillingQuote) BillingQuoteRes {
	return BillingQuoteRes{
		Prompts:     v.Prompts,
		AmountCents: v.AmountCents,
		Currency:    v.Currency,
		Interval:    v.Interval,
	}
}

type BillingChangeRes struct {
	Quantity   int        `json:"quantity"`
	Effective  string     `json:"effective"`
	PaymentURL string     `json:"paymentUrl,omitempty"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
}

func BillingChangeToResponse(v *koidomain.BillingChange) BillingChangeRes {
	return BillingChangeRes{
		Quantity:   v.Quantity,
		Effective:  v.Effective,
		PaymentURL: v.PaymentURL,
		ExpiresAt:  v.ExpiresAt,
	}
}

type BillingCatalogRes struct {
	Currency string            `json:"currency"`
	Interval string            `json:"interval"`
	Options  []BillingQuoteRes `json:"options"`
}

func BillingCatalogToResponse(v *koidomain.BillingCatalog) BillingCatalogRes {
	return BillingCatalogRes{
		Currency: v.Currency,
		Interval: v.Interval,
		Options:  mapItems(v.Options, BillingQuoteToResponse),
	}
}

type CompetitorAppearanceRes struct {
	CompetitorID *uuid.UUID                     `json:"competitorId"`
	Name         string                         `json:"name"`
	Confirmed    bool                           `json:"confirmed"`
	Evidence     []koidomain.AppearanceEvidence `json:"evidence"`
}

func CompetitorAppearanceToResponse(v *koidomain.CompetitorAppearance) CompetitorAppearanceRes {
	return CompetitorAppearanceRes{
		CompetitorID: v.CompetitorID,
		Name:         v.Name,
		Confirmed:    v.Confirmed,
		Evidence:     v.Evidence,
	}
}

func mapItems[T, R any](items []T, mapper func(*T) R) []R {
	result := make([]R, 0, len(items))
	for i := range items {
		result = append(result, mapper(&items[i]))
	}
	return result
}
func optionalResponse[T, R any](item *T, mapper func(*T) R) *R {
	if item == nil {
		return nil
	}
	result := mapper(item)
	return &result
}
func mapPage[T, R any](page *koidomain.Page[T], mapper func(*T) R) koidomain.Page[R] {
	return koidomain.Page[R]{Items: mapItems(page.Items, mapper), TotalItems: page.TotalItems, TotalPages: page.TotalPages, CurrentPage: page.CurrentPage, HasNext: page.HasNext, HasPrev: page.HasPrev}
}
