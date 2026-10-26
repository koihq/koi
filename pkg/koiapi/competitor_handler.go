package koiapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/koihq/koi/pkg/koiapi/middlewares"
	apiresponse "github.com/koihq/koi/pkg/koiapi/response"
	"github.com/koihq/koi/pkg/koidomain"
)

type CompetitorRequest struct {
	BrandRequestScope
	Name           string   `json:"name"`
	Domain         string   `json:"domain"`
	GitHubRepo     string   `json:"githubRepo"`
	Aliases        []string `json:"aliases"`
	AdditionalURLs []string `json:"additionalUrls"`
}

type CompetitorPatchRequest struct {
	Name           *string   `json:"name"`
	Domain         *string   `json:"domain"`
	GitHubRepo     *string   `json:"githubRepo"`
	Aliases        *[]string `json:"aliases"`
	AdditionalURLs *[]string `json:"additionalUrls"`
}

type CompetitorHandler struct {
	querier   koidomain.CompetitorQuerier
	commander koidomain.CompetitorCommander
}

func NewCompetitorHandler(q koidomain.CompetitorQuerier, c koidomain.CompetitorCommander) *CompetitorHandler {
	return &CompetitorHandler{querier: q, commander: c}
}

func (h *CompetitorHandler) Routes() func(chi.Router) {
	return func(r chi.Router) {
		r.Get("/", h.List)
		r.Get("/search", h.Search)
		r.With(middlewares.DecodeBody[CompetitorRequest]()).Post("/", h.Create)
		r.With(middlewares.DecodeBody[BrandRequestScope]()).Post("/confirm", h.ConfirmAll)
		r.Get("/{competitorID}", Get("competitorID", h.querier.Get, CompetitorToResponse))
		r.Delete("/{competitorID}", Delete("competitorID", h.commander.Delete))
		r.With(middlewares.DecodeBody[CompetitorPatchRequest]()).Patch("/{competitorID}", h.Patch)
		r.Post("/{competitorID}/confirm", h.Confirm)
	}
}

func (h *CompetitorHandler) Search(w http.ResponseWriter, r *http.Request) {
	_, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}

	query, err := optionalStringQuery(r, "q")
	if err != nil {
		apiresponse.WriteDomainError(w, r, err)

		return
	}

	results, err := h.querier.Search(r.Context(), brandID, query)
	if err != nil {
		apiresponse.WriteDomainError(w, r, err)

		return
	}

	apiresponse.JSON(w, r, http.StatusOK, results)
}

func (h *CompetitorHandler) List(w http.ResponseWriter, r *http.Request) {
	_, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}

	params, err := readListQuery(r)
	if err != nil {
		apiresponse.WriteDomainError(w, r, err)

		return
	}

	competitors, err := h.querier.List(r.Context(), brandID, params.Page, params.Query, params.Confirmed)
	if err != nil {
		apiresponse.WriteDomainError(w, r, err)

		return
	}

	apiresponse.JSON(w, r, http.StatusOK, mapPage(competitors, CompetitorToResponse))
}

func (h *CompetitorHandler) Create(w http.ResponseWriter, r *http.Request) {
	brandID := middlewares.Body[CompetitorRequest](r).BrandID
	if brandID == uuid.Nil {
		apiresponse.WriteDomainError(w, r, koidomain.Validation(koidomain.FieldError{Field: "brandId", Code: koidomain.FieldInvalid}))
		return
	}

	req := middlewares.Body[CompetitorRequest](r)

	competitor, err := h.commander.Create(r.Context(), brandID, koidomain.CompetitorParams{
		Name:           req.Name,
		Domain:         req.Domain,
		GitHubRepo:     req.GitHubRepo,
		Aliases:        req.Aliases,
		AdditionalURLs: req.AdditionalURLs,
	})
	if err != nil {
		apiresponse.WriteDomainError(w, r, err)

		return
	}

	apiresponse.JSON(w, r, http.StatusCreated, CompetitorToResponse(competitor))
}

func (h *CompetitorHandler) Patch(w http.ResponseWriter, r *http.Request) {
	competitorID, ok := pathID(w, r, "competitorID")
	if !ok {
		return
	}

	req := middlewares.Body[CompetitorPatchRequest](r)
	competitor, err := h.commander.Patch(r.Context(), competitorID, koidomain.CompetitorUpdate{
		Name:           req.Name,
		Domain:         req.Domain,
		GitHubRepo:     req.GitHubRepo,
		Aliases:        req.Aliases,
		AdditionalURLs: req.AdditionalURLs,
	})
	if err != nil {
		apiresponse.WriteDomainError(w, r, err)
		return
	}

	apiresponse.JSON(w, r, http.StatusOK, CompetitorToResponse(competitor))
}

func (h *CompetitorHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	competitorID, ok := pathID(w, r, "competitorID")
	if !ok {
		return
	}

	competitor, err := h.commander.Confirm(r.Context(), competitorID)
	if err != nil {
		apiresponse.WriteDomainError(w, r, err)

		return
	}

	apiresponse.JSON(w, r, http.StatusOK, CompetitorToResponse(competitor))
}

func (h *CompetitorHandler) ConfirmAll(w http.ResponseWriter, r *http.Request) {
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
