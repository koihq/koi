package koiapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/koihq/koi/pkg/koiapi/api"
	"github.com/koihq/koi/pkg/koiapi/middlewares"
	"github.com/koihq/koi/pkg/koidomain"
	"github.com/koihq/koi/pkg/koipricing"
	"github.com/koihq/koi/pkg/koistripe"
)

type BillingRepository interface {
	OrganizationForUser(context.Context) (*koidomain.Organization, error)
	SaveCustomer(context.Context, uuid.UUID, string) (string, error)
	Subscription(context.Context, uuid.UUID) (*koidomain.BillingSubscription, error)
	ReserveCheckout(context.Context, uuid.UUID, string, string, string, int) (*koidomain.BillingCheckoutSession, error)
	FinalizeCheckout(context.Context, uuid.UUID, string, string, time.Time) error
	CompleteCheckout(context.Context, string, string, string, string, string) error
	SyncSubscription(context.Context, string, string, string, string, string, string, string, int, *time.Time, *time.Time, bool) error
	PendingDowngrade(context.Context, uuid.UUID, string, int) error
}

type BillingHandler struct {
	repo          BillingRepository
	stripe        *koistripe.Client
	sess          *scs.SessionManager
	priceID       string
	publicURL     string
	webhookSecret string
}

func NewBillingHandler(repo BillingRepository, stripe *koistripe.Client, sess *scs.SessionManager, priceID, publicURL, webhookSecret string) *BillingHandler {
	return &BillingHandler{repo: repo, stripe: stripe, sess: sess, priceID: priceID, publicURL: publicURL, webhookSecret: webhookSecret}
}

func (h *BillingHandler) Webhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		http.Error(w, "invalid webhook body", http.StatusBadRequest)
		return
	}
	event, err := koistripe.VerifyEvent(body, r.Header.Get("Stripe-Signature"), h.webhookSecret)
	if err != nil {
		http.Error(w, "invalid webhook signature", http.StatusBadRequest)
		return
	}

	switch event.Type {
	case "checkout.session.completed":
		var session koistripe.CheckoutSession
		err = json.Unmarshal(event.Data.Object, &session)
		if err == nil {
			err = h.repo.CompleteCheckout(r.Context(), event.ID, session.ID, session.Customer, session.Reference, session.Subscription)
		}
	case "customer.subscription.created", "customer.subscription.updated", "customer.subscription.deleted":
		var subscription koistripe.Subscription
		err = json.Unmarshal(event.Data.Object, &subscription)
		if err == nil && event.Type != "customer.subscription.deleted" {
			subscription, err = h.stripe.GetSubscription(r.Context(), subscription.ID)
		}
		if err == nil {
			err = h.syncSubscription(r.Context(), event.ID, subscription)
		}
	}
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *BillingHandler) syncSubscription(ctx context.Context, eventID string, subscription koistripe.Subscription) error {
	if len(subscription.Items.Data) != 1 {
		return errors.New("Stripe subscription must have one price item")
	}
	item := subscription.Items.Data[0]
	periodEnd := subscription.CurrentPeriodEnd
	if periodEnd == 0 {
		periodEnd = item.CurrentPeriodEnd
	}
	var currentPeriodEnd *time.Time
	if periodEnd > 0 {
		value := time.Unix(periodEnd, 0).UTC()
		currentPeriodEnd = &value
	}
	var trialEnd *time.Time
	if subscription.TrialEnd != nil {
		value := time.Unix(*subscription.TrialEnd, 0).UTC()
		trialEnd = &value
	}

	return h.repo.SyncSubscription(
		ctx,
		eventID,
		subscription.Metadata["organization_id"],
		subscription.Customer,
		subscription.ID,
		item.Price.ID,
		subscription.Status,
		subscription.Metadata["price_version"],
		item.Quantity,
		currentPeriodEnd,
		trialEnd,
		subscription.CancelAtPeriodEnd,
	)
}

