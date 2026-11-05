package api

import (
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/koihq/koi/pkg/api/middlewares"
	"github.com/koihq/koi/pkg/api/response"
	"github.com/koihq/koi/pkg/domain"
	"github.com/koihq/koi/pkg/providers/stripe"
)

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

func BillingSubscriptionToRes(v *domain.BillingSubscription) BillingSubscriptionRes {
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

func BillingStatusToRes(v *domain.BillingStatus) BillingStatusRes {
	return BillingStatusRes{
		Subscription:   optionalResponse(v.Subscription, BillingSubscriptionToRes),
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

func BillingQuoteToRes(v *domain.BillingQuote) BillingQuoteRes {
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

func BillingChangeToRes(v *domain.BillingChange) BillingChangeRes {
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

func BillingCatalogToRes(v *domain.BillingCatalog) BillingCatalogRes {
	return BillingCatalogRes{
		Currency: v.Currency,
		Interval: v.Interval,
		Options:  mapItems(v.Options, BillingQuoteToRes),
	}
}

type BillingHandler struct {
	querier       domain.BillingQuerier
	commander     domain.BillingCommander
	webhookSecret string
}

func NewBillingHandler(q domain.BillingQuerier, c domain.BillingCommander, webhookSecret string) *BillingHandler {
	return &BillingHandler{querier: q, commander: c, webhookSecret: webhookSecret}
}

func (h *BillingHandler) Routes() func(chi.Router) {
	return func(r chi.Router) {
		r.Get("/catalog", h.Catalog)
		r.Get("/quote", h.Quote)
		r.Get("/subscription", h.Subscription)
		r.With(middlewares.DecodeBody[allowanceRequest]()).Post("/checkout", h.Checkout)
		r.With(middlewares.DecodeBody[confirmCheckoutRequest]()).Post("/confirm-checkout", h.ConfirmCheckout)
		r.With(middlewares.DecodeBody[allowanceRequest]()).Post("/change-allowance", h.ChangeAllowance)
		r.Post("/portal", h.Portal)
	}
}

type allowanceRequest struct {
	Prompts int `json:"prompts"`
}
type confirmCheckoutRequest struct {
	SessionID string `json:"sessionId"`
}
type billingRedirect struct {
	URL string `json:"url"`
}

func (h *BillingHandler) Catalog(w http.ResponseWriter, r *http.Request) {
	catalog, err := h.querier.Catalog(r.Context())
	if err != nil {
		response.WriteDomainError(w, r, err)
		return
	}

	response.JSON(w, r, http.StatusOK, BillingCatalogToRes(&catalog))
}

func (h *BillingHandler) Quote(w http.ResponseWriter, r *http.Request) {
	quantity, err := strconv.Atoi(r.URL.Query().Get("prompts"))
	if err != nil {
		response.WriteDomainError(w, r, domain.Validation(domain.FieldError{Field: "prompts", Code: domain.FieldInvalid}))
		return
	}

	quote, err := h.querier.Quote(r.Context(), quantity)
	if err != nil {
		response.WriteDomainError(w, r, err)
		return
	}

	response.JSON(w, r, http.StatusOK, BillingQuoteToRes(&quote))
}

func (h *BillingHandler) Subscription(w http.ResponseWriter, r *http.Request) {
	status, err := h.querier.Subscription(r.Context())
	if err != nil {
		response.WriteDomainError(w, r, err)
		return
	}

	response.JSON(w, r, http.StatusOK, BillingStatusToRes(&status))
}

func (h *BillingHandler) Checkout(w http.ResponseWriter, r *http.Request) {
	url, err := h.commander.Checkout(r.Context(), middlewares.Body[allowanceRequest](r).Prompts)
	if err != nil {
		response.WriteDomainError(w, r, err)
		return
	}

	response.JSON(w, r, http.StatusOK, billingRedirect{URL: url})
}

func (h *BillingHandler) ConfirmCheckout(w http.ResponseWriter, r *http.Request) {
	status, err := h.commander.ConfirmCheckout(r.Context(), middlewares.Body[confirmCheckoutRequest](r).SessionID)
	if err != nil {
		response.WriteDomainError(w, r, err)
		return
	}
	response.JSON(w, r, http.StatusOK, BillingStatusToRes(&status))
}

func (h *BillingHandler) ChangeAllowance(w http.ResponseWriter, r *http.Request) {
	values := r.Header.Values("Idempotency-Key")
	if len(values) != 1 || !validIdempotencyKey(values[0]) {
		response.WriteDomainError(w, r, domain.Validation(domain.FieldError{Field: "idempotency_key", Code: domain.FieldInvalid}))
		return
	}

	change, err := h.commander.ChangeAllowance(r.Context(), middlewares.Body[allowanceRequest](r).Prompts, values[0])
	if err != nil {
		response.WriteDomainError(w, r, err)
		return
	}

	status := http.StatusAccepted
	if change.Effective == "current" {
		status = http.StatusOK
	}

	response.JSON(w, r, status, BillingChangeToRes(&change))
}

func (h *BillingHandler) Portal(w http.ResponseWriter, r *http.Request) {
	url, err := h.commander.Portal(r.Context())
	if err != nil {
		response.WriteDomainError(w, r, err)
		return
	}

	response.JSON(w, r, http.StatusOK, billingRedirect{URL: url})
}

func (h *BillingHandler) Webhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		http.Error(w, "invalid webhook body", http.StatusBadRequest)
		return
	}

	event, err := stripe.VerifyEvent(body, r.Header.Get("Stripe-Signature"), h.webhookSecret)
	if err != nil {
		http.Error(w, "invalid webhook event", http.StatusBadRequest)
		return
	}

	if event.SubscriptionID != "" {
		if err := h.commander.ProcessEvent(r.Context(), event); err != nil {
			response.WriteDomainError(w, r, err)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}
