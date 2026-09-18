package koiapi

import (
	"math"
	"net/http"
	"strconv"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/koihq/koi/pkg/koiapi/api"
	"github.com/koihq/koi/pkg/koiapi/middlewares"
	"github.com/koihq/koi/pkg/koidomain"
)

type BrandRequest struct {
	Name        string `json:"name"`
	Domain      string `json:"domain"`
	GitHubRepo  string `json:"github_repo"`
	Country     string `json:"country"`
	Description string `json:"description"`
}

type BrandPatchRequest struct {
	Name        *string `json:"name"`
	Domain      *string `json:"domain"`
	GitHubRepo  *string `json:"github_repo"`
	Country     *string `json:"country"`
	Description *string `json:"description"`
}

type BrandHandler struct {
	querier   koidomain.BrandQuerier
	commander koidomain.BrandCommander
	sess      *scs.SessionManager
}

func NewBrandHandler(q koidomain.BrandQuerier, c koidomain.BrandCommander, sess *scs.SessionManager) *BrandHandler {
	return &BrandHandler{querier: q, commander: c, sess: sess}
}

func (h *BrandHandler) Routes() func(chi.Router) {
	return func(r chi.Router) {
		r.Use(Authorizer(h.sess))
		r.With(middlewares.DecodeBody[BrandRequest]()).Post("/", h.Create)
		r.Get("/", h.List)
		r.Get("/countries", h.Countries)
		r.Get("/{id}", h.Get)
		r.With(middlewares.DecodeBody[BrandPatchRequest]()).Patch("/{id}", h.Update)
	}
}

func brandID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		api.WriteDomainError(w, r, koidomain.ErrNotFound)
		return uuid.Nil, false
	}
	return id, true
}

func (h *BrandHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := authenticatedUserID(w, r)
	if !ok {
		return
	}
	req := middlewares.Body[BrandRequest](r)
	brand, err := h.commander.Create(r.Context(), userID, koidomain.BrandParams{
		Name: req.Name, Domain: req.Domain, GitHubRepo: req.GitHubRepo,
		Country: req.Country, Description: req.Description,
	})
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	api.JSON(w, r, http.StatusCreated, brand)
}

func (h *BrandHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := authenticatedUserID(w, r)
	if !ok {
		return
	}
	page, err := positiveQueryInt(r, "page", 1, math.MaxInt)
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	pageSize, err := positiveQueryInt(r, "pageSize", 10, 100)
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	brands, err := h.querier.List(r.Context(), userID, page, pageSize)
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	api.JSON(w, r, http.StatusOK, brands)
}

func positiveQueryInt(r *http.Request, key string, defaultValue, maxValue int) (int, error) {
	values, present := r.URL.Query()[key]
	if !present {
		return defaultValue, nil
	}
	if len(values) != 1 {
		return 0, koidomain.Validation(koidomain.FieldError{Field: key, Code: koidomain.FieldInvalid})
	}
	value, err := strconv.Atoi(values[0])
	if err != nil || value < 1 || value > maxValue {
		return 0, koidomain.Validation(koidomain.FieldError{Field: key, Code: koidomain.FieldInvalid})
	}
	return value, nil
}

func (h *BrandHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := authenticatedUserID(w, r)
	if !ok {
		return
	}
	id, ok := brandID(w, r)
	if !ok {
		return
	}
	brand, err := h.querier.Get(r.Context(), userID, id)
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	api.JSON(w, r, http.StatusOK, brand)
}

func (h *BrandHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := authenticatedUserID(w, r)
	if !ok {
		return
	}
	id, ok := brandID(w, r)
	if !ok {
		return
	}
	req := middlewares.Body[BrandPatchRequest](r)
	brand, err := h.commander.Update(r.Context(), userID, id, koidomain.BrandPatch{
		Name: req.Name, Domain: req.Domain, GitHubRepo: req.GitHubRepo,
		Country: req.Country, Description: req.Description,
	})
	if err != nil {
		api.WriteDomainError(w, r, err)
		return
	}
	api.JSON(w, r, http.StatusOK, brand)
}

func (h *BrandHandler) Countries(w http.ResponseWriter, r *http.Request) {
	api.JSON(w, r, http.StatusOK, koidomain.SupportedCountries())
}
