package domain

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPromptValidationAndNormalization(t *testing.T) {
	prompt := Prompt{
		BrandID:   uuid.New(),
		Text:      "  best\n  analytics   tools? ",
		Source:    PromptSourceManual,
		Confirmed: true,
	}
	prompt.Normalize()
	require.Equal(t, "best analytics tools?", prompt.Text)
	require.NoError(t, prompt.Validate())

	for _, invalid := range []Prompt{
		{
			Text:      "Question",
			Source:    PromptSourceManual,
			Confirmed: true,
		},
		{
			BrandID:   uuid.New(),
			Text:      " ",
			Source:    PromptSourceManual,
			Confirmed: true,
		},
		{
			BrandID:   uuid.New(),
			Text:      "Question",
			Source:    "observed",
			Confirmed: true,
		},
		{
			BrandID: uuid.New(),
			Text:    "Question",
			Source:  PromptSourceManual,
		},
	} {
		require.ErrorIs(t, invalid.Validate(), ErrValidation)
	}

	require.NoError(t, (Prompt{BrandID: uuid.New(), Text: "Question", Source: PromptSourceDrafted}).Validate())
}

// promptCommandDeps runs Atomic against the same mocks and records the order of the steps.
type promptCommandDeps struct {
	store      *MockStore
	brands     *MockBrandRepository
	prompts    *MockPromptRepository
	billing    *MockBillingRepository
	visibility *MockVisibilityRepository
	steps      []string
}

func newPromptCommandDeps(t *testing.T) *promptCommandDeps {
	t.Helper()

	d := &promptCommandDeps{
		store:      NewMockStore(t),
		brands:     NewMockBrandRepository(t),
		prompts:    NewMockPromptRepository(t),
		billing:    NewMockBillingRepository(t),
		visibility: NewMockVisibilityRepository(t),
	}
	d.store.EXPECT().BrandRepo().Return(d.brands).Maybe()
	d.store.EXPECT().PromptRepo().Return(d.prompts).Maybe()
	d.store.EXPECT().BillingRepo().Return(d.billing).Maybe()
	d.store.EXPECT().VisibilityRepo().Return(d.visibility).Maybe()
	d.store.EXPECT().Atomic(mock.Anything, mock.Anything).RunAndReturn(
		func(_ context.Context, fn func(Store) error) error { return fn(d.store) },
	).Maybe()

	return d
}

func (d *promptCommandDeps) lockBrand(brandID, organizationID uuid.UUID) {
	d.brands.EXPECT().LockForUpdate(mock.Anything, brandID).Run(func(context.Context, uuid.UUID) {
		d.steps = append(d.steps, "lock")
	}).Return(&Brand{
		BaseEntity:     BaseEntity{ID: brandID},
		OrganizationID: organizationID,
	}, nil).Once()
}

func (d *promptCommandDeps) invalidateReview(brandID uuid.UUID) {
	d.visibility.EXPECT().InvalidateReview(mock.Anything, brandID, ReviewPrompts).Run(func(context.Context, uuid.UUID, string) {
		d.steps = append(d.steps, "invalidate")
	}).Return(nil).Once()
}

func TestPromptUpdateKeepsProvenanceAndConfirmation(t *testing.T) {
	prompt := &Prompt{
		BrandID: uuid.New(),
		Text:    "Old",
		Source:  PromptSourceDrafted,
	}
	text := " New   text "
	inactive := true

	require.NoError(t, prompt.Update(UpdatePromptParams{Text: &text, Inactive: &inactive}))
	require.Equal(t, "New text", prompt.Text)
	require.True(t, prompt.Inactive)
	require.Equal(t, PromptSourceDrafted, prompt.Source)
	require.False(t, prompt.Confirmed)

	empty := " "
	require.ErrorIs(t, prompt.Update(UpdatePromptParams{Text: &empty}), ErrValidation)
}

