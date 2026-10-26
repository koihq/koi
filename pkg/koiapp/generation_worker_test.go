package koiapp

import (
	"context"
	"errors"
	"fmt"
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
	failure   func(context.Context) error
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

func (s *workerStore) RecordGenerationFailure(ctx context.Context, _ uuid.UUID, _ int, code string, retryAt *time.Time) error {
	s.code, s.retryAt = code, retryAt
	if s.failure != nil {
		return s.failure(ctx)
	}

	return nil
}

type workerProvider struct {
	result   koidomain.GenerationResult
	err      error
	generate func(context.Context) (koidomain.GenerationResult, error)
}

func (p workerProvider) Generate(ctx context.Context, _ string, _ koidomain.GenerationInput) (koidomain.GenerationResult, error) {
	if p.generate != nil {
		return p.generate(ctx)
	}

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
					Attempt:     tc.attempt,
					MaxAttempts: 3,
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
			Attempt:     1,
			MaxAttempts: 3,
		},
		Args: koidatabase.GenerationJobArgs{
			RunID: uuid.New(),
		},
	}
	require.Error(t, worker.Work(t.Context(), job))
	require.Equal(t, "storage_error", store.code)
	require.NotNil(t, store.retryAt)
}

func TestGenerationWorkerTimeoutAndFailureCleanup(t *testing.T) {
	for _, attempt := range []int{1, 2} {
		t.Run(fmt.Sprintf("attempt-%d", attempt), func(t *testing.T) {
			store := &workerStore{
				run: &koidomain.GenerationRun{
					ModelSlug: "test",
				},
			}
			store.failure = func(ctx context.Context) error {
				require.NoError(t, ctx.Err())
				deadline, ok := ctx.Deadline()
				require.True(t, ok)
				require.WithinDuration(t, time.Now().Add(5*time.Second), deadline, time.Second)
				return nil
			}
			worker := &generationWorker{
				store:   store,
				timeout: 2*3*time.Minute + 30*time.Second,
				provider: workerProvider{
					generate: func(ctx context.Context) (koidomain.GenerationResult, error) {
						<-ctx.Done()

						return koidomain.GenerationResult{}, ctx.Err()
					},
				},
			}
			job := &river.Job[koidatabase.GenerationJobArgs]{
				JobRow: &rivertype.JobRow{
					Attempt:     attempt,
					MaxAttempts: 2,
				},
			}
			require.Equal(t, 6*time.Minute+30*time.Second, worker.Timeout(job))
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
			defer cancel()
			err := worker.Work(ctx, job)
			require.ErrorIs(t, err, context.DeadlineExceeded)
			require.Equal(t, "job_timeout", store.code)
			if attempt == job.MaxAttempts {
				var terminal *river.JobCancelError
				require.ErrorAs(t, err, &terminal)
				require.Nil(t, store.retryAt)
			} else {
				require.NotNil(t, store.retryAt)
			}
		})
	}
}

func TestGenerationRetryStartsAtFailureAndShutdownIsResumable(t *testing.T) {
	started := time.Now().Add(-2 * time.Minute)
	job := &river.Job[koidatabase.GenerationJobArgs]{
		JobRow: &rivertype.JobRow{
			Attempt:     2,
			MaxAttempts: 3,
			AttemptedAt: &started,
		},
	}
	store := &workerStore{
		run: &koidomain.GenerationRun{
			ModelSlug: "test",
		},
	}
	worker := &generationWorker{
		store: store,
		provider: workerProvider{
			err: context.Canceled,
		},
	}
	require.WithinDuration(t, time.Now().Add(20*time.Second), worker.NextRetry(job), time.Second)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	require.ErrorIs(t, worker.Work(ctx, job), context.Canceled)
	require.Empty(t, store.code)
	require.Nil(t, store.retryAt)
}
