package domain

import (
	"context"
	"slices"
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
	BrandID        uuid.UUID        `json:"brand_id" gorm:"type:uuid;not null;uniqueIndex:idx_competitors_brand_domain,priority:1,where:domain <> '';uniqueIndex:idx_competitors_brand_github_repo,priority:1,where:github_repo <> ''"`
	Name           string           `json:"name" gorm:"not null"`
	Domain         string           `json:"domain" gorm:"not null;uniqueIndex:idx_competitors_brand_domain,priority:2"`
	GitHubRepo     string           `json:"github_repo" gorm:"column:github_repo;not null;default:'';uniqueIndex:idx_competitors_brand_github_repo,priority:2"`
	Aliases        []string         `json:"aliases" gorm:"type:jsonb;serializer:json;not null;default:'[]'"`
	AdditionalURLs []string         `json:"additional_urls" gorm:"type:jsonb;serializer:json;not null;default:'[]'"`
	Source         CompetitorSource `json:"source" gorm:"not null;type:varchar(16)"`
	Confirmed      bool             `json:"confirmed" gorm:"not null;default:false"`
	Brand          *Brand           `json:"-" gorm:"foreignKey:BrandID;constraint:OnDelete:CASCADE"`
}

func (Competitor) TableName() string { return "competitors" }

func (c Competitor) GetBrandID() uuid.UUID { return c.BrandID }

func (c *Competitor) Normalize() {
	c.Name = normalizeWhitespace(c.Name)
	c.Domain = strings.ToLower(strings.TrimSpace(c.Domain))
	c.GitHubRepo = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(c.GitHubRepo)), "/")
	c.Aliases = normalizeNames(c.Aliases)
	c.AdditionalURLs = normalizeIdentityURLs(c.AdditionalURLs)
}

