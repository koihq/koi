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
	BrandID   uuid.UUID        `json:"brand_id" gorm:"type:uuid;not null;index"`
	Name      string           `json:"name" gorm:"not null"`
	Domain    string           `json:"domain" gorm:"not null"`
	Aliases   []string         `json:"aliases" gorm:"serializer:json;type:jsonb;not null"`
	Source    CompetitorSource `json:"source" gorm:"not null;type:varchar(16)"`
	Confirmed bool             `json:"confirmed" gorm:"not null;default:false"`
	Brand     *Brand           `json:"-" gorm:"foreignKey:BrandID;constraint:OnDelete:CASCADE"`
}

func (Competitor) TableName() string { return "competitors" }

func (c *Competitor) Normalize() {
	c.Name = normalizeWhitespace(c.Name)
	c.Domain = strings.ToLower(strings.TrimSpace(c.Domain))
	aliases := make([]string, 0, len(c.Aliases))
	seen := make(map[string]struct{}, len(c.Aliases))
	for _, raw := range c.Aliases {
		alias := normalizeWhitespace(raw)
		key := strings.ToLower(alias)
		if alias == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		aliases = append(aliases, alias)
	}
	c.Aliases = aliases
}

func (c Competitor) Validate() error {
	var fields []FieldError
	if c.BrandID == uuid.Nil {
		fields = append(fields, FieldError{Field: "brand_id", Code: FieldInvalid})
	}
	if strings.TrimSpace(c.Name) == "" {
		fields = append(fields, FieldError{Field: "name", Code: FieldInvalid})
	}
	if !validHostname(c.Domain) {
		fields = append(fields, FieldError{Field: "domain", Code: FieldInvalid})
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
	Create(context.Context, *Competitor) error
	Get(context.Context, CompetitorFilter) (*Competitor, error)
	List(context.Context, CompetitorFilter) (*Page[Competitor], error)
	Update(context.Context, CompetitorFilter, CompetitorUpdate) error
	Delete(context.Context, CompetitorFilter) error
}

type CompetitorFilter struct {
	ID        uuid.UUID
	BrandID   uuid.UUID
	Query     string
	Confirmed *bool
	Page      PageParams
}

type CompetitorUpdate struct {
	Name      *string
	Domain    *string
	Aliases   *[]string
	Confirmed *bool
}

type CompetitorParams struct {
	Name    string
	Domain  string
	Aliases []string
}

type CompetitorPatch struct {
	Name    *string
	Domain  *string
	Aliases *[]string
}

type CompetitorCommander interface {
	Create(context.Context, uuid.UUID, uuid.UUID, CompetitorParams) (*Competitor, error)
	Update(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, CompetitorPatch) (*Competitor, error)
	Delete(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error
	Confirm(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*Competitor, error)
	ConfirmAll(context.Context, uuid.UUID, uuid.UUID) error
}

type CompetitorQuerier interface {
	List(context.Context, uuid.UUID, uuid.UUID, PageParams, string, *bool) (*Page[Competitor], error)
	Get(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*Competitor, error)
}

type competitorService struct{ store Store }

func NewCompetitorCommander(store Store) CompetitorCommander { return &competitorService{store: store} }
func NewCompetitorQuerier(store Store) CompetitorQuerier     { return &competitorService{store: store} }

func (s *competitorService) Create(ctx context.Context, userID, brandID uuid.UUID, params CompetitorParams) (*Competitor, error) {
	if err := ownedBrand(ctx, s.store, userID, brandID); err != nil {
		return nil, err
	}
	competitor := &Competitor{
		BrandID: brandID, Name: params.Name, Domain: params.Domain, Aliases: params.Aliases,
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

func (s *competitorService) Update(ctx context.Context, userID, brandID, competitorID uuid.UUID, patch CompetitorPatch) (*Competitor, error) {
	if err := ownedBrand(ctx, s.store, userID, brandID); err != nil {
		return nil, err
	}
	filter := CompetitorFilter{ID: competitorID, BrandID: brandID}
	competitor, err := s.store.CompetitorRepo().Get(ctx, filter)
	if err != nil {
		return nil, err
	}
	if patch.Name != nil {
		competitor.Name = *patch.Name
	}
	if patch.Domain != nil {
		competitor.Domain = *patch.Domain
	}
	if patch.Aliases != nil {
		competitor.Aliases = *patch.Aliases
	}
	competitor.Normalize()
	if err := competitor.Validate(); err != nil {
		return nil, err
	}
	if err := s.store.CompetitorRepo().Update(ctx, filter, CompetitorUpdate{
		Name: &competitor.Name, Domain: &competitor.Domain, Aliases: &competitor.Aliases,
	}); err != nil {
		return nil, err
	}
	return competitor, nil
}

func (s *competitorService) Delete(ctx context.Context, userID, brandID, competitorID uuid.UUID) error {
	if err := ownedBrand(ctx, s.store, userID, brandID); err != nil {
		return err
	}
	return s.store.CompetitorRepo().Delete(ctx, CompetitorFilter{ID: competitorID, BrandID: brandID})
}

func (s *competitorService) Confirm(ctx context.Context, userID, brandID, competitorID uuid.UUID) (*Competitor, error) {
	if err := ownedBrand(ctx, s.store, userID, brandID); err != nil {
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

func (s *competitorService) ConfirmAll(ctx context.Context, userID, brandID uuid.UUID) error {
	if err := ownedBrand(ctx, s.store, userID, brandID); err != nil {
		return err
	}
	unconfirmed, confirmed := false, true
	return s.store.CompetitorRepo().Update(ctx,
		CompetitorFilter{BrandID: brandID, Confirmed: &unconfirmed},
		CompetitorUpdate{Confirmed: &confirmed},
	)
}

func (s *competitorService) List(ctx context.Context, userID, brandID uuid.UUID, page PageParams, query string, confirmed *bool) (*Page[Competitor], error) {
	if err := ownedBrand(ctx, s.store, userID, brandID); err != nil {
		return nil, err
	}
	return s.store.CompetitorRepo().List(ctx, CompetitorFilter{BrandID: brandID, Query: query, Confirmed: confirmed, Page: page})
}

func (s *competitorService) Get(ctx context.Context, userID, brandID, competitorID uuid.UUID) (*Competitor, error) {
	if err := ownedBrand(ctx, s.store, userID, brandID); err != nil {
		return nil, err
	}
	return s.store.CompetitorRepo().Get(ctx, CompetitorFilter{ID: competitorID, BrandID: brandID})
}
