package koiapp

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/koidatabase"
	"github.com/koihq/koi/pkg/koidomain"
	"github.com/koihq/koi/pkg/koiproviders/stub"
)

type visibilityWorkerStore struct {
	run      *koidomain.VisibilityRun
	pending  []koidomain.VisibilityResult
	saved    []koidomain.VisibilityResult
	saveErr  error
	beginErr error
	save     func(context.Context) error
	failure  func(context.Context) error
	finished bool
	code     string
	terminal bool
}

func (s *visibilityWorkerStore) BeginVisibilityAttempt(context.Context, uuid.UUID, int64, int) (*koidomain.VisibilityRun, []koidomain.VisibilityResult, error) {
	return s.run, s.pending, s.beginErr
}

func (s *visibilityWorkerStore) SaveVisibilityResult(ctx context.Context, _ uuid.UUID, _ int, result koidomain.VisibilityResult) error {
	if s.save != nil {
		if err := s.save(ctx); err != nil {
			return err
		}
	}

	if s.saveErr != nil {
		return s.saveErr
	}

	s.saved = append(s.saved, result)

	return nil
}

func (s *visibilityWorkerStore) FinishVisibilityAttempt(context.Context, uuid.UUID, int) error {
	s.finished = true

	return nil
}

func (s *visibilityWorkerStore) RecordVisibilityFailure(ctx context.Context, _ uuid.UUID, _ int64, _ int, code string, terminal bool) error {
	s.code, s.terminal = code, terminal
	if s.failure != nil {
		return s.failure(ctx)
	}

	return nil
}

func newVisibilityWorkerFixture() (*visibilityWorkerStore, *river.Job[koidatabase.VisibilityJobArgs]) {
	runID := uuid.New()

	store := &visibilityWorkerStore{
		run: &koidomain.VisibilityRun{
			BaseEntity: koidomain.BaseEntity{
				ID: runID,
			},
			TotalResults: 4,
			BrandSnapshot: koidomain.BrandSnapshot{
				Name:       "Koi",
				GitHubRepo: "https://github.com/koihq/koi",
			},
		},
	}

	for _, engine := range koidomain.VisibilityEngines() {
		store.pending = append(store.pending, koidomain.VisibilityResult{
			RunID:      runID,
			PromptID:   uuid.New(),
			PromptText: "Best tools?",
			Engine:     engine,
			Collector:  koidomain.CollectorStub,
			Sources:    []string{},
		})
	}

	job := &river.Job[koidatabase.VisibilityJobArgs]{
		JobRow: &rivertype.JobRow{
			ID:          1,
			Attempt:     1,
			MaxAttempts: 3,
		},
		Args: koidatabase.VisibilityJobArgs{
			RunID: runID,
		},
	}

	return store, job
}

func TestVisibilityWorkerCompletesEveryStubOutcome(t *testing.T) {
	store, job := newVisibilityWorkerFixture()

	worker := &visibilityWorker{
		store:   store,
		collect: stub.Collect,
	}

	require.NoError(t, worker.Work(t.Context(), job))
	require.True(t, store.finished)
	require.Len(t, store.saved, 4)
	outcomes := map[koidomain.Outcome]bool{}

	for _, result := range store.saved {
		require.NoError(t, result.Validate())
		outcomes[*result.Outcome] = true
		if *result.Outcome == koidomain.OutcomeCited {
			require.Equal(t, []string{store.run.BrandSnapshot.GitHubRepo}, result.Sources)
		}
	}

	require.Len(t, outcomes, 4)
}

func TestVisibilityWorkerCollectorErrorsRemainPerCell(t *testing.T) {
	store, job := newVisibilityWorkerFixture()

	worker := &visibilityWorker{
		store: store,
		collect: func(context.Context, koidomain.BrandSnapshot, koidomain.VisibilityResult) (koidomain.VisibilityResult, error) {
			return koidomain.VisibilityResult{}, errors.New("collector unavailable")
		},
	}

	require.NoError(t, worker.Work(t.Context(), job))
	require.True(t, store.finished)
	require.Len(t, store.saved, 4)

	for _, result := range store.saved {
		require.Equal(t, koidomain.OutcomeError, *result.Outcome)
		require.Equal(t, "collector_error", *result.ErrorCode)
		require.NotNil(t, result.FetchedAt)
		require.Empty(t, result.Sources)
	}
}

