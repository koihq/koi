package koidomain

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPromptValidationAndNormalization(t *testing.T) {
	prompt := Prompt{BrandID: uuid.New(), Text: "  best\n  analytics   tools? ", Source: PromptSourceManual, Confirmed: true}
	prompt.Normalize()
	require.Equal(t, "best analytics tools?", prompt.Text)
	require.NoError(t, prompt.Validate())

	for _, invalid := range []Prompt{
		{Text: "Question", Source: PromptSourceManual, Confirmed: true},
		{BrandID: uuid.New(), Text: " ", Source: PromptSourceManual, Confirmed: true},
		{BrandID: uuid.New(), Text: "Question", Source: "observed", Confirmed: true},
		{BrandID: uuid.New(), Text: "Question", Source: PromptSourceManual},
	} {
		require.ErrorIs(t, invalid.Validate(), ErrValidation)
	}
	require.NoError(t, (Prompt{BrandID: uuid.New(), Text: "Question", Source: PromptSourceDrafted}).Validate())
}

func TestPromptManualCreateAndImmutableProvenance(t *testing.T) {
	store := NewMockStore(t)
	brands := NewMockBrandRepository(t)
	prompts := NewMockPromptRepository(t)
	userID, brandID, promptID := uuid.New(), uuid.New(), uuid.New()
	store.EXPECT().BrandRepo().Return(brands).Twice()
	store.EXPECT().PromptRepo().Return(prompts).Times(3)
	brands.EXPECT().GetByIDAndUserID(mock.Anything, brandID, userID).Return(&Brand{}, nil).Twice()
	prompts.EXPECT().Create(mock.Anything, mock.MatchedBy(func(prompt *Prompt) bool {
		return prompt.Text == "What is Koi?" && prompt.Source == PromptSourceManual && prompt.Confirmed
	})).RunAndReturn(func(_ context.Context, prompt *Prompt) error {
		prompt.ID = promptID
		return nil
	})

	created, err := NewPromptCommander(store).Create(t.Context(), userID, brandID, PromptParams{Text: " What   is Koi? "})
	require.NoError(t, err)
	require.Equal(t, promptID, created.ID)

	drafted := &Prompt{BaseEntity: BaseEntity{ID: promptID}, BrandID: brandID, Text: "Old", Source: PromptSourceDrafted}
	prompts.EXPECT().Get(mock.Anything, PromptFilter{ID: promptID, BrandID: brandID}).Return(drafted, nil)
	prompts.EXPECT().Update(mock.Anything, PromptFilter{ID: promptID, BrandID: brandID}, mock.MatchedBy(func(update PromptUpdate) bool {
		return update.Text != nil && *update.Text == "New text" && update.Confirmed == nil
	})).Return(nil)
	text := " New   text "
	updated, err := NewPromptCommander(store).Update(t.Context(), userID, brandID, promptID, PromptPatch{Text: &text})
	require.NoError(t, err)
	require.Equal(t, PromptSourceDrafted, updated.Source)
	require.False(t, updated.Confirmed)
}

func TestPromptConfirmationAndOwnership(t *testing.T) {
	store := NewMockStore(t)
	brands := NewMockBrandRepository(t)
	prompts := NewMockPromptRepository(t)
	userID, brandID, promptID := uuid.New(), uuid.New(), uuid.New()
	store.EXPECT().BrandRepo().Return(brands).Times(3)
	store.EXPECT().PromptRepo().Return(prompts).Times(3)
	brands.EXPECT().GetByIDAndUserID(mock.Anything, brandID, userID).Return(&Brand{}, nil).Twice()
	brands.EXPECT().GetByIDAndUserID(mock.Anything, brandID, userID).Return(nil, ErrNotFound).Once()
	confirmed := &Prompt{BaseEntity: BaseEntity{ID: promptID}, BrandID: brandID, Source: PromptSourceDrafted}
	prompts.EXPECT().Get(mock.Anything, PromptFilter{ID: promptID, BrandID: brandID}).Return(confirmed, nil)
	prompts.EXPECT().Update(mock.Anything, PromptFilter{ID: promptID, BrandID: brandID}, mock.MatchedBy(func(update PromptUpdate) bool {
		return update.Confirmed != nil && *update.Confirmed
	})).Return(nil)

	got, err := NewPromptCommander(store).Confirm(t.Context(), userID, brandID, promptID)
	require.NoError(t, err)
	require.True(t, got.Confirmed)

	prompts.EXPECT().Update(mock.Anything, mock.MatchedBy(func(filter PromptFilter) bool {
		return filter.BrandID == brandID && filter.ID == uuid.Nil && filter.Confirmed != nil && !*filter.Confirmed
	}), mock.MatchedBy(func(update PromptUpdate) bool {
		return update.Confirmed != nil && *update.Confirmed
	})).Return(nil)
	require.NoError(t, NewPromptCommander(store).ConfirmAll(t.Context(), userID, brandID))

	err = NewPromptCommander(store).Delete(t.Context(), userID, brandID, promptID)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestListsReady(t *testing.T) {
	confirmedPrompt := Prompt{Confirmed: true}
	confirmedCompetitor := Competitor{Confirmed: true}
	require.False(t, ListsReady(nil, []Competitor{confirmedCompetitor}))
	require.False(t, ListsReady([]Prompt{confirmedPrompt}, nil))
	require.False(t, ListsReady([]Prompt{{Confirmed: false}}, []Competitor{confirmedCompetitor}))
	require.False(t, ListsReady([]Prompt{confirmedPrompt}, []Competitor{{Confirmed: false}}))
	require.True(t, ListsReady([]Prompt{confirmedPrompt}, []Competitor{confirmedCompetitor}))
}