func (c Competitor) Validate() error {
	var fields []FieldError
	if c.BrandID == uuid.Nil {
		fields = append(fields, FieldError{Field: "brand_id", Code: FieldInvalid})
	}

	if strings.TrimSpace(c.Name) == "" {
		fields = append(fields, FieldError{Field: "name", Code: FieldInvalid})
	}

	if c.Domain == "" && c.GitHubRepo == "" && (c.Source != CompetitorSourceObserved || c.Confirmed) {
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
	if len(c.Name) > 255 {
		fields = append(fields, FieldError{Field: "name", Code: FieldInvalid})
	}
	if len(c.Aliases) > 20 {
		fields = append(fields, FieldError{Field: "aliases", Code: FieldInvalid})
	}
	if len(c.AdditionalURLs) > 20 {
		fields = append(fields, FieldError{Field: "additional_urls", Code: FieldInvalid})
	}
	for _, alias := range c.Aliases {
		if len(alias) > 255 {
			fields = append(fields, FieldError{Field: "aliases", Code: FieldInvalid})
			break
		}
	}
	for _, value := range c.AdditionalURLs {
		if _, ok := NormalizeIdentityURL(value); !ok {
			fields = append(fields, FieldError{Field: "additional_urls", Code: FieldInvalid})
			break
		}
	}

	if len(fields) > 0 {
		return Validation(fields...)
	}

	return nil
}

type CompetitorRepository interface {
	BaseEntityRepository[Competitor]
	MutableRepository[Competitor]
	Lister[Competitor, CompetitorFilter]
	// ForBrand returns every competitor of a brand, for checks across the whole set.
	ForBrand(ctx context.Context, brandID uuid.UUID) ([]Competitor, error)
	Observations(ctx context.Context, brandID uuid.UUID) ([]CompetitorObservation, error)
	SaveObservations(ctx context.Context, observations []CompetitorObservation) error
}

type CompetitorFilter struct {
	BrandFilter
	Domain    string
	Query     string
	Confirmed *bool
}

type CreateCompetitorParams struct {
	Name           string
	Domain         string
	GitHubRepo     string
	Aliases        []string
	AdditionalURLs []string
}

type UpdateCompetitorParams struct {
	Name           *string
	Domain         *string
	GitHubRepo     *string
	Aliases        *[]string
	AdditionalURLs *[]string
}

// NewCompetitor builds a competitor added by hand, which is confirmed from the start.
func NewCompetitor(brandID uuid.UUID, params CreateCompetitorParams) (*Competitor, error) {
	competitor := &Competitor{
		BrandID:        brandID,
		Name:           params.Name,
		Domain:         params.Domain,
		GitHubRepo:     params.GitHubRepo,
		Aliases:        params.Aliases,
		AdditionalURLs: params.AdditionalURLs,
		Source:         CompetitorSourceManual,
		Confirmed:      true,
	}

	competitor.Normalize()
	if err := competitor.Validate(); err != nil {
		return nil, err
	}

	return competitor, nil
}

// Update applies the fields present in params. Provenance and confirmation never change here.
func (c *Competitor) Update(params UpdateCompetitorParams) error {
	if params.Name != nil {
		c.Name = *params.Name
	}

	if params.Domain != nil {
		c.Domain = *params.Domain
	}

	if params.GitHubRepo != nil {
		c.GitHubRepo = *params.GitHubRepo
	}

	if params.Aliases != nil {
		c.Aliases = *params.Aliases
	}

	if params.AdditionalURLs != nil {
		c.AdditionalURLs = *params.AdditionalURLs
	}

	c.Normalize()

	return c.Validate()
}

// Confirm fails for an observed competitor that still has no domain or repository.
func (c *Competitor) Confirm() error {
	c.Confirmed = true
	c.Normalize()

	return c.Validate()
}

// ConflictsWith reports the first field whose URL identifies another competitor of the brand.
func (c Competitor) ConflictsWith(others []Competitor) error {
	for _, other := range others {
		if other.ID == c.ID || !c.SharesIdentityURL(other) {
			continue
		}

		field := "additional_urls"
		if c.Domain != "" && (Competitor{Domain: c.Domain}).SharesIdentityURL(other) {
			field = "domain"
		} else if c.GitHubRepo != "" && (Competitor{GitHubRepo: c.GitHubRepo}).SharesIdentityURL(other) {
			field = "github_repo"
		}

		return Validation(FieldError{Field: field, Code: FieldTaken})
	}

	return nil
}

// Dismissals returns the observations to save so that deleting this observed competitor sticks:
// later answers that name it must not bring it back as a suggestion.
func (c Competitor) Dismissals(observations []CompetitorObservation) []CompetitorObservation {
	var dismissed []CompetitorObservation

	for _, state := range observations {
		observation := AppearanceObservation{Name: state.Identity.Name, URLs: state.Identity.IdentityURLs()}
		index, _ := ResolveObservation(observation, []Competitor{c})
		if state.Identity.ID != c.ID && index < 0 {
			continue
		}

		identity := c
		identity.Aliases = slices.Concat(identity.Aliases, []string{state.Identity.Name})
		identity.Normalize()
		state.Identity, state.Dismissed = identity, true
		dismissed = append(dismissed, state)
	}

	if len(dismissed) > 0 {
		return dismissed
	}

	return []CompetitorObservation{
		{
			BrandID:   c.BrandID,
			Key:       AppearanceNameKey(c.Name),
			Dismissed: true,
			Identity:  c,
		},
	}
}

type CompanySearchResult struct {
	Name   string `json:"name"`
	Domain string `json:"domain"`
}

type CompanySearcher interface {
	Search(context.Context, string) ([]CompanySearchResult, error)
}

type CompetitorCommander interface {
	Create(context.Context, uuid.UUID, CreateCompetitorParams) (*Competitor, error)
	Update(context.Context, uuid.UUID, UpdateCompetitorParams) (*Competitor, error)
	Delete(context.Context, uuid.UUID) error
	Confirm(context.Context, uuid.UUID) (*Competitor, error)
	ConfirmAll(context.Context, uuid.UUID) error
}

type CompetitorQuerier interface {
	BaseEntityQuerier[Competitor]
	Lister[Competitor, CompetitorFilter]
	Search(context.Context, uuid.UUID, string) ([]CompanySearchResult, error)
}

type competitorCommander struct{ store Store }

func NewCompetitorCommander(store Store) CompetitorCommander {
	return &competitorCommander{store: store}
}

func competitorChildren(s Store) BaseEntityQuerier[Competitor] { return s.CompetitorRepo() }

// requireUniqueCompetitorURLs runs under the brand lock, which keeps the set stable.
func requireUniqueCompetitorURLs(ctx context.Context, tx Store, competitor *Competitor) error {
	others, err := tx.CompetitorRepo().ForBrand(ctx, competitor.BrandID)
	if err != nil {
		return err
	}

	return competitor.ConflictsWith(others)
}

func (c *competitorCommander) Create(ctx context.Context, brandID uuid.UUID, params CreateCompetitorParams) (*Competitor, error) {
	var competitor *Competitor

	err := withBrandLock(ctx, c.store, brandID, func(tx Store, _ *Brand) error {
		var err error
		competitor, err = NewCompetitor(brandID, params)
		if err != nil {
			return err
		}

		if err := requireUniqueCompetitorURLs(ctx, tx, competitor); err != nil {
			return err
		}

		if err := tx.CompetitorRepo().Create(ctx, competitor); err != nil {
			return err
		}

		return tx.VisibilityRepo().InvalidateReview(ctx, brandID, ReviewCompetitors)
	})
	if err != nil {
		return nil, err
	}

	return competitor, nil
}

// Update keeps the review: editing a competitor's identity does not change who is tracked.
func (c *competitorCommander) Update(ctx context.Context, id uuid.UUID, params UpdateCompetitorParams) (*Competitor, error) {
	var competitor *Competitor

	err := withLockedChild(ctx, c.store, id, competitorChildren, func(tx Store, _ *Brand, locked *Competitor) error {
		competitor = locked
		if params == (UpdateCompetitorParams{}) {
			return nil
		}

		if err := competitor.Update(params); err != nil {
			return err
		}

		if err := requireUniqueCompetitorURLs(ctx, tx, competitor); err != nil {
			return err
		}

		return tx.CompetitorRepo().Update(ctx, competitor)
	})
	if err != nil {
		return nil, err
	}

	return competitor, nil
}

func (c *competitorCommander) Delete(ctx context.Context, id uuid.UUID) error {
	return withLockedChild(ctx, c.store, id, competitorChildren, func(tx Store, _ *Brand, competitor *Competitor) error {
		observed := competitor.Source == CompetitorSourceObserved
		if observed {
			observations, err := tx.CompetitorRepo().Observations(ctx, competitor.BrandID)
			if err != nil {
				return err
			}

			if err := tx.CompetitorRepo().SaveObservations(ctx, competitor.Dismissals(observations)); err != nil {
				return err
			}
		}

		if err := tx.CompetitorRepo().Delete(ctx, id); err != nil {
			return err
		}

		// An unconfirmed suggestion was never part of the reviewed set.
		if observed && !competitor.Confirmed {
			return nil
		}

		return tx.VisibilityRepo().InvalidateReview(ctx, competitor.BrandID, ReviewCompetitors)
	})
}

func (c *competitorCommander) Confirm(ctx context.Context, id uuid.UUID) (*Competitor, error) {
	var competitor *Competitor

	err := withLockedChild(ctx, c.store, id, competitorChildren, func(tx Store, _ *Brand, locked *Competitor) error {
		competitor = locked

		if err := competitor.Confirm(); err != nil {
			return err
		}

		if err := requireUniqueCompetitorURLs(ctx, tx, competitor); err != nil {
			return err
		}

		return tx.CompetitorRepo().Update(ctx, competitor)
	})
	if err != nil {
		return nil, err
	}

	return competitor, nil
}

func (c *competitorCommander) ConfirmAll(ctx context.Context, brandID uuid.UUID) error {
	return c.store.VisibilityRepo().FinishReview(ctx, brandID, ReviewCompetitors)
}

type competitorQuerier struct {
	brandChildQuerier[Competitor, CompetitorFilter]
	searcher CompanySearcher
}

func NewCompetitorQuerier(store Store, searcher CompanySearcher) CompetitorQuerier {
	return &competitorQuerier{
		brandChildQuerier: brandChildQuerier[Competitor, CompetitorFilter]{
			store: store,
			repo: func(s Store) brandChildRepository[Competitor, CompetitorFilter] {
				return s.CompetitorRepo()
			},
		},
		searcher: searcher,
	}
}

func (q *competitorQuerier) Search(ctx context.Context, brandID uuid.UUID, query string) ([]CompanySearchResult, error) {
	if err := ownedBrand(ctx, q.store, brandID); err != nil {
		return nil, err
	}

	if len([]rune(strings.TrimSpace(query))) < 3 {
		return nil, Validation(FieldError{Field: "q", Code: FieldMinLength})
	}

	if q.searcher == nil {
		return nil, ErrFeatureUnavailable
	}

	return q.searcher.Search(ctx, query)
}
