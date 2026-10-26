package koidomain

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

func TestPromptManualCreateAndImmutableProvenance(t *testing.T) {
	store := NewMockStore(t)
	brands := NewMockBrandRepository(t)
	prompts := NewMockPromptRepository(t)
	userID, brandID, promptID := uuid.New(), uuid.New(), uuid.New()
	store.EXPECT().BrandRepo().Return(brands).Once()
	store.EXPECT().PromptRepo().Return(prompts).Times(3)
	brands.EXPECT().Get(mock.Anything, brandID).Return(&Brand{}, nil).Once()
	prompts.EXPECT().Create(mock.Anything, mock.MatchedBy(func(prompt *Prompt) bool {
		return prompt.Text == "What is Koi?" && prompt.Source == PromptSourceManual && prompt.Confirmed
	})).RunAndReturn(func(_ context.Context, prompt *Prompt) error {
		prompt.ID = promptID

		return nil
	})

	created, err := NewPromptCommander(store).Create(identityCtx(t, userID), brandID, PromptParams{Text: " What   is Koi? "})
	require.NoError(t, err)
	require.Equal(t, promptID, created.ID)

	drafted := &Prompt{
		BaseEntity: BaseEntity{
			ID: promptID,
		},
		BrandID: brandID,
		Text:    "Old",
		Source:  PromptSourceDrafted,
	}
	prompts.EXPECT().Get(mock.Anything, promptID).Return(drafted, nil)
	prompts.EXPECT().Update(mock.Anything, promptID, mock.MatchedBy(func(update PromptUpdate) bool {
		return update.Text != nil && *update.Text == "New text" && update.Confirmed == nil
	})).Return(nil)
	text := " New   text "
	updated, err := NewPromptCommander(store).Update(identityCtx(t, userID), promptID, PromptPatch{Text: &text})
	require.NoError(t, err)
	require.Equal(t, PromptSourceDrafted, updated.Source)
	require.False(t, updated.Confirmed)
}

func TestPromptConfirmationAndOwnership(t *testing.T) {
	store := NewMockStore(t)
	prompts := NewMockPromptRepository(t)
	userID, brandID, promptID := uuid.New(), uuid.New(), uuid.New()
	store.EXPECT().PromptRepo().Return(prompts).Times(3)
	confirmed := &Prompt{
		BaseEntity: BaseEntity{
			ID: promptID,
		},
		BrandID: brandID,
		Source:  PromptSourceDrafted,
	}
	prompts.EXPECT().Get(mock.Anything, promptID).Return(confirmed, nil)
	prompts.EXPECT().Update(mock.Anything, promptID, mock.MatchedBy(func(update PromptUpdate) bool {
		return update.Confirmed != nil && *update.Confirmed
	})).Return(nil)

	got, err := NewPromptCommander(store).Confirm(identityCtx(t, userID), promptID)
	require.NoError(t, err)
	require.True(t, got.Confirmed)

	visibility := NewMockVisibilityRepository(t)
	store.EXPECT().VisibilityRepo().Return(visibility).Once()
	visibility.EXPECT().FinishReview(mock.Anything, brandID, "prompts").Return(nil).Once()

	require.NoError(t, NewPromptCommander(store).ConfirmAll(identityCtx(t, userID), brandID))

	prompts.EXPECT().Delete(mock.Anything, promptID).Return(ErrNotFound)
	err = NewPromptCommander(store).Delete(identityCtx(t, userID), promptID)
	require.ErrorIs(t, err, ErrNotFound)
}
