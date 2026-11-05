package domain

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

func (p Prompt) GetBrandID() uuid.UUID { return p.BrandID }

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
	MutableRepository[Prompt]
	Lister[Prompt, PromptFilter]
	HasConfirmed(ctx context.Context, brandID uuid.UUID) (bool, error)
}

type PromptFilter struct {
	BrandFilter
	Query      string
	Confirmed  *bool
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

type CreatePromptParams struct {
	Text string
}

type UpdatePromptParams struct {
	Text     *string
	Inactive *bool
}

// NewPrompt builds a prompt written by hand, which is confirmed from the start.
func NewPrompt(brandID uuid.UUID, params CreatePromptParams) (*Prompt, error) {
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

	return prompt, nil
}

// Update applies the fields present in params. Provenance and confirmation never change here.
func (p *Prompt) Update(params UpdatePromptParams) error {
	if params.Text != nil {
		p.Text = *params.Text
	}

	if params.Inactive != nil {
		p.Inactive = *params.Inactive
	}

	p.Normalize()

	return p.Validate()
}

func (p *Prompt) Confirm() {
	p.Confirmed = true
}

type PromptCommander interface {
	Create(context.Context, uuid.UUID, CreatePromptParams) (*Prompt, error)
	Update(context.Context, uuid.UUID, UpdatePromptParams) (*Prompt, error)
	Delete(context.Context, uuid.UUID) error
	Confirm(context.Context, uuid.UUID) (*Prompt, error)
	ConfirmAll(context.Context, uuid.UUID) error
}

type PromptQuerier interface {
	BaseEntityQuerier[Prompt]
	Lister[Prompt, PromptFilter]
	ListWithVisibility(ctx context.Context, filter PromptFilter, page PageParams) (*Page[Prompt], error)
}

type promptCommander struct{ store Store }

func NewPromptCommander(store Store) PromptCommander { return &promptCommander{store: store} }

func (c *promptCommander) Create(ctx context.Context, brandID uuid.UUID, params CreatePromptParams) (*Prompt, error) {
	var prompt *Prompt

	err := withBrandLock(ctx, c.store, brandID, func(tx Store, brand *Brand) error {
		var err error
		prompt, err = NewPrompt(brandID, params)
		if err != nil {
			return err
		}

		if err := tx.BillingRepo().ReservePromptCapacity(ctx, brand.OrganizationID, 1); err != nil {
			return err
		}

		if err := tx.PromptRepo().Create(ctx, prompt); err != nil {
			return err
		}

		return tx.VisibilityRepo().InvalidateReview(ctx, brandID, ReviewPrompts)
	})
	if err != nil {
		return nil, err
	}

	return prompt, nil
}

func (c *promptCommander) Update(ctx context.Context, id uuid.UUID, params UpdatePromptParams) (*Prompt, error) {
	var prompt *Prompt

	err := withLockedChild(ctx, c.store, id, promptChildren, func(tx Store, brand *Brand, locked *Prompt) error {
		prompt = locked
		wasInactive := prompt.Inactive

		if err := prompt.Update(params); err != nil {
			return err
		}

		if wasInactive && !prompt.Inactive {
			if err := tx.BillingRepo().ReservePromptCapacity(ctx, brand.OrganizationID, 1); err != nil {
				return err
			}
		}

		if err := tx.PromptRepo().Update(ctx, prompt); err != nil {
			return err
		}

		if params.Text == nil && params.Inactive == nil {
			return nil
		}

		return tx.VisibilityRepo().InvalidateReview(ctx, prompt.BrandID, ReviewPrompts)
	})
	if err != nil {
		return nil, err
	}

	return prompt, nil
}

func (c *promptCommander) Delete(ctx context.Context, id uuid.UUID) error {
	return withLockedChild(ctx, c.store, id, promptChildren, func(tx Store, _ *Brand, prompt *Prompt) error {
		if err := tx.PromptRepo().Delete(ctx, id); err != nil {
			return err
		}

		return tx.VisibilityRepo().InvalidateReview(ctx, prompt.BrandID, ReviewPrompts)
	})
}

func (c *promptCommander) Confirm(ctx context.Context, id uuid.UUID) (*Prompt, error) {
	var prompt *Prompt

	err := withLockedChild(ctx, c.store, id, promptChildren, func(tx Store, _ *Brand, locked *Prompt) error {
		prompt = locked
		prompt.Confirm()

		return tx.PromptRepo().Update(ctx, prompt)
	})
	if err != nil {
		return nil, err
	}

	return prompt, nil
}

func (c *promptCommander) ConfirmAll(ctx context.Context, brandID uuid.UUID) error {
	return c.store.VisibilityRepo().FinishReview(ctx, brandID, ReviewPrompts)
}

func promptChildren(s Store) BaseEntityQuerier[Prompt] { return s.PromptRepo() }

type promptQuerier struct {
	brandChildQuerier[Prompt, PromptFilter]
}

func NewPromptQuerier(store Store) PromptQuerier {
	return &promptQuerier{
		brandChildQuerier: brandChildQuerier[Prompt, PromptFilter]{
			store: store,
			repo: func(s Store) brandChildRepository[Prompt, PromptFilter] {
				return s.PromptRepo()
			},
		},
	}
}

// ListWithVisibility adds each prompt's report for the period and engines in the filter.
func (q *promptQuerier) ListWithVisibility(ctx context.Context, filter PromptFilter, page PageParams) (*Page[Prompt], error) {
	prompts, err := q.List(ctx, filter, page)
	if err != nil {
		return nil, err
	}

	ids := make([]uuid.UUID, 0, len(prompts.Items))
	for _, prompt := range prompts.Items {
		ids = append(ids, prompt.ID)
	}

	byPrompt, _, err := NewVisibilityAnalyticsQuerier(q.store).Aggregate(ctx, filter.BrandID, ids, filter.Days, filter.Engines)
	if err != nil {
		return nil, err
	}

	for i := range prompts.Items {
		prompts.Items[i].Visibility = byPrompt[prompts.Items[i].ID]
	}

	return prompts, nil
}