func TestVisibilityWorkerStorageFailureAndCancellation(t *testing.T) {
	for _, attempt := range []int{1, 3} {
		store, job := newVisibilityWorkerFixture()
		job.Attempt = attempt
		store.saveErr = errors.New("database unavailable")

		worker := &visibilityWorker{
			store:   store,
			collect: stub.Collect,
		}

		require.Error(t, worker.Work(t.Context(), job))
		require.False(t, store.finished)
		require.Equal(t, "storage_error", store.code)
		require.Equal(t, attempt == 3, store.terminal)
	}

	store, job := newVisibilityWorkerFixture()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	worker := &visibilityWorker{
		store:   store,
		collect: stub.Collect,
	}

	require.ErrorIs(t, worker.Work(ctx, job), context.Canceled)
	require.Empty(t, store.saved)
	require.Empty(t, store.code)
	store.run = nil
	require.NoError(t, worker.Work(t.Context(), job))
}

func TestVisibilityWorkerTimeouts(t *testing.T) {
	for _, stage := range []string{"begin", "before collection", "collection", "save"} {
		for _, attempt := range []int{1, 3} {
			t.Run(fmt.Sprintf("%s/attempt-%d", stage, attempt), func(t *testing.T) {
				store, job := newVisibilityWorkerFixture()
				job.Attempt = attempt

				worker := &visibilityWorker{
					store:   store,
					collect: stub.Collect,
				}

				ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
				defer cancel()

				switch stage {
				case "begin":
					<-ctx.Done()
					store.beginErr = ctx.Err()
				case "before collection":
					<-ctx.Done()
				case "collection":
					worker.collect = func(ctx context.Context, _ koidomain.BrandSnapshot, _ koidomain.VisibilityResult) (koidomain.VisibilityResult, error) {
						<-ctx.Done()

						return koidomain.VisibilityResult{}, ctx.Err()
					}
				case "save":
					store.save = func(ctx context.Context) error {
						<-ctx.Done()

						return ctx.Err()
					}
				}

				err := worker.Work(ctx, job)
				require.ErrorIs(t, err, context.DeadlineExceeded)
				require.False(t, store.finished)
				require.Empty(t, store.saved)
				if attempt == job.MaxAttempts {
					require.Equal(t, "job_timeout", store.code)
					require.True(t, store.terminal)
					var cancelled *river.JobCancelError
					require.ErrorAs(t, err, &cancelled)
				} else {
					require.Empty(t, store.code)
					require.False(t, store.terminal)
				}
			})
		}
	}
}

func TestVisibilityWorkerTerminalTimeoutCleanup(t *testing.T) {
	store, job := newVisibilityWorkerFixture()
	job.Attempt = job.MaxAttempts
	cleanupErr := errors.New("cleanup unavailable")
	ctx := context.WithValue(t.Context(), struct{}{}, "job context")
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(-time.Second))
	defer cancel()

	store.failure = func(cleanupCtx context.Context) error {
		require.NoError(t, cleanupCtx.Err())
		require.Equal(t, "job context", cleanupCtx.Value(struct{}{}))
		deadline, ok := cleanupCtx.Deadline()
		require.True(t, ok)
		require.WithinDuration(t, time.Now().Add(5*time.Second), deadline, time.Second)

		return cleanupErr
	}

	worker := &visibilityWorker{
		store:   store,
		collect: stub.Collect,
	}

	err := worker.Work(ctx, job)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.ErrorIs(t, err, cleanupErr)
	require.True(t, store.terminal)
	require.Equal(t, "job_timeout", store.code)
}

func TestVisibilityWorkerShutdownDoesNotFinalizeLastAttempt(t *testing.T) {
	store, job := newVisibilityWorkerFixture()
	job.Attempt = job.MaxAttempts
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	worker := &visibilityWorker{
		store:   store,
		collect: stub.Collect,
	}

	require.ErrorIs(t, worker.Work(ctx, job), context.Canceled)
	require.Empty(t, store.code)
	require.False(t, store.finished)
}
