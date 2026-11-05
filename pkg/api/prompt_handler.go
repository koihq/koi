package api

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/koihq/koi/pkg/api/middlewares"
	"github.com/koihq/koi/pkg/domain"
)

type BrandRequestScope struct {
	BrandID uuid.UUID `json:"brandId"`
}

type CreatePromptReq struct {
	BrandRequestScope
	Text string `json:"text"`
}

type UpdatePromptReq struct {
	Text     *string `json:"text"`
	Inactive *bool   `json:"inactive"`
}

type PromptRes struct {
	ID         uuid.UUID                `json:"id"`
	BrandID    uuid.UUID                `json:"brandId"`
	Text       string                   `json:"text"`
	Source     domain.PromptSource      `json:"source"`
	Confirmed  bool                     `json:"confirmed"`
	Inactive   bool                     `json:"inactive"`
	HasResults bool                     `json:"hasResults"`
	Visibility *domain.VisibilityReport `json:"visibility,omitempty"`
}

func PromptToRes(v *domain.Prompt) PromptRes {
	return PromptRes{
		ID:         v.ID,
		BrandID:    v.BrandID,
		Text:       v.Text,
		Source:     v.Source,
		Confirmed:  v.Confirmed,
		Inactive:   v.Inactive,
		HasResults: v.HasResults,
		Visibility: v.Visibility,
	}
}

type PromptHandler struct {
	querier   domain.PromptQuerier
	commander domain.PromptCommander
}

func NewPromptHandler(q domain.PromptQuerier, c domain.PromptCommander) *PromptHandler {
	return &PromptHandler{querier: q, commander: c}
}

func (h *PromptHandler) Routes() func(chi.Router) {
	return func(r chi.Router) {
		r.Get("/", List(promptFilter, h.querier.ListWithVisibility, PromptToRes))
		r.With(middlewares.DecodeBody[CreatePromptReq]()).Post("/", Create(h.create, PromptToRes))
		r.With(middlewares.DecodeBody[BrandRequestScope]()).Post("/confirm", Command(h.confirmAll))
		r.Get("/{id}", Get(h.querier.Get, PromptToRes))
		r.With(middlewares.DecodeBody[UpdatePromptReq]()).Patch("/{id}", Update(h.update, PromptToRes))
		r.Delete("/{id}", Delete(h.commander.Delete))
		r.Post("/{id}/confirm", Action(h.commander.Confirm, PromptToRes))
	}
}

func promptFilter(r *http.Request) (domain.PromptFilter, error) {
	brand, err := brandFilter(r)
	if err != nil {
		return domain.PromptFilter{}, err
	}

	query, err := optionalStringQuery(r, "q")
	if err != nil {
		return domain.PromptFilter{}, err
	}

	sortValue, err := optionalStringQuery(r, "sort")
	if err != nil {
		return domain.PromptFilter{}, err
	}

	sort := domain.PromptSort(sortValue)
	if sort == "" {
		sort = domain.PromptSortOriginal
	}

	switch sort {
	case domain.PromptSortOriginal, domain.PromptSortText, domain.PromptSortRate, domain.PromptSortChange:
	default:
		return domain.PromptFilter{}, domain.Validation(domain.FieldError{Field: "sort", Code: domain.FieldInvalid})
	}

	descending, err := optionalBoolQuery(r, "descending")
	if err != nil {
		return domain.PromptFilter{}, err
	}

	days, engines, err := visibilityScope(r)
	if err != nil {
		return domain.PromptFilter{}, err
	}

	confirmed, err := optionalBoolQuery(r, "confirmed")
	if err != nil {
		return domain.PromptFilter{}, err
	}

	inactive, err := optionalBoolQuery(r, "inactive")
	if err != nil {
		return domain.PromptFilter{}, err
	}

	filter := domain.PromptFilter{
		BrandFilter: brand,
		Query:       query,
		Confirmed:   confirmed,
		Inactive:    inactive,
		Sort:        sort,
		Days:        days,
		Engines:     engines,
	}
	if descending != nil {
		filter.Descending = *descending
	}

	return filter, nil
}

func (h *PromptHandler) create(ctx context.Context, req CreatePromptReq) (*domain.Prompt, error) {
	if err := requireBrand(req.BrandID); err != nil {
		return nil, err
	}

	return h.commander.Create(ctx, req.BrandID, domain.CreatePromptParams{Text: req.Text})
}

func (h *PromptHandler) update(ctx context.Context, id uuid.UUID, req UpdatePromptReq) (*domain.Prompt, error) {
	return h.commander.Update(ctx, id, domain.UpdatePromptParams{Text: req.Text, Inactive: req.Inactive})
}

func (h *PromptHandler) confirmAll(ctx context.Context, req BrandRequestScope) error {
	if err := requireBrand(req.BrandID); err != nil {
		return err
	}

	return h.commander.ConfirmAll(ctx, req.BrandID)
}
