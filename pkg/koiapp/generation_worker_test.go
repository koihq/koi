package koiapp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/koihq/koi/pkg/koidatabase"
	"github.com/koihq/koi/pkg/koidomain"
	"github.com/koihq/koi/pkg/koiproviders/openrouter"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/require"
)

type workerStore struct {
	run       *koidomain.GenerationRun
	result    koidomain.GenerationResult
	code      string
	retryAt   *time.Time
	finishErr error
}

func (s *workerStore) BeginGenerationAttempt(context.Context, uuid.UUID, int64, int) (*koidomain.GenerationRun, koidomain.GenerationInput, error) {
	return s.run, koidomain.GenerationInput{
		Brand: koidomain.BrandSnapshot{
			Name:   "Koi",
			Domain: "koi.dev",
		},
	}, nil
}

func (s *workerStore) FinishGenerationAttempt(_ context.Context, _ uuid.UUID, _ int, result koidomain.GenerationResult) (int, int, error) {
	s.result = result
	if s.finishErr != nil {
		return 0, 0, s.finishErr
	}

	return len(result.Prompts), len(result.Competitors), nil
}

func (s *workerStore) RecordGenerationFailure(_ context.Context, _ uuid.UUID, _ int, code string, retryAt *time.Time) error {
	s.code, s.retryAt = code, retryAt

	return nil
}

type workerProvider struct {
	result koidomain.GenerationResult
	err    error
}

func (p workerProvider) Generate(context.Context, string, koidomain.GenerationInput) (koidomain.GenerationResult, error) {
	return p.result, p.err
}

func TestGenerationWorkerOutcomes(t *testing.T) {
	for _, tc := range []struct {
		name     string
		attempt  int
		provider workerProvider
		code     string
		retry    bool
	}{
		{
			name:    "success",
			attempt: 1,
			provider: workerProvider{
				result: koidomain.GenerationResult{
					Prompts: []string{"Question?"},
				},
			},
		},
		{
			name:    "retry provider",
			attempt: 1,
			provider: workerProvider{
				err: &openrouter.Error{
					Code:      "provider_rate_limited",
					Retryable: true,
					Err:       errors.New("rate limit"),
				},
			},
			code:  "provider_rate_limited",
			retry: true,
		},
		{
			name:    "last retry",
			attempt: 3,
			provider: workerProvider{
				err: &openrouter.Error{
					Code:      "provider_rate_limited",
					Retryable: true,
					Err:       errors.New("rate limit"),
				},
			},
			code: "provider_rate_limited",
		},
		{
			name:    "terminal auth",
			attempt: 1,
			provider: workerProvider{
				err: &openrouter.Error{
					Code: "provider_auth",
					Err:  errors.New("unauthorized"),
				},
			},
			code: "provider_auth",
		},
		{
			name:    "terminal payment",
			attempt: 1,
			provider: workerProvider{
				err: &openrouter.Error{
					Code: "provider_rejected",
					Err:  errors.New("payment required"),
				},
			},
			code: "provider_rejected",
		},
		{
			name:    "invalid output retries",
			attempt: 1,
			provider: workerProvider{
				err: &openrouter.Error{
					Code:      "invalid_output",
					Retryable: true,
					Err:       errors.New("invalid"),
				},
			},
			code:  "invalid_output",
			retry: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &workerStore{
				run: &koidomain.GenerationRun{
					ModelSlug: "test",
				},
			}
			worker := &generationWorker{store: store, provider: tc.provider}
			job := &river.Job[koidatabase.GenerationJobArgs]{
				JobRow: &rivertype.JobRow{
					Attempt: tc.attempt,
				},
				Args: koidatabase.GenerationJobArgs{
					RunID: uuid.New(),
				},
			}
			err := worker.Work(t.Context(), job)
			if tc.code == "" {
				require.NoError(t, err)
				require.Equal(t, tc.provider.result, store.result)
			} else {
				require.Error(t, err)
				require.Equal(t, tc.code, store.code)
				require.Equal(t, tc.retry, store.retryAt != nil)
				if tc.retry {
					require.WithinDuration(t, time.Now().Add(time.Duration(tc.attempt*tc.attempt*5)*time.Second), *store.retryAt, time.Second)
				}
			}
		})
	}
}

func TestGenerationWorkerDraftSaveFailureRetries(t *testing.T) {
	store := &workerStore{
		run: &koidomain.GenerationRun{
			ModelSlug: "test",
		},
		finishErr: errors.New("database unavailable"),
	}
	worker := &generationWorker{
		store: store,
		provider: workerProvider{
			result: koidomain.GenerationResult{
				Prompts: []string{"Question?"},
			},
		},
	}
	job := &river.Job[koidatabase.GenerationJobArgs]{
		JobRow: &rivertype.JobRow{
			Attempt: 1,
		},
		Args: koidatabase.GenerationJobArgs{
			RunID: uuid.New(),
		},
	}
	require.Error(t, worker.Work(t.Context(), job))
	require.Equal(t, "storage_error", store.code)
	require.NotNil(t, store.retryAt)
}
