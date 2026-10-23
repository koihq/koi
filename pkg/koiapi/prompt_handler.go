package koiapi

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/koihq/koi/pkg/koiapi/middlewares"
	apiresponse "github.com/koihq/koi/pkg/koiapi/response"
	"github.com/koihq/koi/pkg/koidomain"
)

type BrandRequestScope struct {
	BrandID uuid.UUID `json:"brandId"`
}

type PromptRequest struct {
	BrandRequestScope
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
}

func NewPromptHandler(q koidomain.PromptQuerier, c koidomain.PromptCommander, analytics ...*koidomain.VisibilityAnalyticsQuerier) *PromptHandler {
	h := &PromptHandler{querier: q, commander: c}
	if len(analytics) > 0 {
		h.analytics = analytics[0]
	}
	return h
}

func (h *PromptHandler) Routes() func(chi.Router) {
	return func(r chi.Router) {
		r.Get("/", h.List)
		r.With(middlewares.DecodeBody[PromptRequest]()).Post("/", Create(h.create, PromptToResponse))
		r.With(middlewares.DecodeBody[BrandRequestScope]()).Post("/confirm", h.ConfirmAll)
		r.Get("/{promptID}", Get("promptID", h.querier.Get, PromptToResponse))
		r.With(middlewares.DecodeBody[PromptPatchRequest]()).Patch("/{promptID}", Update("promptID", h.update, PromptToResponse))
		r.Delete("/{promptID}", Delete("promptID", h.commander.Delete))
		r.Post("/{promptID}/confirm", h.Confirm)
	}
}

func pathID(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, name))
	if err != nil {
		apiresponse.WriteDomainError(w, r, koidomain.ErrNotFound)

		return uuid.Nil, false
	}

	return id, true
}

func requestOwnerAndBrand(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	userID, ok := authenticatedUserID(w, r)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}

	value, err := optionalStringQuery(r, "brandId")
	brandID, parseErr := uuid.Parse(value)
	ok = err == nil && parseErr == nil && brandID != uuid.Nil
	if !ok {
		apiresponse.WriteDomainError(w, r, koidomain.Validation(koidomain.FieldError{Field: "brandId", Code: koidomain.FieldInvalid}))
	}

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
		apiresponse.WriteDomainError(w, r, err)

		return
	}

	prompts, err := h.querier.List(r.Context(), brandID, params)
	if err != nil {
		apiresponse.WriteDomainError(w, r, err)

		return
	}
	if h.analytics != nil {
		ids := make([]uuid.UUID, 0, len(prompts.Items))
		for _, prompt := range prompts.Items {
			ids = append(ids, prompt.ID)
		}
		byPrompt, _, err := h.analytics.Aggregate(r.Context(), brandID, ids, params.Days, params.Engines)
		if err != nil {
			apiresponse.WriteDomainError(w, r, err)
			return
		}
		for i := range prompts.Items {
			prompts.Items[i].Visibility = byPrompt[prompts.Items[i].ID]
		}
	}

	apiresponse.JSON(w, r, http.StatusOK, mapPage(prompts, PromptToResponse))
}

func (h *PromptHandler) create(ctx context.Context, req PromptRequest) (*koidomain.Prompt, error) {
	if req.BrandID == uuid.Nil {
		return nil, koidomain.Validation(koidomain.FieldError{Field: "brandId", Code: koidomain.FieldInvalid})
	}
	return h.commander.Create(ctx, req.BrandID, koidomain.PromptParams{Text: req.Text})
}

func (h *PromptHandler) update(ctx context.Context, id uuid.UUID, req PromptPatchRequest) (*koidomain.Prompt, error) {
	return h.commander.Update(ctx, id, koidomain.PromptPatch{Text: req.Text, Inactive: req.Inactive})
}

func (h *PromptHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	promptID, ok := pathID(w, r, "promptID")
	if !ok {
		return
	}

	prompt, err := h.commander.Confirm(r.Context(), promptID)
	if err != nil {
		apiresponse.WriteDomainError(w, r, err)

		return
	}

	apiresponse.JSON(w, r, http.StatusOK, PromptToResponse(prompt))
}

func (h *PromptHandler) ConfirmAll(w http.ResponseWriter, r *http.Request) {
	brandID := middlewares.Body[BrandRequestScope](r).BrandID
	if brandID == uuid.Nil {
		apiresponse.WriteDomainError(w, r, koidomain.Validation(koidomain.FieldError{Field: "brandId", Code: koidomain.FieldInvalid}))
		return
	}

	if err := h.commander.ConfirmAll(r.Context(), brandID); err != nil {
		apiresponse.WriteDomainError(w, r, err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
