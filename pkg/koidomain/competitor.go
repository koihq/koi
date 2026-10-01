package koidomain

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

type CompetitorSource string

const (
	CompetitorSourceManual   CompetitorSource = "manual"
	CompetitorSourceDrafted  CompetitorSource = "drafted"
	CompetitorSourceObserved CompetitorSource = "observed"
)

type Competitor struct {
	BaseEntity
	BrandID    uuid.UUID        `json:"brand_id" gorm:"type:uuid;not null"`
	Name       string           `json:"name" gorm:"not null"`
	Domain     string           `json:"domain" gorm:"not null"`
	GitHubRepo string           `json:"github_repo" gorm:"column:github_repo;not null;default:''"`
	Source     CompetitorSource `json:"source" gorm:"not null;type:varchar(16)"`
	Confirmed  bool             `json:"confirmed" gorm:"not null;default:false"`
	Brand      *Brand           `json:"-" gorm:"foreignKey:BrandID;constraint:OnDelete:CASCADE"`
}

func (Competitor) TableName() string { return "competitors" }

func (c *Competitor) Normalize() {
	c.Name = normalizeWhitespace(c.Name)
	c.Domain = strings.ToLower(strings.TrimSpace(c.Domain))
	c.GitHubRepo = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(c.GitHubRepo)), "/")
}

func (c Competitor) Validate() error {
	var fields []FieldError
	if c.BrandID == uuid.Nil {
		fields = append(fields, FieldError{Field: "brand_id", Code: FieldInvalid})
	}
	if strings.TrimSpace(c.Name) == "" {
		fields = append(fields, FieldError{Field: "name", Code: FieldInvalid})
	}
	if c.Domain == "" && c.GitHubRepo == "" {
		fields = append(fields, FieldError{Field: "domain", Code: FieldInvalid}, FieldError{Field: "github_repo", Code: FieldInvalid})
	} else if c.Domain != "" && !validHostname(c.Domain) {
		fields = append(fields, FieldError{Field: "domain", Code: FieldInvalid})
	}
	if c.GitHubRepo != "" && !validGitHubRepo(c.GitHubRepo) {
		fields = append(fields, FieldError{Field: "github_repo", Code: FieldInvalid})
	}
	if c.Source != CompetitorSourceManual && c.Source != CompetitorSourceDrafted && c.Source != CompetitorSourceObserved {
		fields = append(fields, FieldError{Field: "source", Code: FieldInvalid})
	}
	if c.Source == CompetitorSourceManual && !c.Confirmed {
		fields = append(fields, FieldError{Field: "confirmed", Code: FieldInvalid})
	}
	if len(fields) > 0 {
		return Validation(fields...)
	}
	return nil
}

type CompetitorRepository interface {
	BaseEntityRepository[Competitor, CompetitorFilter]
	List(context.Context, CompetitorFilter) (*Page[Competitor], error)
	Update(context.Context, CompetitorFilter, CompetitorUpdate) error
	Delete(context.Context, CompetitorFilter) error
}

type CompetitorFilter struct {
	ID        uuid.UUID
	BrandID   uuid.UUID
	Domain    string
	Query     string
	Confirmed *bool
	Page      PageParams
}

type CompetitorUpdate struct {
	Confirmed *bool
}

type CompetitorParams struct {
	Name       string
	Domain     string
	GitHubRepo string
}

type CompanySearchResult struct {
	Name   string `json:"name"`
	Domain string `json:"domain"`
}

type CompanySearcher interface {
	Search(context.Context, string) ([]CompanySearchResult, error)
}

type CompetitorCommander interface {
	Create(context.Context, uuid.UUID, CompetitorParams) (*Competitor, error)
	Delete(context.Context, uuid.UUID, uuid.UUID) error
	Confirm(context.Context, uuid.UUID, uuid.UUID) (*Competitor, error)
	ConfirmAll(context.Context, uuid.UUID) error
}

