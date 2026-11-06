package api

import (
	"context"
	"net/http"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/koihq/koi/pkg/api/middlewares"
	"github.com/koihq/koi/pkg/api/response"
	"github.com/koihq/koi/pkg/domain"
)

type CreateBrandReq struct {
	Name        string `json:"name"`
	Domain      string `json:"domain"`
	GitHubRepo  string `json:"githubRepo"`
	Country     string `json:"country"`
	Description string `json:"description"`
}

type UpdateBrandReq struct {
	Name        *string `json:"name"`
	Domain      *string `json:"domain"`
	GitHubRepo  *string `json:"githubRepo"`
	Country     *string `json:"country"`
	Description *string `json:"description"`
}

type BrandRes struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organizationId"`
	Name           string    `json:"name"`
	Domain         string    `json:"domain"`
	GitHubRepo     string    `json:"githubRepo"`
	Country        string    `json:"country"`
	Description    string    `json:"description"`
}

func BrandToRes(v *domain.Brand) BrandRes {
	return BrandRes{
		ID:             v.ID,
		OrganizationID: v.OrganizationID,
		Name:           v.Name,
		Domain:         v.Domain,
		GitHubRepo:     v.GitHubRepo,
		Country:        v.Country,
		Description:    v.Description,
	}
}

type BrandHandler struct {
	querier   domain.BrandQuerier
	commander domain.BrandCommander
	sess      *scs.SessionManager
	billing   func(http.Handler) http.Handler
	looker    domain.BrandLooker
}

type BrandLookupRes struct {
	Found       bool   `json:"found"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source"`
}

// NewBrandHandler guards brand changes with billing. Self-hosted instances pass passThrough.
func NewBrandHandler(q domain.BrandQuerier, c domain.BrandCommander, sess *scs.SessionManager, billing func(http.Handler) http.Handler) *BrandHandler {
	return &BrandHandler{querier: q, commander: c, sess: sess, billing: billing}
}

func (h *BrandHandler) Routes() func(chi.Router) {
	return func(r chi.Router) {
		r.With(middlewares.DecodeBody[CreateBrandReq]()).Post("/", Create(h.create, BrandToRes))
		r.Get("/", List(brandListFilter, h.querier.List, BrandToRes))
		r.Get("/countries", h.Countries)
		r.Get("/lookup", h.Lookup)
		r.Get("/{id}", Get(h.querier.Get, BrandToRes))
		r.With(h.billing).Post("/{id}/activate", Action(h.activate, BrandToRes))
		r.With(h.billing, middlewares.DecodeBody[UpdateBrandReq]()).Patch("/{id}", Update(h.update, BrandToRes))
	}
}

func brandListFilter(r *http.Request) (domain.BrandListFilter, error) {
	query, err := optionalStringQuery(r, "q")
	if err != nil {
		return domain.BrandListFilter{}, err
	}

	return domain.BrandListFilter{Query: query}, nil
}

// activate stores a UI preference. It never changes the scope of another request.
func (h *BrandHandler) activate(ctx context.Context, id uuid.UUID) (*domain.Brand, error) {
	brand, err := h.querier.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	h.sess.Put(ctx, "active_brand_id", brand.ID.String())

	return brand, nil
}

func (h *BrandHandler) create(ctx context.Context, req CreateBrandReq) (*domain.Brand, error) {
	return h.commander.Create(ctx, domain.CreateBrandParams{
		Name:        req.Name,
		Domain:      req.Domain,
		GitHubRepo:  req.GitHubRepo,
		Country:     req.Country,
		Description: req.Description,
	})
}

func (h *BrandHandler) update(ctx context.Context, id uuid.UUID, req UpdateBrandReq) (*domain.Brand, error) {
	return h.commander.Update(ctx, id, domain.UpdateBrandParams{
		Name:        req.Name,
		Domain:      req.Domain,
		GitHubRepo:  req.GitHubRepo,
		Country:     req.Country,
		Description: req.Description,
	})
}

// Lookup only suggests values for the setup form, so it stays open before any subscription.
func (h *BrandHandler) Lookup(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	lookup, err := domain.LookupBrand(r.Context(), h.looker, query.Get("domain"), query.Get("githubRepo"))
	if err != nil {
		response.WriteDomainError(w, r, err)

		return
	}

	response.JSON(w, r, http.StatusOK, BrandLookupRes{
		Found:       lookup.Found,
		Name:        lookup.Name,
		Description: lookup.Description,
		Source:      lookup.Source,
	})
}

func (h *BrandHandler) Countries(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, r, http.StatusOK, domain.SearchSupportedCountries(r.URL.Query().Get("q")))
}