func (h *BillingHandler) Routes() func(chi.Router) {
	return func(r chi.Router) {
		r.Get("/quote", h.Quote)
		r.Group(func(r chi.Router) {
			r.Use(Authorizer(h.sess))
			r.Get("/subscription", h.Subscription)
			r.With(middlewares.DecodeBody[allowanceRequest]()).Post("/checkout", h.Checkout)
			r.With(middlewares.DecodeBody[allowanceRequest]()).Post("/change-allowance", h.ChangeAllowance)
			r.Post("/portal", h.Portal)
		})
	}
}

func (h *BillingHandler) ChangeAllowance(w http.ResponseWriter, r *http.Request) {
	quantity := middlewares.Body[allowanceRequest](r).Prompts
	if _, err := h.price(r, quantity); err != nil {
		h.writePriceError(w, r, err)
		return
	}
	organization, err := h.repo.OrganizationForUser(r.Context())
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	local, err := h.repo.Subscription(r.Context(), organization.ID)
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	if local == nil || local.Status != "active" && local.Status != "trialing" || local.CancelAtPeriodEnd {
		api.WriteDomainError(w, r, koidomain.ErrBillingConflict)
		return
	}
	if local.StripePriceID != h.priceID || local.PriceVersion != koipricing.Current().Version {
		api.WriteDomainError(w, r, koidomain.ErrBillingLegacyPrice)
		return
	}
	if quantity == local.Quantity {
		api.JSON(w, r, http.StatusOK, map[string]any{"quantity": quantity, "effective": "current"})
		return
	}

	current, err := h.stripe.GetSubscription(r.Context(), local.StripeSubscriptionID)
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	if organization.StripeCustomerID == nil || current.Customer != *organization.StripeCustomerID || current.Metadata["organization_id"] != organization.ID.String() ||
		len(current.Items.Data) != 1 || current.Items.Data[0].Price.ID != local.StripePriceID ||
		current.Items.Data[0].Quantity != local.Quantity {
		api.WriteDomainError(w, r, errors.New("Stripe subscription differs from local billing state"))
		return
	}

	if quantity > local.Quantity {
		if current.Schedule != nil {
			api.WriteDomainError(w, r, koidomain.ErrBillingConflict)
			return
		}
		_, err = h.stripe.Upgrade(r.Context(), current.ID, current.Items.Data[0].ID, quantity)
		if err != nil {
			api.WriteDomainError(w, r, err)
			return
		}
		api.JSON(w, r, http.StatusAccepted, map[string]any{"quantity": quantity, "effective": "after_payment"})
		return
	}

	scheduleID := current.Schedule
	if scheduleID == nil {
		schedule, err := h.stripe.CreateSchedule(r.Context(), current.ID)
		if err != nil {
			api.WriteDomainError(w, r, err)
			return
		}
		scheduleID = &schedule.ID
	}
	schedule, err := h.stripe.GetSchedule(r.Context(), *scheduleID)
	if err == nil {
		periodEnd := current.CurrentPeriodEnd
		if periodEnd == 0 {
			periodEnd = current.Items.Data[0].CurrentPeriodEnd
		}
		err = h.stripe.ScheduleDowngrade(r.Context(), schedule, local.StripePriceID, local.Quantity, quantity, periodEnd)
	}
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	if err := h.repo.PendingDowngrade(r.Context(), organization.ID, *scheduleID, quantity); err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	api.JSON(w, r, http.StatusAccepted, map[string]any{"quantity": quantity, "effective": "next_cycle"})
}

type allowanceRequest struct {
	Prompts int `json:"prompts"`
}

func (h *BillingHandler) price(r *http.Request, quantity int) (int64, error) {
	catalog := koipricing.Current()
	want, err := catalog.Quote(quantity)
	if err != nil {
		return 0, err
	}

	price, err := h.stripe.GetPrice(r.Context(), h.priceID)
	if err != nil {
		return 0, err
	}

	if err := price.ValidateCatalog(catalog); err != nil {
		return 0, err
	}
	amount, err := price.MonthlyAmount(quantity)
	if err != nil {
		return 0, err
	}
	if amount != want {
		return 0, errors.New("Stripe price does not match the configured pricing version")
	}

	return amount, nil
}

