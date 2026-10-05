package koiapi

import (
	"net/http"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"

	"github.com/koihq/koi/pkg/koiapi/api"
	"github.com/koihq/koi/pkg/koiapi/middlewares"
	"github.com/koihq/koi/pkg/koidomain"
)

type CompetitorRequest struct {
	Name       string `json:"name"`
	Domain     string `json:"domain"`
	GitHubRepo string `json:"github_repo"`
}

type CompetitorHandler struct {
	querier   koidomain.CompetitorQuerier
	commander koidomain.CompetitorCommander
	sess      *scs.SessionManager
}

func NewCompetitorHandler(q koidomain.CompetitorQuerier, c koidomain.CompetitorCommander, sess *scs.SessionManager) *CompetitorHandler {
	return &CompetitorHandler{querier: q, commander: c, sess: sess}
}

func (h *CompetitorHandler) Routes() func(chi.Router) {
	return func(r chi.Router) {
		r.Use(Authorizer(h.sess))
		r.Get("/", h.List)
		r.Get("/search", h.Search)
		r.With(middlewares.DecodeBody[CompetitorRequest]()).Post("/", h.Create)
		r.Post("/confirm", h.ConfirmAll)
		r.Get("/{competitorID}", h.Get)
		r.Delete("/{competitorID}", h.Delete)
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
		api.WriteDomainError(w, r, err)
		return
	}
	results, err := h.querier.Search(r.Context(), brandID, query)
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	api.JSON(w, r, http.StatusOK, results)
}

func (h *CompetitorHandler) List(w http.ResponseWriter, r *http.Request) {
	_, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}
	params, err := readListQuery(r)
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	competitors, err := h.querier.List(r.Context(), brandID, params.Page, params.Query, params.Confirmed)
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	api.JSON(w, r, http.StatusOK, competitors)
}

func (h *CompetitorHandler) Get(w http.ResponseWriter, r *http.Request) {
	_, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}
	competitorID, ok := pathID(w, r, "competitorID")
	if !ok {
		return
	}
	competitor, err := h.querier.Get(r.Context(), brandID, competitorID)
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	api.JSON(w, r, http.StatusOK, competitor)
}

func (h *CompetitorHandler) Create(w http.ResponseWriter, r *http.Request) {
	_, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}
	req := middlewares.Body[CompetitorRequest](r)
	competitor, err := h.commander.Create(r.Context(), brandID, koidomain.CompetitorParams{
		Name: req.Name, Domain: req.Domain, GitHubRepo: req.GitHubRepo,
	})
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	api.JSON(w, r, http.StatusCreated, competitor)
}

func (h *CompetitorHandler) Delete(w http.ResponseWriter, r *http.Request) {
	_, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}
	competitorID, ok := pathID(w, r, "competitorID")
	if !ok {
		return
	}
	if err := h.commander.Delete(r.Context(), brandID, competitorID); err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CompetitorHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	_, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}
	competitorID, ok := pathID(w, r, "competitorID")
	if !ok {
		return
	}
	competitor, err := h.commander.Confirm(r.Context(), brandID, competitorID)
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	api.JSON(w, r, http.StatusOK, competitor)
}

func (h *CompetitorHandler) ConfirmAll(w http.ResponseWriter, r *http.Request) {
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
