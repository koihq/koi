package koiapi

import (
	"net/http"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/koihq/koi/pkg/koiapi/api"
	"github.com/koihq/koi/pkg/koiapi/middlewares"
	"github.com/koihq/koi/pkg/koidomain"
)

type PromptRequest struct {
	Text string `json:"text"`
}

type PromptPatchRequest struct {
	Text     *string `json:"text"`
	Inactive *bool   `json:"inactive"`
}

type PromptHandler struct {
	querier   koidomain.PromptQuerier
	commander koidomain.PromptCommander
	analytics *koidomain.VisibilityAnalyticsQuerier
	sess      *scs.SessionManager
}

func NewPromptHandler(q koidomain.PromptQuerier, c koidomain.PromptCommander, sess *scs.SessionManager, analytics ...*koidomain.VisibilityAnalyticsQuerier) *PromptHandler {
	h := &PromptHandler{querier: q, commander: c, sess: sess}
	if len(analytics) > 0 {
		h.analytics = analytics[0]
	}
	return h
}

func (h *PromptHandler) Routes() func(chi.Router) {
	return func(r chi.Router) {
		r.Use(Authorizer(h.sess))
		r.Get("/", h.List)
		r.With(middlewares.DecodeBody[PromptRequest]()).Post("/", h.Create)
		r.Post("/confirm", h.ConfirmAll)
		r.Get("/{promptID}", h.Get)
		r.With(middlewares.DecodeBody[PromptPatchRequest]()).Patch("/{promptID}", h.Update)
		r.Delete("/{promptID}", h.Delete)
		r.Post("/{promptID}/confirm", h.Confirm)
	}
}

func pathID(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, name))
	if err != nil {
		api.WriteDomainError(w, r, koidomain.ErrNotFound)

		return uuid.Nil, false
	}

	return id, true
}

func requestOwnerAndBrand(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	userID, ok := authenticatedUserID(w, r)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}

	brandID, ok := pathID(w, r, "brandID")

	return userID, brandID, ok
}

func promptListQuery(r *http.Request, includeConfirmed bool) (koidomain.PromptListParams, error) {
	page, err := pageParams(r)
	if err != nil {
		return koidomain.PromptListParams{}, err
	}

	query, err := optionalStringQuery(r, "q")
	if err != nil {
		return koidomain.PromptListParams{}, err
	}

	sortValue, err := optionalStringQuery(r, "sort")
	if err != nil {
		return koidomain.PromptListParams{}, err
	}
	sort := koidomain.PromptSort(sortValue)
	if sort == "" {
		sort = koidomain.PromptSortOriginal
	}
	switch sort {
	case koidomain.PromptSortOriginal, koidomain.PromptSortText, koidomain.PromptSortRate, koidomain.PromptSortChange:
	default:
		return koidomain.PromptListParams{}, koidomain.Validation(koidomain.FieldError{Field: "sort", Code: koidomain.FieldInvalid})
	}

	descending, err := optionalBoolQuery(r, "descending")
	if err != nil {
		return koidomain.PromptListParams{}, err
	}

	days, engines, err := visibilityScope(r)
	if err != nil {
		return koidomain.PromptListParams{}, err
	}

	params := koidomain.PromptListParams{
		Page:    page,
		Query:   query,
		Sort:    sort,
		Days:    days,
		Engines: engines,
	}
	if descending != nil {
		params.Descending = *descending
	}
	if includeConfirmed {
		params.Confirmed, err = optionalBoolQuery(r, "confirmed")
	}

	return params, err
}

func (h *PromptHandler) List(w http.ResponseWriter, r *http.Request) {
	_, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}

	params, err := promptListQuery(r, true)
	if err != nil {
		api.WriteDomainError(w, r, err)

		return
	}

	prompts, err := h.querier.List(r.Context(), brandID, params)
	if err != nil {
		api.WriteDomainError(w, r, err)

		return
	}
	if h.analytics != nil {
		ids := make([]uuid.UUID, 0, len(prompts.Items))
		for _, prompt := range prompts.Items {
			ids = append(ids, prompt.ID)
		}
		byPrompt, _, err := h.analytics.Aggregate(r.Context(), brandID, ids, params.Days, params.Engines)
		if err != nil {
			api.WriteDomainError(w, r, err)
			return
		}
		for i := range prompts.Items {
			prompts.Items[i].Visibility = byPrompt[prompts.Items[i].ID]
		}
	}

	api.JSON(w, r, http.StatusOK, prompts)
}

func (h *PromptHandler) Get(w http.ResponseWriter, r *http.Request) {
	_, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}

	promptID, ok := pathID(w, r, "promptID")
	if !ok {
		return
	}

	prompt, err := h.querier.Get(r.Context(), brandID, promptID)
	if err != nil {
		api.WriteDomainError(w, r, err)

		return
	}

	api.JSON(w, r, http.StatusOK, prompt)
}

func (h *PromptHandler) Create(w http.ResponseWriter, r *http.Request) {
	_, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}

	prompt, err := h.commander.Create(r.Context(), brandID, koidomain.PromptParams{
		Text: middlewares.Body[PromptRequest](r).Text,
	})
	if err != nil {
		api.WriteDomainError(w, r, err)

		return
	}

	api.JSON(w, r, http.StatusCreated, prompt)
}

func (h *PromptHandler) Update(w http.ResponseWriter, r *http.Request) {
	_, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}

	promptID, ok := pathID(w, r, "promptID")
	if !ok {
		return
	}

	req := middlewares.Body[PromptPatchRequest](r)
	prompt, err := h.commander.Update(r.Context(), brandID, promptID, koidomain.PromptPatch{
		Text:     req.Text,
		Inactive: req.Inactive,
	})
	if err != nil {
		api.WriteDomainError(w, r, err)

		return
	}

	api.JSON(w, r, http.StatusOK, prompt)
}

func (h *PromptHandler) Delete(w http.ResponseWriter, r *http.Request) {
	_, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}

	promptID, ok := pathID(w, r, "promptID")
	if !ok {
		return
	}

	if err := h.commander.Delete(r.Context(), brandID, promptID); err != nil {
		api.WriteDomainError(w, r, err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *PromptHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	_, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}

	promptID, ok := pathID(w, r, "promptID")
	if !ok {
		return
	}

	prompt, err := h.commander.Confirm(r.Context(), brandID, promptID)
	if err != nil {
		api.WriteDomainError(w, r, err)

		return
	}

	api.JSON(w, r, http.StatusOK, prompt)
}

func (h *PromptHandler) ConfirmAll(w http.ResponseWriter, r *http.Request) {
	_, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}

	if err := h.commander.ConfirmAll(r.Context(), brandID); err != nil {
		api.WriteDomainError(w, r, err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