func (h *BillingHandler) Quote(w http.ResponseWriter, r *http.Request) {
	quantity, err := strconv.Atoi(r.URL.Query().Get("prompts"))
	if err != nil {
		api.WriteDomainError(w, r, koidomain.Validation(koidomain.FieldError{Field: "prompts", Code: koidomain.FieldInvalid}))
		return
	}

	amount, err := h.price(r, quantity)
	if err != nil {
		h.writePriceError(w, r, err)
		return
	}

	api.JSON(w, r, http.StatusOK, map[string]any{
		"prompts":       quantity,
		"amount_cents":  amount,
		"currency":      "eur",
		"interval":      "month",
		"price_version": koipricing.Current().Version,
	})
}

func (h *BillingHandler) Checkout(w http.ResponseWriter, r *http.Request) {
	quantity := middlewares.Body[allowanceRequest](r).Prompts
	if _, err := h.price(r, quantity); err != nil {
		h.writePriceError(w, r, err)
		return
	}

	organization, err := h.repo.OrganizationForUser(r.Context())
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	customerID := ""
	if organization.StripeCustomerID != nil {
		customerID = *organization.StripeCustomerID
	} else {
		customerID, err = h.stripe.CreateCustomer(r.Context(), organization.ID.String())
		if err != nil {
			api.WriteDomainError(w, r, err)
			return
		}
		customerID, err = h.repo.SaveCustomer(r.Context(), organization.ID, customerID)
		if err != nil {
			api.WriteDomainError(w, r, err)
			return
		}
	}

	priceVersion := koipricing.Current().Version
	reservation, err := h.repo.ReserveCheckout(r.Context(), organization.ID, customerID, h.priceID, priceVersion, quantity)
	if errors.Is(err, koidomain.ErrBillingSubscriptionExists) || errors.Is(err, koidomain.ErrBillingCheckoutPending) {
		api.WriteDomainError(w, r, err)
		return
	}
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	if reservation.URL != "" {
		api.JSON(w, r, http.StatusOK, map[string]string{"url": reservation.URL})
		return
	}

	session, err := h.stripe.CreateCheckout(r.Context(), customerID, organization.ID.String(), h.priceID, priceVersion, reservation.ID.String(), quantity, h.publicURL, false)
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	if err := h.repo.FinalizeCheckout(r.Context(), reservation.ID, session.ID, session.URL, time.Unix(session.ExpiresAt, 0).UTC()); err != nil {
		api.WriteDomainError(w, r, err)
		return
	}

	api.JSON(w, r, http.StatusOK, map[string]string{"url": session.URL})
}

func (h *BillingHandler) Subscription(w http.ResponseWriter, r *http.Request) {
	organization, err := h.repo.OrganizationForUser(r.Context())
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	subscription, err := h.repo.Subscription(r.Context(), organization.ID)
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}

	limit := 0
	if subscription != nil {
		limit = subscription.PromptLimit(time.Now().UTC())
	}
	api.JSON(w, r, http.StatusOK, map[string]any{"subscription": subscription, "prompt_limit": limit})
}

func (h *BillingHandler) Portal(w http.ResponseWriter, r *http.Request) {
	organization, err := h.repo.OrganizationForUser(r.Context())
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	if organization.StripeCustomerID == nil {
		api.WriteDomainError(w, r, koidomain.ErrBillingConflict)
		return
	}

	url, err := h.stripe.CreatePortal(r.Context(), *organization.StripeCustomerID, h.publicURL)
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	api.JSON(w, r, http.StatusOK, map[string]string{"url": url})
}

func (h *BillingHandler) writePriceError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, koipricing.ErrPromptRange) {
		api.WriteDomainError(w, r, koidomain.Validation(koidomain.FieldError{Field: "prompts", Code: koidomain.FieldInvalid}))
		return
	}
	api.WriteDomainError(w, r, err)
}
