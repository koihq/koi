package domain

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGenerationCreateDecision(t *testing.T) {
	brandID, runID := uuid.New(), uuid.New()
	brand := &Brand{
		BaseEntity: BaseEntity{
			ID: brandID,
		},
		Name:        "Koi",
		Domain:      "koi.dev",
		Country:     "IT",
		Description: "Visibility",
	}
	key := "setup-1"
	active := &GenerationRun{
		BaseEntity: BaseEntity{
			ID: runID,
		},
		State: GenerationQueued,
	}
	terminal := &GenerationRun{
		BaseEntity: BaseEntity{
			ID: runID,
		},
		State: GenerationSucceeded,
	}

	for _, tc := range []struct {
		name      string
		confirmed bool
		key       *string
		byKey     *GenerationRun
		active    *GenerationRun
		success   bool
		enabled   bool
		terminal  bool
		err       error
		created   bool
	}{
		{
			name:    "new draft",
			enabled: true,
			created: true,
		},
		{
			name:    "active key replay",
			key:     &key,
			byKey:   active,
			enabled: true,
		},
		{
			name:     "terminal key replay",
			key:      &key,
			byKey:    terminal,
			enabled:  true,
			terminal: true,
		},
		{
			name:    "active draft",
			active:  active,
			enabled: true,
		},
		{
			name:    "draft already succeeded",
			success: true,
			enabled: true,
			err:     ErrInitialAlreadyCompleted,
		},
		{
			name:      "tracking already started",
			confirmed: true,
			enabled:   true,
			err:       ErrInitialAlreadyCompleted,
		},
		{
			name: "disabled",
			err:  ErrFeatureUnavailable,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := NewMockStore(t)
			brands := NewMockBrandRepository(t)
			runs := NewMockGenerationRepository(t)
			var created *GenerationRun
			locked := *brand
			if tc.confirmed {
				now := time.Now()
				locked.TrackingConfirmedAt = &now
			}

			store.EXPECT().Atomic(mock.Anything, mock.Anything).RunAndReturn(
				func(_ context.Context, fn func(Store) error) error { return fn(store) },
			)
			store.EXPECT().BrandRepo().Return(brands)
			brands.EXPECT().LockForUpdate(mock.Anything, brandID).Return(&locked, nil)
			if tc.enabled {
				store.EXPECT().GenerationRepo().Return(runs)
				if tc.key != nil {
					runs.EXPECT().ByKey(mock.Anything, brandID, key).Return(tc.byKey, nil)
				}

				if tc.byKey == nil {
					if tc.active != nil {
						runs.EXPECT().Active(mock.Anything, brandID).Return(tc.active, nil)
					} else {
						runs.EXPECT().Active(mock.Anything, brandID).Return(nil, ErrNotFound)
						if !tc.confirmed {
							runs.EXPECT().HasSucceeded(mock.Anything, brandID).Return(tc.success, nil)
						}

						if tc.created {
							runs.EXPECT().CreateAndEnqueue(mock.Anything, mock.Anything).RunAndReturn(
								func(_ context.Context, run *GenerationRun) error { created = run; return nil },
							)
						}
					}
				}
			}

			got, terminalReplay, err := NewGenerationCommander(store, tc.enabled).Create(t.Context(), brandID, tc.key)
			if tc.err != nil {
				require.True(t, errors.Is(err, tc.err), "error: %v", err)
				require.Nil(t, created)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.terminal, terminalReplay)
			if tc.created {
				require.Same(t, created, got)
				require.Equal(t, GenerationQueued, got.State)
				require.Equal(t, BrandSnapshot{
					Name:        "Koi",
					Domain:      "koi.dev",
					Country:     "IT",
					Description: "Visibility",
				}, got.BrandSnapshot)
				require.Equal(t, GenerationModelSlug, got.ModelSlug)
			} else if tc.byKey != nil {
				require.Same(t, tc.byKey, got)
			} else {
				require.Same(t, tc.active, got)
			}
		})
	}
}

func TestGenerationCreateRequiresOwnedBrand(t *testing.T) {
	store := NewMockStore(t)
	brands := NewMockBrandRepository(t)
	brandID := uuid.New()
	store.EXPECT().Atomic(mock.Anything, mock.Anything).RunAndReturn(
		func(_ context.Context, fn func(Store) error) error { return fn(store) },
	)
	store.EXPECT().BrandRepo().Return(brands)
	brands.EXPECT().LockForUpdate(mock.Anything, brandID).Return(nil, ErrNotFound)
	_, _, err := NewGenerationCommander(store, true).Create(t.Context(), brandID, nil)
	require.ErrorIs(t, err, ErrNotFound)
}
