package koidomain

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

type PromptSource string

const (
	PromptSourceManual  PromptSource = "manual"
	PromptSourceDrafted PromptSource = "drafted"
)

type Prompt struct {
	BaseEntity
	BrandID   uuid.UUID    `json:"brand_id" gorm:"type:uuid;not null;index"`
	Text      string       `json:"text" gorm:"not null"`
	Source    PromptSource `json:"source" gorm:"not null;type:varchar(16)"`
	Confirmed bool         `json:"confirmed" gorm:"not null;default:false"`
	Brand     *Brand       `json:"-" gorm:"foreignKey:BrandID;constraint:OnDelete:CASCADE"`
}

func (Prompt) TableName() string { return "prompts" }

func normalizeWhitespace(value string) string { return strings.Join(strings.Fields(value), " ") }

func (p *Prompt) Normalize() { p.Text = normalizeWhitespace(p.Text) }

func (p Prompt) Validate() error {
	var fields []FieldError
	if p.BrandID == uuid.Nil {
		fields = append(fields, FieldError{Field: "brand_id", Code: FieldInvalid})
	}

	if strings.TrimSpace(p.Text) == "" {
		fields = append(fields, FieldError{Field: "text", Code: FieldInvalid})
	}

	if p.Source != PromptSourceManual && p.Source != PromptSourceDrafted {
		fields = append(fields, FieldError{Field: "source", Code: FieldInvalid})
	}

	if p.Source == PromptSourceManual && !p.Confirmed {
		fields = append(fields, FieldError{Field: "confirmed", Code: FieldInvalid})
	}

	if len(fields) > 0 {
		return Validation(fields...)
	}

	return nil
}

type PromptRepository interface {
	BaseEntityRepository[Prompt, PromptFilter]
	List(context.Context, PromptFilter) (*Page[Prompt], error)
	Update(context.Context, PromptFilter, PromptUpdate) error
	Delete(context.Context, PromptFilter) error
}

type PromptFilter struct {
	ID        uuid.UUID
	BrandID   uuid.UUID
	Query     string
	Confirmed *bool
	Page      PageParams
}

type PromptUpdate struct {
	Text      *string
	Confirmed *bool
}

type PromptParams struct{ Text string }
type PromptPatch struct{ Text *string }

type PromptCommander interface {
	Create(context.Context, uuid.UUID, PromptParams) (*Prompt, error)
	Update(context.Context, uuid.UUID, uuid.UUID, PromptPatch) (*Prompt, error)
	Delete(context.Context, uuid.UUID, uuid.UUID) error
	Confirm(context.Context, uuid.UUID, uuid.UUID) (*Prompt, error)
	ConfirmAll(context.Context, uuid.UUID) error
}

type PromptQuerier interface {
	List(context.Context, uuid.UUID, PageParams, string, *bool) (*Page[Prompt], error)
	Get(context.Context, uuid.UUID, uuid.UUID) (*Prompt, error)
}

type promptService struct{ store Store }

func NewPromptCommander(store Store) PromptCommander { return &promptService{store: store} }

func NewPromptQuerier(store Store) PromptQuerier { return &promptService{store: store} }

func ownedBrand(ctx context.Context, store Store, brandID uuid.UUID) error {
	_, err := store.BrandRepo().Get(ctx, brandID)

	return err
}

func (s *promptService) Create(ctx context.Context, brandID uuid.UUID, params PromptParams) (*Prompt, error) {
	if err := ownedBrand(ctx, s.store, brandID); err != nil {
		return nil, err
	}

	prompt := &Prompt{
		BrandID:   brandID,
		Text:      params.Text,
		Source:    PromptSourceManual,
		Confirmed: true,
	}
	prompt.Normalize()
	if err := prompt.Validate(); err != nil {
		return nil, err
	}

	if err := s.store.PromptRepo().Create(ctx, prompt); err != nil {
		return nil, err
	}

	return prompt, nil
}

func (s *promptService) Update(ctx context.Context, brandID, promptID uuid.UUID, patch PromptPatch) (*Prompt, error) {
	if err := ownedBrand(ctx, s.store, brandID); err != nil {
		return nil, err
	}

	filter := PromptFilter{ID: promptID, BrandID: brandID}
	prompt, err := s.store.PromptRepo().Get(ctx, filter)
	if err != nil {
		return nil, err
	}

	if patch.Text != nil {
		prompt.Text = *patch.Text
	}

	prompt.Normalize()
	if err := prompt.Validate(); err != nil {
		return nil, err
	}

	if err := s.store.PromptRepo().Update(ctx, filter, PromptUpdate{Text: &prompt.Text}); err != nil {
		return nil, err
	}

	return prompt, nil
}

func (s *promptService) Delete(ctx context.Context, brandID, promptID uuid.UUID) error {
	if err := ownedBrand(ctx, s.store, brandID); err != nil {
		return err
	}

	return s.store.PromptRepo().Delete(ctx, PromptFilter{ID: promptID, BrandID: brandID})
}

func (s *promptService) Confirm(ctx context.Context, brandID, promptID uuid.UUID) (*Prompt, error) {
	if err := ownedBrand(ctx, s.store, brandID); err != nil {
		return nil, err
	}

	filter := PromptFilter{ID: promptID, BrandID: brandID}
	prompt, err := s.store.PromptRepo().Get(ctx, filter)
	if err != nil {
		return nil, err
	}

	confirmed := true
	if err := s.store.PromptRepo().Update(ctx, filter, PromptUpdate{Confirmed: &confirmed}); err != nil {
		return nil, err
	}

	prompt.Confirmed = true

	return prompt, nil
}

func (s *promptService) ConfirmAll(ctx context.Context, brandID uuid.UUID) error {
	return s.store.VisibilityRepo().FinishReview(ctx, brandID, "prompts")
}

func (s *promptService) List(ctx context.Context, brandID uuid.UUID, page PageParams, query string, confirmed *bool) (*Page[Prompt], error) {
	if err := ownedBrand(ctx, s.store, brandID); err != nil {
		return nil, err
	}

	return s.store.PromptRepo().List(ctx, PromptFilter{
		BrandID:   brandID,
		Query:     query,
		Confirmed: confirmed,
		Page:      page,
	})
}

func (s *promptService) Get(ctx context.Context, brandID, promptID uuid.UUID) (*Prompt, error) {
	if err := ownedBrand(ctx, s.store, brandID); err != nil {
		return nil, err
	}

	return s.store.PromptRepo().Get(ctx, PromptFilter{ID: promptID, BrandID: brandID})
}
