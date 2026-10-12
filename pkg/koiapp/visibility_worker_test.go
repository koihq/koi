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
	"github.com/koihq/koi/pkg/koiproviders/cloro"
)

type visibilityWorkerStore struct {
	run                *koidomain.VisibilityRun
	pending            []koidomain.VisibilityResult
	saved              []koidomain.VisibilityResult
	beginErr, saveErr  error
	save, failure      func(context.Context) error
	finished, terminal bool
	code               string
}

func (s *visibilityWorkerStore) BeginVisibilityAttempt(context.Context, uuid.UUID, int64, int) (*koidomain.VisibilityRun, []koidomain.VisibilityResult, error) {
	return s.run, append([]koidomain.VisibilityResult(nil), s.pending...), s.beginErr
}

func (s *visibilityWorkerStore) ContactVisibilityTask(_ context.Context, _ uuid.UUID, _ int, cell *koidomain.VisibilityResult, now time.Time) error {
	cell.CheckedAt = &now
	if cell.SubmittedAt == nil {
		cell.SubmittedAt = &now
	}
	return nil
}

func (s *visibilityWorkerStore) SaveVisibilityTask(_ context.Context, _ uuid.UUID, _ int, cellID uuid.UUID, id string) error {
	for i := range s.pending {
		if s.pending[i].ID == cellID {
			s.pending[i].CloroTaskID = &id
		}
	}
	return nil
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
	for i, cell := range s.pending {
		if cell.ID == result.ID {
			s.pending = append(append([]koidomain.VisibilityResult(nil), s.pending[:i]...), s.pending[i+1:]...)
			break
		}
	}
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

type visibilityTestProvider struct {
	submit func(context.Context, koidomain.VisibilityResult, string) (string, error)
	poll   func(context.Context, string, koidomain.EngineID) (*cloro.Answer, error)
}

func (p visibilityTestProvider) Submit(ctx context.Context, cell koidomain.VisibilityResult, country string) (string, error) {
	if p.submit != nil {
		return p.submit(ctx, cell, country)
	}
	return cell.ID.String(), nil
}

func (p visibilityTestProvider) Poll(ctx context.Context, id string, engine koidomain.EngineID) (*cloro.Answer, error) {
	if p.poll != nil {
		return p.poll(ctx, id, engine)
	}
	return &cloro.Answer{Text: "Koi is recommended", Sources: []string{"https://koi.dev"}}, nil
}

func newVisibilityWorkerFixture() (*visibilityWorkerStore, *river.Job[koidatabase.VisibilityJobArgs]) {
	runID := uuid.New()
	store := &visibilityWorkerStore{
		run: &koidomain.VisibilityRun{
			BaseEntity: koidomain.BaseEntity{
				ID:        runID,
				CreatedAt: time.Now(),
			},
			TotalResults: 4,
			BrandSnapshot: koidomain.BrandSnapshot{
				Name:    "Koi",
				Domain:  "koi.dev",
				Country: "IT",
			},
		},
	}
	for _, engine := range koidomain.VisibilityEngines() {
		store.pending = append(store.pending, koidomain.VisibilityResult{
			BaseEntity: koidomain.BaseEntity{
				ID: uuid.New(),
			},
			RunID:      runID,
			PromptID:   uuid.New(),
			PromptText: "Best tools?",
			Engine:     engine,
			Collector:  koidomain.CollectorCloro,
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

func TestVisibilityWorkerSubmitsThenResumesPolling(t *testing.T) {
	store, job := newVisibilityWorkerFixture()
	submissions := 0
	provider := visibilityTestProvider{
		submit: func(_ context.Context, cell koidomain.VisibilityResult, country string) (string, error) {
			require.NotNil(t, cell.SubmittedAt)
			require.Equal(t, "IT", country)
			submissions++
			return cell.ID.String(), nil
		},
	}
	worker := &visibilityWorker{store: store, provider: provider}
	var snooze *river.JobSnoozeError
	require.ErrorAs(t, worker.Work(t.Context(), job), &snooze)
	require.Equal(t, 10*time.Second, snooze.Duration)
	require.Equal(t, 1, job.Attempt)
	require.Equal(t, 4, submissions)
	require.False(t, store.finished)

	// A new worker uses the persisted task IDs and never resubmits the cells.
	worker = &visibilityWorker{store: store, provider: provider}
	require.NoError(t, worker.Work(t.Context(), job))
	require.Equal(t, 4, submissions)
	require.True(t, store.finished)
	require.Len(t, store.saved, 4)
	for _, result := range store.saved {
		require.NoError(t, result.Validate())
		require.Equal(t, koidomain.OutcomeCited, *result.Outcome)
	}
}

func TestVisibilityWorkerProviderFailuresAndPending(t *testing.T) {
	for _, tc := range []struct {
		name     string
		err      error
		terminal bool
	}{
		{"pending", nil, false},
		{"retryable", &cloro.Error{Code: "cloro_unavailable", Retryable: true}, false},
		{"terminal", &cloro.Error{Code: "cloro_task_failed"}, true},
		{"account", &cloro.Error{Code: "cloro_credits", AccountWide: true}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store, job := newVisibilityWorkerFixture()
			for i := range store.pending {
				id := store.pending[i].ID.String()
				store.pending[i].CloroTaskID = &id
			}
			worker := &visibilityWorker{
				store: store,
				provider: visibilityTestProvider{
					poll: func(context.Context, string, koidomain.EngineID) (*cloro.Answer, error) { return nil, tc.err },
				},
			}
			err := worker.Work(t.Context(), job)
			if tc.terminal {
				require.NoError(t, err)
				require.True(t, store.finished)
				require.Len(t, store.saved, 4)
				for _, cell := range store.saved {
					require.Equal(t, koidomain.OutcomeError, *cell.Outcome)
					require.NoError(t, cell.Validate())
				}
			} else {
				var snooze *river.JobSnoozeError
				require.ErrorAs(t, err, &snooze)
				require.Empty(t, store.saved)
				require.False(t, store.finished)
			}
		})
	}
}

func TestVisibilityWorkerStorageFailureAndTimeout(t *testing.T) {
	for _, attempt := range []int{1, 3} {
		for _, stage := range []string{"storage", "begin", "submit", "poll", "save"} {
			t.Run(fmt.Sprintf("%s/%d", stage, attempt), func(t *testing.T) {
				store, job := newVisibilityWorkerFixture()
				job.Attempt = attempt
				provider := visibilityTestProvider{}
				ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
				defer cancel()
				if stage == "storage" {
					store.beginErr = errors.New("database unavailable")
				} else if stage == "begin" {
					<-ctx.Done()
					store.beginErr = ctx.Err()
				} else if stage == "submit" {
					provider.submit = func(ctx context.Context, _ koidomain.VisibilityResult, _ string) (string, error) {
						<-ctx.Done()
						return "", ctx.Err()
					}
				} else {
					id := "task"
					store.pending[0].CloroTaskID = &id
					if stage == "poll" {
						provider.poll = func(ctx context.Context, _ string, _ koidomain.EngineID) (*cloro.Answer, error) {
							<-ctx.Done()
							return nil, ctx.Err()
						}
					} else {
						store.save = func(ctx context.Context) error {
							<-ctx.Done()

							return ctx.Err()
						}
					}
				}
				worker := &visibilityWorker{store: store, provider: provider}
				err := worker.Work(ctx, job)
				require.Error(t, err)
				require.False(t, store.finished)
				if stage == "storage" {
					require.Equal(t, "storage_error", store.code)
				} else {
					require.ErrorIs(t, err, context.DeadlineExceeded)
					if attempt == job.MaxAttempts {
						require.Equal(t, "job_timeout", store.code)
					} else {
						require.Empty(t, store.code)
					}
				}
			})
		}
	}
}

func TestVisibilityWorkerShutdownDoesNotFinalizeLastAttempt(t *testing.T) {
	store, job := newVisibilityWorkerFixture()
	job.Attempt = job.MaxAttempts
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	worker := &visibilityWorker{
		store:    store,
		provider: visibilityTestProvider{}}
	require.ErrorIs(t, worker.Work(ctx, job), context.Canceled)
	require.Empty(t, store.code)
	require.False(t, store.finished)
}