func TestPromptCreateReservesCapacityAndInvalidatesReviewInOrder(t *testing.T) {
	d := newPromptCommandDeps(t)
	brandID, organizationID, promptID := uuid.New(), uuid.New(), uuid.New()
	d.lockBrand(brandID, organizationID)
	d.billing.EXPECT().ReservePromptCapacity(mock.Anything, organizationID, int64(1)).Run(func(context.Context, uuid.UUID, int64) {
		d.steps = append(d.steps, "capacity")
	}).Return(nil).Once()
	d.prompts.EXPECT().Create(mock.Anything, mock.MatchedBy(func(prompt *Prompt) bool {
		return prompt.Text == "What is Koi?" && prompt.Source == PromptSourceManual && prompt.Confirmed
	})).RunAndReturn(func(_ context.Context, prompt *Prompt) error {
		d.steps = append(d.steps, "create")
		prompt.ID = promptID

		return nil
	}).Once()
	d.invalidateReview(brandID)

	created, err := NewPromptCommander(d.store).Create(t.Context(), brandID, CreatePromptParams{Text: " What   is Koi? "})
	require.NoError(t, err)
	require.Equal(t, promptID, created.ID)
	require.Equal(t, []string{"lock", "capacity", "create", "invalidate"}, d.steps)
}

func TestPromptCreateChecksTheBrandBeforeValidatingAndStopsAtTheLimit(t *testing.T) {
	brandID, organizationID := uuid.New(), uuid.New()

	foreign := newPromptCommandDeps(t)
	foreign.brands.EXPECT().LockForUpdate(mock.Anything, brandID).Return(nil, ErrNotFound).Once()
	_, err := NewPromptCommander(foreign.store).Create(t.Context(), brandID, CreatePromptParams{Text: " "})
	require.ErrorIs(t, err, ErrNotFound)

	invalid := newPromptCommandDeps(t)
	invalid.lockBrand(brandID, organizationID)
	_, err = NewPromptCommander(invalid.store).Create(t.Context(), brandID, CreatePromptParams{Text: " "})
	require.ErrorIs(t, err, ErrValidation)

	limited := newPromptCommandDeps(t)
	limited.lockBrand(brandID, organizationID)
	limited.billing.EXPECT().ReservePromptCapacity(mock.Anything, organizationID, int64(1)).Return(ErrPromptLimitReached).Once()
	_, err = NewPromptCommander(limited.store).Create(t.Context(), brandID, CreatePromptParams{Text: "Question"})
	require.ErrorIs(t, err, ErrPromptLimitReached)
}

func TestPromptUpdateReadsUnderLockAndInvalidatesReview(t *testing.T) {
	d := newPromptCommandDeps(t)
	brandID, organizationID, promptID := uuid.New(), uuid.New(), uuid.New()
	stale := &Prompt{BaseEntity: BaseEntity{ID: promptID}, BrandID: brandID, Text: "Stale", Source: PromptSourceDrafted}
	locked := &Prompt{BaseEntity: BaseEntity{ID: promptID}, BrandID: brandID, Text: "Locked", Source: PromptSourceDrafted, Confirmed: true}
	d.prompts.EXPECT().Get(mock.Anything, promptID).Return(stale, nil).Once()
	d.lockBrand(brandID, organizationID)
	d.prompts.EXPECT().Get(mock.Anything, promptID).Run(func(context.Context, uuid.UUID) {
		d.steps = append(d.steps, "read")
	}).Return(locked, nil).Once()
	d.prompts.EXPECT().Update(mock.Anything, locked).Run(func(context.Context, *Prompt) {
		d.steps = append(d.steps, "write")
	}).Return(nil).Once()
	d.invalidateReview(brandID)

	text := " New   text "
	updated, err := NewPromptCommander(d.store).Update(t.Context(), promptID, UpdatePromptParams{Text: &text})
	require.NoError(t, err)
	require.Equal(t, "New text", updated.Text)
	require.True(t, updated.Confirmed, "the update starts from the row read under the lock")
	require.Equal(t, []string{"lock", "read", "write", "invalidate"}, d.steps)
}

