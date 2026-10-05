package koiapp

import (
	"context"
	"errors"
	"testing"

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
	finished bool
	code     string
	terminal bool
}

func (s *visibilityWorkerStore) BeginVisibilityAttempt(context.Context, uuid.UUID, int64, int) (*koidomain.VisibilityRun, []koidomain.VisibilityResult, error) {
	return s.run, s.pending, nil
}
func (s *visibilityWorkerStore) SaveVisibilityResult(_ context.Context, _ uuid.UUID, _ int, result koidomain.VisibilityResult) error {
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
func (s *visibilityWorkerStore) RecordVisibilityFailure(_ context.Context, _ uuid.UUID, _ int64, _ int, code string, terminal bool) error {
	s.code, s.terminal = code, terminal
	return nil
}

func newVisibilityWorkerFixture() (*visibilityWorkerStore, *river.Job[koidatabase.VisibilityJobArgs]) {
	runID := uuid.New()
	store := &visibilityWorkerStore{
		run: &koidomain.VisibilityRun{
			BaseEntity: koidomain.BaseEntity{ID: runID}, TotalResults: 4,
			BrandSnapshot: koidomain.BrandSnapshot{Name: "Koi", GitHubRepo: "https://github.com/koihq/koi"},
		},
	}
	for _, engine := range koidomain.VisibilityEngines() {
		store.pending = append(store.pending, koidomain.VisibilityResult{
			RunID: runID, PromptID: uuid.New(), PromptText: "Best tools?",
			Engine: engine, Collector: koidomain.CollectorStub, Sources: []string{},
		})
	}
	job := &river.Job[koidatabase.VisibilityJobArgs]{
		JobRow: &rivertype.JobRow{ID: 1, Attempt: 1, MaxAttempts: 3},
		Args:   koidatabase.VisibilityJobArgs{RunID: runID},
	}
	return store, job
}

func TestVisibilityWorkerCompletesEveryStubOutcome(t *testing.T) {
	store, job := newVisibilityWorkerFixture()
	worker := &visibilityWorker{store: store, collect: stub.Collect}
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
		worker := &visibilityWorker{store: store, collect: stub.Collect}
		require.Error(t, worker.Work(t.Context(), job))
		require.False(t, store.finished)
		require.Equal(t, "storage_error", store.code)
		require.Equal(t, attempt == 3, store.terminal)
	}
	store, job := newVisibilityWorkerFixture()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	worker := &visibilityWorker{store: store, collect: stub.Collect}
	require.ErrorIs(t, worker.Work(ctx, job), context.Canceled)
	require.Empty(t, store.saved)
	require.Empty(t, store.code)
	store.run = nil
	require.NoError(t, worker.Work(t.Context(), job))
}
