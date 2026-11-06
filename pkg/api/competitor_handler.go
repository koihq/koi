package api

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/koihq/koi/pkg/api/middlewares"
	"github.com/koihq/koi/pkg/api/response"
	"github.com/koihq/koi/pkg/domain"
)

type CreateCompetitorReq struct {
	BrandRequestScope
	Name           string   `json:"name"`
	Domain         string   `json:"domain"`
	GitHubRepo     string   `json:"githubRepo"`
	Aliases        []string `json:"aliases"`
	AdditionalURLs []string `json:"additionalUrls"`
}

type UpdateCompetitorReq struct {
	Name           *string   `json:"name"`
	Domain         *string   `json:"domain"`
	GitHubRepo     *string   `json:"githubRepo"`
	Aliases        *[]string `json:"aliases"`
	AdditionalURLs *[]string `json:"additionalUrls"`
}

type CompetitorRes struct {
	ID             uuid.UUID               `json:"id"`
	BrandID        uuid.UUID               `json:"brandId"`
	Name           string                  `json:"name"`
	Domain         string                  `json:"domain"`
	GitHubRepo     string                  `json:"githubRepo"`
	Aliases        []string                `json:"aliases"`
	AdditionalURLs []string                `json:"additionalUrls"`
	Source         domain.CompetitorSource `json:"source"`
	Confirmed      bool                    `json:"confirmed"`
}

func CompetitorToRes(v *domain.Competitor) CompetitorRes {
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

type CompetitorAppearanceRes struct {
	CompetitorID *uuid.UUID                  `json:"competitorId"`
	Name         string                      `json:"name"`
	Confirmed    bool                        `json:"confirmed"`
	Evidence     []domain.AppearanceEvidence `json:"evidence"`
}

func CompetitorAppearanceToRes(v *domain.CompetitorAppearance) CompetitorAppearanceRes {
	return CompetitorAppearanceRes{
		CompetitorID: v.CompetitorID,
		Name:         v.Name,
		Confirmed:    v.Confirmed,
		Evidence:     v.Evidence,
	}
}

type CompetitorHandler struct {
	querier   domain.CompetitorQuerier
	commander domain.CompetitorCommander
}

func NewCompetitorHandler(q domain.CompetitorQuerier, c domain.CompetitorCommander) *CompetitorHandler {
	return &CompetitorHandler{querier: q, commander: c}
}

func (h *CompetitorHandler) Routes() func(chi.Router) {
	return func(r chi.Router) {
		r.Get("/", List(competitorFilter, h.querier.List, CompetitorToRes))
		r.Get("/search", h.Search)
		r.With(middlewares.DecodeBody[CreateCompetitorReq]()).Post("/", Create(h.create, CompetitorToRes))
		r.Get("/{id}", Get(h.querier.Get, CompetitorToRes))
		r.With(middlewares.DecodeBody[UpdateCompetitorReq]()).Patch("/{id}", Update(h.update, CompetitorToRes))
		r.Delete("/{id}", Delete(h.commander.Delete))
		r.Post("/{id}/confirm", Action(h.commander.Confirm, CompetitorToRes))
	}
}

func competitorFilter(r *http.Request) (domain.CompetitorFilter, error) {
	brand, err := brandFilter(r)
	if err != nil {
		return domain.CompetitorFilter{}, err
	}

	confirmed, err := optionalBoolQuery(r, "confirmed")
	if err != nil {
		return domain.CompetitorFilter{}, err
	}

	query, err := optionalStringQuery(r, "q")
	if err != nil {
		return domain.CompetitorFilter{}, err
	}

	return domain.CompetitorFilter{
		BrandFilter: brand,
		Query:       query,
		Confirmed:   confirmed,
	}, nil
}

func (h *CompetitorHandler) Search(w http.ResponseWriter, r *http.Request) {
	brand, err := brandFilter(r)
	if err != nil {
		response.WriteDomainError(w, r, err)
		return
	}

	query, err := optionalStringQuery(r, "q")
	if err != nil {
		response.WriteDomainError(w, r, err)
		return
	}

	results, err := h.querier.Search(r.Context(), brand.BrandID, query)
	if err != nil {
		response.WriteDomainError(w, r, err)
		return
	}

	response.JSON(w, r, http.StatusOK, results)
}

func (h *CompetitorHandler) create(ctx context.Context, req CreateCompetitorReq) (*domain.Competitor, error) {
	if err := requireBrand(req.BrandID); err != nil {
		return nil, err
	}

	return h.commander.Create(ctx, req.BrandID, domain.CreateCompetitorParams{
		Name:           req.Name,
		Domain:         req.Domain,
		GitHubRepo:     req.GitHubRepo,
		Aliases:        req.Aliases,
		AdditionalURLs: req.AdditionalURLs,
	})
}

func (h *CompetitorHandler) update(ctx context.Context, id uuid.UUID, req UpdateCompetitorReq) (*domain.Competitor, error) {
	return h.commander.Update(ctx, id, domain.UpdateCompetitorParams{
		Name:           req.Name,
		Domain:         req.Domain,
		GitHubRepo:     req.GitHubRepo,
		Aliases:        req.Aliases,
		AdditionalURLs: req.AdditionalURLs,
	})
}
