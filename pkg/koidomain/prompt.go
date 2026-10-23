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
	BrandID    uuid.UUID         `json:"brand_id" gorm:"type:uuid;not null;index;uniqueIndex:idx_prompts_brand_text,priority:1"`
	Text       string            `json:"text" gorm:"not null;uniqueIndex:idx_prompts_brand_text,priority:2,expression:lower(text)"`
	Source     PromptSource      `json:"source" gorm:"not null;type:varchar(16)"`
	Confirmed  bool              `json:"confirmed" gorm:"not null;default:false"`
	Inactive   bool              `json:"inactive" gorm:"not null;default:false"`
	Visibility *VisibilityReport `json:"visibility,omitempty" gorm:"-"`
	Brand      *Brand            `json:"-" gorm:"foreignKey:BrandID;constraint:OnDelete:CASCADE"`
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
	BaseEntityRepository[Prompt]
	List(context.Context, PromptFilter) (*Page[Prompt], error)
	Update(context.Context, uuid.UUID, PromptUpdate) error
	Delete(context.Context, uuid.UUID) error
}

type PromptFilter struct {
	ID         uuid.UUID
	BrandID    uuid.UUID
	Query      string
	Confirmed  *bool
	Page       PageParams
	Sort       PromptSort
	Descending bool
	Days       int
	Engines    []EngineID
}

type PromptSort string

const (
	PromptSortOriginal PromptSort = "original"
	PromptSortText     PromptSort = "prompt"
	PromptSortRate     PromptSort = "rate"
	PromptSortChange   PromptSort = "change"
)

type PromptListParams struct {
	Page       PageParams
	Query      string
	Confirmed  *bool
	Sort       PromptSort
	Descending bool
	Days       int
	Engines    []EngineID
}

type PromptUpdate struct {
	Text      *string
	Confirmed *bool
	Inactive  *bool
}

type PromptParams struct{ Text string }
type PromptPatch struct {
	Text     *string
	Inactive *bool
}

type PromptCommander interface {
	Create(context.Context, uuid.UUID, PromptParams) (*Prompt, error)
	Update(context.Context, uuid.UUID, PromptPatch) (*Prompt, error)
	Delete(context.Context, uuid.UUID) error
	Confirm(context.Context, uuid.UUID) (*Prompt, error)
	ConfirmAll(context.Context, uuid.UUID) error
}

type PromptQuerier interface {
	List(context.Context, uuid.UUID, PromptListParams) (*Page[Prompt], error)
	Get(context.Context, uuid.UUID) (*Prompt, error)
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

func (s *promptService) Update(ctx context.Context, promptID uuid.UUID, patch PromptPatch) (*Prompt, error) {
	prompt, err := s.store.PromptRepo().Get(ctx, promptID)
	if err != nil {
		return nil, err
	}

	if patch.Text != nil {
		prompt.Text = *patch.Text
	}
	if patch.Inactive != nil {
		prompt.Inactive = *patch.Inactive
	}

	prompt.Normalize()
	if err := prompt.Validate(); err != nil {
		return nil, err
	}

	text := patch.Text
	if text != nil {
		text = &prompt.Text
	}
	if err := s.store.PromptRepo().Update(ctx, promptID, PromptUpdate{Text: text, Inactive: patch.Inactive}); err != nil {
		return nil, err
	}

	return prompt, nil
}

func (s *promptService) Delete(ctx context.Context, promptID uuid.UUID) error {
	return s.store.PromptRepo().Delete(ctx, promptID)
}

func (s *promptService) Confirm(ctx context.Context, promptID uuid.UUID) (*Prompt, error) {
	prompt, err := s.store.PromptRepo().Get(ctx, promptID)
	if err != nil {
		return nil, err
	}

	confirmed := true
	if err := s.store.PromptRepo().Update(ctx, promptID, PromptUpdate{Confirmed: &confirmed}); err != nil {
		return nil, err
	}

	prompt.Confirmed = true

	return prompt, nil
}

func (s *promptService) ConfirmAll(ctx context.Context, brandID uuid.UUID) error {
	return s.store.VisibilityRepo().FinishReview(ctx, brandID, "prompts")
}

func (s *promptService) List(ctx context.Context, brandID uuid.UUID, params PromptListParams) (*Page[Prompt], error) {
	if err := ownedBrand(ctx, s.store, brandID); err != nil {
		return nil, err
	}

	return s.store.PromptRepo().List(ctx, PromptFilter{
		BrandID:    brandID,
		Query:      params.Query,
		Confirmed:  params.Confirmed,
		Page:       params.Page,
		Sort:       params.Sort,
		Descending: params.Descending,
		Days:       params.Days,
		Engines:    params.Engines,
	})
}

func (s *promptService) Get(ctx context.Context, promptID uuid.UUID) (*Prompt, error) {
	return s.store.PromptRepo().Get(ctx, promptID)
}
