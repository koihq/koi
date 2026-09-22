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
	Name    string   `json:"name"`
	Domain  string   `json:"domain"`
	Aliases []string `json:"aliases"`
}

type CompetitorPatchRequest struct {
	Name    *string   `json:"name"`
	Domain  *string   `json:"domain"`
	Aliases *[]string `json:"aliases"`
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
		r.With(middlewares.DecodeBody[CompetitorRequest]()).Post("/", h.Create)
		r.Post("/confirm", h.ConfirmAll)
		r.With(middlewares.DecodeBody[CompetitorPatchRequest]()).Patch("/{competitorID}", h.Update)
		r.Delete("/{competitorID}", h.Delete)
		r.Post("/{competitorID}/confirm", h.Confirm)
	}
}

func (h *CompetitorHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}
	competitors, err := h.querier.List(r.Context(), userID, brandID)
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	api.JSON(w, r, http.StatusOK, competitors)
}

func (h *CompetitorHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}
	req := middlewares.Body[CompetitorRequest](r)
	competitor, err := h.commander.Create(r.Context(), userID, brandID, koidomain.CompetitorParams{
		Name: req.Name, Domain: req.Domain, Aliases: req.Aliases,
	})
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	api.JSON(w, r, http.StatusCreated, competitor)
}

func (h *CompetitorHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}
	competitorID, ok := pathID(w, r, "competitorID")
	if !ok {
		return
	}
	req := middlewares.Body[CompetitorPatchRequest](r)
	competitor, err := h.commander.Update(r.Context(), userID, brandID, competitorID, koidomain.CompetitorPatch{
		Name: req.Name, Domain: req.Domain, Aliases: req.Aliases,
	})
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	api.JSON(w, r, http.StatusOK, competitor)
}

func (h *CompetitorHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}
	competitorID, ok := pathID(w, r, "competitorID")
	if !ok {
		return
	}
	if err := h.commander.Delete(r.Context(), userID, brandID, competitorID); err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CompetitorHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	userID, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}
	competitorID, ok := pathID(w, r, "competitorID")
	if !ok {
		return
	}
	competitor, err := h.commander.Confirm(r.Context(), userID, brandID, competitorID)
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	api.JSON(w, r, http.StatusOK, competitor)
}

func (h *CompetitorHandler) ConfirmAll(w http.ResponseWriter, r *http.Request) {
	userID, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}
	competitors, err := h.commander.ConfirmAll(r.Context(), userID, brandID)
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	api.JSON(w, r, http.StatusOK, competitors)
}
