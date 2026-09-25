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
	Name   string `json:"name"`
	Domain string `json:"domain"`
}

type CompetitorHandler struct {
	querier   koidomain.CompetitorQuerier
	commander koidomain.CompetitorCommander
	searcher  CompetitorSearcher
	sess      *scs.SessionManager
}

func NewCompetitorHandler(q koidomain.CompetitorQuerier, c koidomain.CompetitorCommander, searcher CompetitorSearcher, sess *scs.SessionManager) *CompetitorHandler {
	return &CompetitorHandler{querier: q, commander: c, searcher: searcher, sess: sess}
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
	userID, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}
	query, err := optionalStringQuery(r, "q")
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	if len([]rune(query)) < 3 {
		api.WriteDomainError(w, r, koidomain.Validation(koidomain.FieldError{Field: "q", Code: koidomain.FieldMinLength}))
		return
	}
	if _, err := h.querier.List(r.Context(), userID, brandID, koidomain.PageParams{Page: 1, PageSize: 1}, "", nil); err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	results, err := h.searcher.Search(r.Context(), query)
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	api.JSON(w, r, http.StatusOK, results)
}

func (h *CompetitorHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}
	page, err := pageParams(r)
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	confirmed, err := optionalBoolQuery(r, "confirmed")
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	query, err := optionalStringQuery(r, "q")
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	competitors, err := h.querier.List(r.Context(), userID, brandID, page, query, confirmed)
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	api.JSON(w, r, http.StatusOK, competitors)
}

func (h *CompetitorHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}
	competitorID, ok := pathID(w, r, "competitorID")
	if !ok {
		return
	}
	competitor, err := h.querier.Get(r.Context(), userID, brandID, competitorID)
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	api.JSON(w, r, http.StatusOK, competitor)
}

func (h *CompetitorHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}
	req := middlewares.Body[CompetitorRequest](r)
	competitor, err := h.commander.Create(r.Context(), userID, brandID, koidomain.CompetitorParams{
		Name: req.Name, Domain: req.Domain,
	})
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	api.JSON(w, r, http.StatusCreated, competitor)
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
	if err := h.commander.ConfirmAll(r.Context(), userID, brandID); err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