type CompetitorQuerier interface {
	List(context.Context, uuid.UUID, PageParams, string, *bool) (*Page[Competitor], error)
	Get(context.Context, uuid.UUID, uuid.UUID) (*Competitor, error)
	Search(context.Context, uuid.UUID, string) ([]CompanySearchResult, error)
}

type competitorService struct {
	store    Store
	searcher CompanySearcher
}

func NewCompetitorCommander(store Store) CompetitorCommander { return &competitorService{store: store} }
func NewCompetitorQuerier(store Store, searcher CompanySearcher) CompetitorQuerier {
	return &competitorService{store: store, searcher: searcher}
}

func (s *competitorService) Create(ctx context.Context, brandID uuid.UUID, params CompetitorParams) (*Competitor, error) {
	if err := ownedBrand(ctx, s.store, brandID); err != nil {
		return nil, err
	}
	competitor := &Competitor{
		BrandID: brandID, Name: params.Name, Domain: params.Domain, GitHubRepo: params.GitHubRepo,
		Source: CompetitorSourceManual, Confirmed: true,
	}
	competitor.Normalize()
	if err := competitor.Validate(); err != nil {
		return nil, err
	}
	if err := s.store.CompetitorRepo().Create(ctx, competitor); err != nil {
		return nil, err
	}
	return competitor, nil
}

func (s *competitorService) Delete(ctx context.Context, brandID, competitorID uuid.UUID) error {
	if err := ownedBrand(ctx, s.store, brandID); err != nil {
		return err
	}
	return s.store.CompetitorRepo().Delete(ctx, CompetitorFilter{ID: competitorID, BrandID: brandID})
}

func (s *competitorService) Confirm(ctx context.Context, brandID, competitorID uuid.UUID) (*Competitor, error) {
	if err := ownedBrand(ctx, s.store, brandID); err != nil {
		return nil, err
	}
	filter := CompetitorFilter{ID: competitorID, BrandID: brandID}
	competitor, err := s.store.CompetitorRepo().Get(ctx, filter)
	if err != nil {
		return nil, err
	}
	confirmed := true
	if err := s.store.CompetitorRepo().Update(ctx, filter, CompetitorUpdate{Confirmed: &confirmed}); err != nil {
		return nil, err
	}
	competitor.Confirmed = true
	return competitor, nil
}

func (s *competitorService) ConfirmAll(ctx context.Context, brandID uuid.UUID) error {
	if err := ownedBrand(ctx, s.store, brandID); err != nil {
		return err
	}
	unconfirmed, confirmed := false, true
	return s.store.CompetitorRepo().Update(ctx,
		CompetitorFilter{BrandID: brandID, Confirmed: &unconfirmed},
		CompetitorUpdate{Confirmed: &confirmed},
	)
}

func (s *competitorService) List(ctx context.Context, brandID uuid.UUID, page PageParams, query string, confirmed *bool) (*Page[Competitor], error) {
	if err := ownedBrand(ctx, s.store, brandID); err != nil {
		return nil, err
	}
	return s.store.CompetitorRepo().List(ctx, CompetitorFilter{BrandID: brandID, Query: query, Confirmed: confirmed, Page: page})
}

func (s *competitorService) Get(ctx context.Context, brandID, competitorID uuid.UUID) (*Competitor, error) {
	if err := ownedBrand(ctx, s.store, brandID); err != nil {
		return nil, err
	}
	return s.store.CompetitorRepo().Get(ctx, CompetitorFilter{ID: competitorID, BrandID: brandID})
}

func (s *competitorService) Search(ctx context.Context, brandID uuid.UUID, query string) ([]CompanySearchResult, error) {
	if err := ownedBrand(ctx, s.store, brandID); err != nil {
		return nil, err
	}
	if len([]rune(strings.TrimSpace(query))) < 3 {
		return nil, Validation(FieldError{Field: "q", Code: FieldMinLength})
	}
	if s.searcher == nil {
		return nil, ErrFeatureUnavailable
	}
	return s.searcher.Search(ctx, query)
}
