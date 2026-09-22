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
	Create(context.Context, *Prompt) error
	ListByBrandID(context.Context, uuid.UUID) ([]Prompt, error)
	GetByIDAndBrandID(context.Context, uuid.UUID, uuid.UUID) (*Prompt, error)
	UpdateByIDAndBrandID(context.Context, uuid.UUID, *Prompt) error
	DeleteByIDAndBrandID(context.Context, uuid.UUID, uuid.UUID) error
	ConfirmByIDAndBrandID(context.Context, uuid.UUID, uuid.UUID) error
	ConfirmAllByBrandID(context.Context, uuid.UUID) error
}

type PromptParams struct{ Text string }
type PromptPatch struct{ Text *string }

type PromptCommander interface {
	Create(context.Context, uuid.UUID, uuid.UUID, PromptParams) (*Prompt, error)
	Update(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, PromptPatch) (*Prompt, error)
	Delete(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error
	Confirm(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*Prompt, error)
	ConfirmAll(context.Context, uuid.UUID, uuid.UUID) ([]Prompt, error)
}

type PromptQuerier interface {
	List(context.Context, uuid.UUID, uuid.UUID) ([]Prompt, error)
}

type promptService struct{ store Store }

func NewPromptCommander(store Store) PromptCommander { return &promptService{store: store} }
func NewPromptQuerier(store Store) PromptQuerier     { return &promptService{store: store} }

func ownedBrand(ctx context.Context, store Store, userID, brandID uuid.UUID) error {
	_, err := store.BrandRepo().GetByIDAndUserID(ctx, brandID, userID)
	return err
}

func (s *promptService) Create(ctx context.Context, userID, brandID uuid.UUID, params PromptParams) (*Prompt, error) {
	if err := ownedBrand(ctx, s.store, userID, brandID); err != nil {
		return nil, err
	}
	prompt := &Prompt{BrandID: brandID, Text: params.Text, Source: PromptSourceManual, Confirmed: true}
	prompt.Normalize()
	if err := prompt.Validate(); err != nil {
		return nil, err
	}
	if err := s.store.PromptRepo().Create(ctx, prompt); err != nil {
		return nil, err
	}
	return prompt, nil
}

func (s *promptService) Update(ctx context.Context, userID, brandID, promptID uuid.UUID, patch PromptPatch) (*Prompt, error) {
	if err := ownedBrand(ctx, s.store, userID, brandID); err != nil {
		return nil, err
	}
	prompt, err := s.store.PromptRepo().GetByIDAndBrandID(ctx, promptID, brandID)
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
	if err := s.store.PromptRepo().UpdateByIDAndBrandID(ctx, brandID, prompt); err != nil {
		return nil, err
	}
	return prompt, nil
}

func (s *promptService) Delete(ctx context.Context, userID, brandID, promptID uuid.UUID) error {
	if err := ownedBrand(ctx, s.store, userID, brandID); err != nil {
		return err
	}
	return s.store.PromptRepo().DeleteByIDAndBrandID(ctx, promptID, brandID)
}

func (s *promptService) Confirm(ctx context.Context, userID, brandID, promptID uuid.UUID) (*Prompt, error) {
	if err := ownedBrand(ctx, s.store, userID, brandID); err != nil {
		return nil, err
	}
	if err := s.store.PromptRepo().ConfirmByIDAndBrandID(ctx, promptID, brandID); err != nil {
		return nil, err
	}
	return s.store.PromptRepo().GetByIDAndBrandID(ctx, promptID, brandID)
}

func (s *promptService) ConfirmAll(ctx context.Context, userID, brandID uuid.UUID) ([]Prompt, error) {
	if err := ownedBrand(ctx, s.store, userID, brandID); err != nil {
		return nil, err
	}
	if err := s.store.PromptRepo().ConfirmAllByBrandID(ctx, brandID); err != nil {
		return nil, err
	}
	return s.store.PromptRepo().ListByBrandID(ctx, brandID)
}

func (s *promptService) List(ctx context.Context, userID, brandID uuid.UUID) ([]Prompt, error) {
	if err := ownedBrand(ctx, s.store, userID, brandID); err != nil {
		return nil, err
	}
	return s.store.PromptRepo().ListByBrandID(ctx, brandID)
}
