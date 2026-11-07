package domain

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

// maxPromptTextLen is high on purpose: it leaves room for detailed prompts and only stops abuse,
// since the text goes into paid calls.
const maxPromptTextLen = 4000

type PromptSource string

const (
	PromptSourceManual  PromptSource = "manual"
	PromptSourceDrafted PromptSource = "drafted"
)

type Prompt struct {
	BaseEntity
	BrandID    uuid.UUID         `json:"brand_id" gorm:"type:uuid;not null;index;uniqueIndex:idx_prompts_brand_text,priority:1"`
	Text       string            `json:"text" gorm:"not null;uniqueIndex:idx_prompts_brand_text,priority:2,expression:md5(lower(text))"`
	Source     PromptSource      `json:"source" gorm:"not null;type:varchar(16)"`
	Confirmed  bool              `json:"confirmed" gorm:"not null;default:false"`
	Inactive   bool              `json:"inactive" gorm:"not null;default:false"`
	Visibility *VisibilityReport `json:"visibility,omitempty" gorm:"-"`
	// HasResults is true once any run has recorded this prompt. Such a prompt can only be stopped.
	HasResults bool   `json:"has_results" gorm:"-"`
	Brand      *Brand `json:"-" gorm:"foreignKey:BrandID;constraint:OnDelete:CASCADE"`
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

	if utf8.RuneCountInString(p.Text) > maxPromptTextLen {
		fields = append(fields, FieldError{Field: "text", Code: FieldMaxLength})
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
	Inactive   *bool
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
// Only a confirmed prompt can be stopped: a draft has nothing to stop, so a prompt is always
// exactly one of tracking, draft or stopped.
func (p *Prompt) Update(params UpdatePromptParams) error {
	if params.Text != nil {
		p.Text = *params.Text
	}

	if params.Inactive != nil {
		if *params.Inactive && !p.Confirmed {
			return Validation(FieldError{Field: "inactive", Code: FieldInvalid})
		}

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

		return tx.PromptRepo().Create(ctx, prompt)
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

		return markPromptResults(ctx, tx, prompt.BrandID, prompt)
	})
	if err != nil {
		return nil, err
	}

	return prompt, nil
}

// Delete removes a prompt that has no saved results. A prompt with results can only be
// stopped, so its history stays reachable and resuming continues the same series. Run
// creation reads prompts under the same brand lock, so the check cannot race a new run.
func (c *promptCommander) Delete(ctx context.Context, id uuid.UUID) error {
	return withLockedChild(ctx, c.store, id, promptChildren, func(tx Store, _ *Brand, prompt *Prompt) error {
		if err := markPromptResults(ctx, tx, prompt.BrandID, prompt); err != nil {
			return err
		}

		if prompt.HasResults {
			return ErrResourceConflict
		}

		return tx.PromptRepo().Delete(ctx, id)
	})
}

func (c *promptCommander) Confirm(ctx context.Context, id uuid.UUID) (*Prompt, error) {
	var prompt *Prompt

	err := withLockedChild(ctx, c.store, id, promptChildren, func(tx Store, _ *Brand, locked *Prompt) error {
		prompt = locked
		prompt.Confirm()

		if err := tx.PromptRepo().Update(ctx, prompt); err != nil {
			return err
		}

		return markPromptResults(ctx, tx, prompt.BrandID, prompt)
	})
	if err != nil {
		return nil, err
	}

	return prompt, nil
}

func promptChildren(s Store) BaseEntityQuerier[Prompt] { return s.PromptRepo() }

// markPromptResults sets HasResults on prompts of one brand with a single query.
func markPromptResults(ctx context.Context, store Store, brandID uuid.UUID, prompts ...*Prompt) error {
	if len(prompts) == 0 {
		return nil
	}

	ids := make([]uuid.UUID, 0, len(prompts))
	for _, prompt := range prompts {
		ids = append(ids, prompt.ID)
	}

	withResults, err := store.VisibilityRepo().PromptsWithResults(ctx, brandID, ids)
	if err != nil {
		return err
	}

	for _, prompt := range prompts {
		prompt.HasResults = withResults[prompt.ID]
	}

	return nil
}

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

func (q *promptQuerier) Get(ctx context.Context, id uuid.UUID) (*Prompt, error) {
	prompt, err := q.brandChildQuerier.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := markPromptResults(ctx, q.store, prompt.BrandID, prompt); err != nil {
		return nil, err
	}

	return prompt, nil
}

func (q *promptQuerier) List(ctx context.Context, filter PromptFilter, page PageParams) (*Page[Prompt], error) {
	prompts, err := q.brandChildQuerier.List(ctx, filter, page)
	if err != nil {
		return nil, err
	}

	items := make([]*Prompt, 0, len(prompts.Items))
	for i := range prompts.Items {
		items = append(items, &prompts.Items[i])
	}

	if err := markPromptResults(ctx, q.store, filter.BrandID, items...); err != nil {
		return nil, err
	}

	return prompts, nil
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