func TestPromptUpdateReservesCapacityOnlyWhenReactivating(t *testing.T) {
	brandID, organizationID, promptID := uuid.New(), uuid.New(), uuid.New()
	active, inactive := false, true

	reactivating := newPromptCommandDeps(t)
	reactivated := &Prompt{BaseEntity: BaseEntity{ID: promptID}, BrandID: brandID, Text: "Q", Source: PromptSourceDrafted, Inactive: true}
	reactivating.prompts.EXPECT().Get(mock.Anything, promptID).Return(reactivated, nil).Twice()
	reactivating.lockBrand(brandID, organizationID)
	reactivating.billing.EXPECT().ReservePromptCapacity(mock.Anything, organizationID, int64(1)).Return(ErrPromptLimitReached).Once()

	_, err := NewPromptCommander(reactivating.store).Update(t.Context(), promptID, UpdatePromptParams{Inactive: &active})
	require.ErrorIs(t, err, ErrPromptLimitReached)

	// No ReservePromptCapacity expectation: the mock fails the test if deactivating reserves.
	deactivating := newPromptCommandDeps(t)
	deactivated := &Prompt{BaseEntity: BaseEntity{ID: promptID}, BrandID: brandID, Text: "Q", Source: PromptSourceDrafted}
	deactivating.prompts.EXPECT().Get(mock.Anything, promptID).Return(deactivated, nil).Twice()
	deactivating.lockBrand(brandID, organizationID)
	deactivating.prompts.EXPECT().Update(mock.Anything, deactivated).Return(nil).Once()
	deactivating.invalidateReview(brandID)

	_, err = NewPromptCommander(deactivating.store).Update(t.Context(), promptID, UpdatePromptParams{Inactive: &inactive})
	require.NoError(t, err)
}

func TestPromptEmptyUpdateKeepsTheReview(t *testing.T) {
	d := newPromptCommandDeps(t)
	brandID, organizationID, promptID := uuid.New(), uuid.New(), uuid.New()
	prompt := &Prompt{BaseEntity: BaseEntity{ID: promptID}, BrandID: brandID, Text: "Q", Source: PromptSourceDrafted}
	d.prompts.EXPECT().Get(mock.Anything, promptID).Return(prompt, nil).Twice()
	d.lockBrand(brandID, organizationID)
	d.prompts.EXPECT().Update(mock.Anything, prompt).Return(nil).Once()

	_, err := NewPromptCommander(d.store).Update(t.Context(), promptID, UpdatePromptParams{})
	require.NoError(t, err)
}

func TestPromptConfirmAndConfirmAll(t *testing.T) {
	d := newPromptCommandDeps(t)
	brandID, organizationID, promptID := uuid.New(), uuid.New(), uuid.New()
	prompt := &Prompt{BaseEntity: BaseEntity{ID: promptID}, BrandID: brandID, Source: PromptSourceDrafted}
	d.prompts.EXPECT().Get(mock.Anything, promptID).Return(prompt, nil).Twice()
	d.lockBrand(brandID, organizationID)
	d.prompts.EXPECT().Update(mock.Anything, mock.MatchedBy(func(p *Prompt) bool { return p.Confirmed })).Return(nil).Once()

	got, err := NewPromptCommander(d.store).Confirm(t.Context(), promptID)
	require.NoError(t, err)
	require.True(t, got.Confirmed)

	d.visibility.EXPECT().FinishReview(mock.Anything, brandID, ReviewPrompts).Return(nil).Once()
	require.NoError(t, NewPromptCommander(d.store).ConfirmAll(t.Context(), brandID))
}

func TestPromptDeleteInOrderAndUnknownPrompt(t *testing.T) {
	d := newPromptCommandDeps(t)
	brandID, organizationID, promptID := uuid.New(), uuid.New(), uuid.New()
	prompt := &Prompt{BaseEntity: BaseEntity{ID: promptID}, BrandID: brandID}
	d.prompts.EXPECT().Get(mock.Anything, promptID).Return(prompt, nil).Twice()
	d.lockBrand(brandID, organizationID)
	d.prompts.EXPECT().Delete(mock.Anything, promptID).Run(func(context.Context, uuid.UUID) {
		d.steps = append(d.steps, "delete")
	}).Return(nil).Once()
	d.invalidateReview(brandID)

	require.NoError(t, NewPromptCommander(d.store).Delete(t.Context(), promptID))
	require.Equal(t, []string{"lock", "delete", "invalidate"}, d.steps)

	missing := newPromptCommandDeps(t)
	missing.prompts.EXPECT().Get(mock.Anything, promptID).Return(nil, ErrNotFound).Once()
	require.ErrorIs(t, NewPromptCommander(missing.store).Delete(t.Context(), promptID), ErrNotFound)
}
