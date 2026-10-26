package koidomain

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestVisibilitySummaryPropagatesErrors(t *testing.T) {
	failed := errors.New("database unavailable")
	for _, tt := range []struct {
		name        string
		brandError  error
		promptError error
	}{
		{name: "foreign brand", brandError: ErrNotFound},
		{name: "prompt state unavailable", promptError: failed},
	} {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMockStore(t)
			brands := NewMockBrandRepository(t)
			store.EXPECT().BrandRepo().Return(brands)
			brandID := uuid.New()
			brands.EXPECT().Get(mock.Anything, brandID).Return(&Brand{}, tt.brandError)
			if tt.brandError == nil {
				visibility := NewMockVisibilityRepository(t)
				store.EXPECT().VisibilityRepo().Return(visibility)
				visibility.EXPECT().AggregateVisibility(mock.Anything, brandID, []uuid.UUID(nil), 30, []EngineID(nil), mock.Anything).
					Return(nil, &VisibilityReport{}, nil)
				prompts := NewMockPromptRepository(t)
				store.EXPECT().PromptRepo().Return(prompts)
				prompts.EXPECT().HasConfirmed(mock.Anything, brandID).Return(false, tt.promptError)
			}

			summary, err := NewVisibilityAnalyticsQuerier(store).Summary(t.Context(), brandID, 30, nil)
			want := tt.brandError
			if want == nil {
				want = tt.promptError
			}
			require.ErrorIs(t, err, want)
			require.Nil(t, summary)
		})
	}
}
