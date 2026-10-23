package koiapi

import (
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/koihq/koi/pkg/koiapi/middlewares"
	apiresponse "github.com/koihq/koi/pkg/koiapi/response"
	"github.com/koihq/koi/pkg/koidomain"
)

type BrandRequest struct {
	Name        string `json:"name"`
	Domain      string `json:"domain"`
	GitHubRepo  string `json:"githubRepo"`
	Country     string `json:"country"`
	Description string `json:"description"`
}

type BrandPatchRequest struct {
	Name        *string `json:"name"`
	Domain      *string `json:"domain"`
	GitHubRepo  *string `json:"githubRepo"`
	Country     *string `json:"country"`
	Description *string `json:"description"`
}

type BrandHandler struct {
	querier   koidomain.BrandQuerier
	commander koidomain.BrandCommander
	sess      *scs.SessionManager
	billing   func(http.Handler) http.Handler
}

func NewBrandHandler(q koidomain.BrandQuerier, c koidomain.BrandCommander, sess *scs.SessionManager, billing ...func(http.Handler) http.Handler) *BrandHandler {
	h := &BrandHandler{querier: q, commander: c, sess: sess}
	if len(billing) > 0 {
		h.billing = billing[0]
	}
	return h
}

func (h *BrandHandler) Routes() func(chi.Router) {
	return func(r chi.Router) {
		r.With(middlewares.DecodeBody[BrandRequest]()).Post("/", h.Create)
		r.Get("/", List(pageParams, h.querier.List, BrandToResponse))
		r.Get("/countries", h.Countries)
		r.Get("/{id}", Get("id", h.querier.Get, BrandToResponse))
		if h.billing != nil {
			r.With(h.billing).Post("/{id}/activate", h.Activate)
			r.With(h.billing, middlewares.DecodeBody[BrandPatchRequest]()).Patch("/{id}", h.Update)
		} else {
			r.Post("/{id}/activate", h.Activate)
			r.With(middlewares.DecodeBody[BrandPatchRequest]()).Patch("/{id}", h.Update)
		}
	}
}

func (h *BrandHandler) Activate(w http.ResponseWriter, r *http.Request) {
	_, ok := authenticatedUserID(w, r)
	if !ok {
		return
	}

	id, ok := brandID(w, r)
	if !ok {
		return
	}

	brand, err := h.querier.Get(r.Context(), id)
	if err != nil {
		apiresponse.WriteDomainError(w, r, err)
		return
	}

	h.sess.Put(r.Context(), "active_brand_id", brand.ID.String())
	apiresponse.JSON(w, r, http.StatusOK, BrandToResponse(brand))
}

func brandID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		apiresponse.WriteDomainError(w, r, koidomain.ErrNotFound)

		return uuid.Nil, false
	}

	return id, true
}

func (h *BrandHandler) Create(w http.ResponseWriter, r *http.Request) {
	_, ok := authenticatedUserID(w, r)
	if !ok {
		return
	}

	req := middlewares.Body[BrandRequest](r)

	brand, err := h.commander.Create(r.Context(), koidomain.BrandParams{
		Name:        req.Name,
		Domain:      req.Domain,
		GitHubRepo:  req.GitHubRepo,
		Country:     req.Country,
		Description: req.Description,
	})
	if err != nil {
		apiresponse.WriteDomainError(w, r, err)

		return
	}

	apiresponse.JSON(w, r, http.StatusCreated, BrandToResponse(brand))
}

func pageParams(r *http.Request) (koidomain.PageParams, error) {
	page, err := positiveQueryInt(r, "page", 1, math.MaxInt)
	if err != nil {
		return koidomain.PageParams{}, err
	}

	pageSize, err := positiveQueryInt(r, "pageSize", 10, 100)
	if err != nil {
		return koidomain.PageParams{}, err
	}

	return koidomain.PageParams{Page: page, PageSize: pageSize}, nil
}

type listQuery struct {
	Page      koidomain.PageParams
	Query     string
	Confirmed *bool
}

func readListQuery(r *http.Request) (listQuery, error) {
	page, err := pageParams(r)
	if err != nil {
		return listQuery{}, err
	}

	confirmed, err := optionalBoolQuery(r, "confirmed")
	if err != nil {
		return listQuery{}, err
	}

	query, err := optionalStringQuery(r, "q")
	if err != nil {
		return listQuery{}, err
	}

	return listQuery{Page: page, Query: query, Confirmed: confirmed}, nil
}

func optionalBoolQuery(r *http.Request, key string) (*bool, error) {
	values, present := r.URL.Query()[key]
	if !present {
		return nil, nil
	}

	if len(values) != 1 || values[0] != "true" && values[0] != "false" {
		return nil, koidomain.Validation(koidomain.FieldError{Field: key, Code: koidomain.FieldInvalid})
	}

	value := values[0] == "true"

	return &value, nil
}

func optionalStringQuery(r *http.Request, key string) (string, error) {
	values, present := r.URL.Query()[key]
	if !present {
		return "", nil
	}

	if len(values) != 1 {
		return "", koidomain.Validation(koidomain.FieldError{Field: key, Code: koidomain.FieldInvalid})
	}

	return strings.TrimSpace(values[0]), nil
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

func (h *BrandHandler) Update(w http.ResponseWriter, r *http.Request) {
	_, ok := authenticatedUserID(w, r)
	if !ok {
		return
	}

	id, ok := brandID(w, r)
	if !ok {
		return
	}

	req := middlewares.Body[BrandPatchRequest](r)

	brand, err := h.commander.Update(r.Context(), id, koidomain.BrandPatch{
		Name:        req.Name,
		Domain:      req.Domain,
		GitHubRepo:  req.GitHubRepo,
		Country:     req.Country,
		Description: req.Description,
	})
	if err != nil {
		apiresponse.WriteDomainError(w, r, err)

		return
	}

	apiresponse.JSON(w, r, http.StatusOK, BrandToResponse(brand))
}

func (h *BrandHandler) Countries(w http.ResponseWriter, r *http.Request) {
	apiresponse.JSON(w, r, http.StatusOK, koidomain.SearchSupportedCountries(r.URL.Query().Get("q")))
}
