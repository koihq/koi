package koidomain

import (
	"context"
	"errors"
	"testing"

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
		Mode:  GenerationInitial,
		State: GenerationQueued,
	}
	terminal := &GenerationRun{
		BaseEntity: BaseEntity{
			ID: runID,
		},
		Mode:  GenerationInitial,
		State: GenerationSucceeded,
	}

	for _, tc := range []struct {
		name     string
		mode     GenerationMode
		key      *string
		byKey    *GenerationRun
		active   *GenerationRun
		success  bool
		enabled  bool
		terminal bool
		err      error
		created  bool
	}{
		{
			name:    "new initial",
			mode:    GenerationInitial,
			enabled: true,
			created: true,
		},
		{
			name:    "active key replay",
			mode:    GenerationInitial,
			key:     &key,
			byKey:   active,
			enabled: true,
		},
		{
			name:     "terminal key replay",
			mode:     GenerationInitial,
			key:      &key,
			byKey:    terminal,
			enabled:  true,
			terminal: true,
		},
		{
			name:    "key mode conflict",
			mode:    GenerationPrompts,
			key:     &key,
			byKey:   active,
			enabled: true,
			err:     ErrIdempotencyConflict,
		},
		{
			name:    "same active mode",
			mode:    GenerationInitial,
			active:  active,
			enabled: true,
		},
		{
			name:    "initial satisfies prompts",
			mode:    GenerationPrompts,
			active:  active,
			enabled: true,
		},
		{
			name:    "initial satisfies competitors",
			mode:    GenerationCompetitors,
			active:  active,
			enabled: true,
		},
		{
			name: "active mode conflict",
			mode: GenerationCompetitors,
			active: &GenerationRun{
				BaseEntity: BaseEntity{
					ID: runID,
				},
				Mode:  GenerationPrompts,
				State: GenerationRunning,
			},
			enabled: true,
			err:     ErrGenerationConflict,
		},
		{
			name:    "initial already completed",
			mode:    GenerationInitial,
			success: true,
			enabled: true,
			err:     ErrInitialAlreadyCompleted,
		},
		{
			name:    "repeatable after initial",
			mode:    GenerationPrompts,
			success: true,
			enabled: true,
			created: true,
		},
		{
			name: "disabled",
			mode: GenerationInitial,
			err:  ErrFeatureUnavailable,
		},
		{
			name:    "invalid mode",
			mode:    "other",
			enabled: true,
			err:     ErrValidation,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := NewMockStore(t)
			brands := NewMockBrandRepository(t)
			runs := NewMockGenerationRepository(t)
			var created *GenerationRun
			if validGenerationMode(tc.mode) {
				store.EXPECT().Atomic(mock.Anything, mock.Anything).RunAndReturn(
					func(_ context.Context, fn func(Store) error) error { return fn(store) },
				)
				store.EXPECT().BrandRepo().Return(brands)
				brands.EXPECT().LockForUpdate(mock.Anything, brandID).Return(brand, nil)
				if tc.enabled {
					store.EXPECT().GenerationRepo().Return(runs)
					if tc.key != nil {
						if tc.byKey != nil {
							runs.EXPECT().ByKey(mock.Anything, brandID, key).Return(tc.byKey, nil)
						} else {
							runs.EXPECT().ByKey(mock.Anything, brandID, key).Return(nil, ErrNotFound)
						}
					}

					if tc.byKey == nil {
						if tc.active != nil {
							runs.EXPECT().Active(mock.Anything, brandID).Return(tc.active, nil)
						} else {
							runs.EXPECT().Active(mock.Anything, brandID).Return(nil, ErrNotFound)
							if tc.mode == GenerationInitial {
								runs.EXPECT().HasSucceededInitial(mock.Anything, brandID).Return(tc.success, nil)
							}

							if tc.created {
								runs.EXPECT().CreateAndEnqueue(mock.Anything, mock.Anything).RunAndReturn(
									func(_ context.Context, run *GenerationRun) error { created = run; return nil },
								)
							}
						}
					}
				}
			}

			got, terminalReplay, err := NewGenerationCommander(store, tc.enabled).Create(t.Context(), brandID, tc.mode, tc.key)
			if tc.err != nil {
				require.True(t, errors.Is(err, tc.err), "error: %v", err)
				if errors.Is(err, ErrGenerationConflict) {
					var domainErr *Error
					require.ErrorAs(t, err, &domainErr)
					require.Equal(t, runID, domainErr.RunID)
				}

				require.Nil(t, created)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.terminal, terminalReplay)
			if tc.created {
				require.Same(t, created, got)
				require.Equal(t, tc.mode, got.Mode)
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
	_, _, err := NewGenerationCommander(store, true).Create(t.Context(), brandID, GenerationInitial, nil)
	require.ErrorIs(t, err, ErrNotFound)
}
