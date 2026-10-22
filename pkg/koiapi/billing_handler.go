package koiapi

import (
	"io"
	"net/http"
	"strconv"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"

	"github.com/koihq/koi/pkg/koiapi/api"
	"github.com/koihq/koi/pkg/koiapi/middlewares"
	"github.com/koihq/koi/pkg/koidomain"
	"github.com/koihq/koi/pkg/koistripe"
)

type BillingHandler struct {
	querier       koidomain.BillingQuerier
	commander     koidomain.BillingCommander
	sess          *scs.SessionManager
	webhookSecret string
}

func NewBillingHandler(q koidomain.BillingQuerier, c koidomain.BillingCommander, sess *scs.SessionManager, webhookSecret string) *BillingHandler {
	return &BillingHandler{querier: q, commander: c, sess: sess, webhookSecret: webhookSecret}
}

func (h *BillingHandler) Routes() func(chi.Router) {
	return func(r chi.Router) {
		r.Get("/catalog", h.Catalog)
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

type allowanceRequest struct {
	Prompts int `json:"prompts"`
}
type billingRedirect struct {
	URL string `json:"url"`
}

func (h *BillingHandler) Catalog(w http.ResponseWriter, r *http.Request) {
	catalog, err := h.querier.Catalog(r.Context())
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}

	api.JSON(w, r, http.StatusOK, catalog)
}

func (h *BillingHandler) Quote(w http.ResponseWriter, r *http.Request) {
	quantity, err := strconv.Atoi(r.URL.Query().Get("prompts"))
	if err != nil {
		api.WriteDomainError(w, r, koidomain.Validation(koidomain.FieldError{Field: "prompts", Code: koidomain.FieldInvalid}))
		return
	}

	quote, err := h.querier.Quote(r.Context(), quantity)
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}

	api.JSON(w, r, http.StatusOK, quote)
}

func (h *BillingHandler) Subscription(w http.ResponseWriter, r *http.Request) {
	status, err := h.querier.Subscription(r.Context())
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}

	api.JSON(w, r, http.StatusOK, status)
}

func (h *BillingHandler) Checkout(w http.ResponseWriter, r *http.Request) {
	url, err := h.commander.Checkout(r.Context(), middlewares.Body[allowanceRequest](r).Prompts)
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}

	api.JSON(w, r, http.StatusOK, billingRedirect{URL: url})
}

func (h *BillingHandler) ChangeAllowance(w http.ResponseWriter, r *http.Request) {
	values := r.Header.Values("Idempotency-Key")
	if len(values) != 1 || !validIdempotencyKey(values[0]) {
		api.WriteDomainError(w, r, koidomain.Validation(koidomain.FieldError{Field: "idempotency_key", Code: koidomain.FieldInvalid}))
		return
	}

	change, err := h.commander.ChangeAllowance(r.Context(), middlewares.Body[allowanceRequest](r).Prompts, values[0])
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}

	status := http.StatusAccepted
	if change.Effective == "current" {
		status = http.StatusOK
	}

	api.JSON(w, r, status, change)
}

func (h *BillingHandler) Portal(w http.ResponseWriter, r *http.Request) {
	url, err := h.commander.Portal(r.Context())
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}

	api.JSON(w, r, http.StatusOK, billingRedirect{URL: url})
}

func (h *BillingHandler) Webhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		http.Error(w, "invalid webhook body", http.StatusBadRequest)
		return
	}

	event, err := koistripe.VerifyEvent(body, r.Header.Get("Stripe-Signature"), h.webhookSecret)
	if err != nil {
		http.Error(w, "invalid webhook event", http.StatusBadRequest)
		return
	}

	if event.SubscriptionID != "" {
		if err := h.commander.ProcessEvent(r.Context(), event); err != nil {
			api.WriteDomainError(w, r, err)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}
