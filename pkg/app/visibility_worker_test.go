package app

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/database"
	"github.com/koihq/koi/pkg/domain"
	"github.com/koihq/koi/pkg/providers/cloro"
)

type visibilityWorkerStore struct {
	mu                 sync.Mutex
	run                *domain.VisibilityRun
	pending            []domain.VisibilityResult
	saved              []domain.VisibilityResult
	beginErr, saveErr  error
	save, failure      func(context.Context) error
	finished, terminal bool
	code               string
}

func (s *visibilityWorkerStore) BeginVisibilityAttempt(context.Context, uuid.UUID, int64, int) (*domain.VisibilityRun, []domain.VisibilityResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.run, append([]domain.VisibilityResult(nil), s.pending...), s.beginErr
}

func (s *visibilityWorkerStore) ContactVisibilityTask(_ context.Context, _ uuid.UUID, _ int, cell *domain.VisibilityResult, now time.Time) error {
	cell.CheckedAt = &now
	if cell.SubmittedAt == nil {
		cell.SubmittedAt = &now
	}
	return nil
}

func (s *visibilityWorkerStore) BeginSyncVisibilityTask(_ context.Context, _ uuid.UUID, _ int, cell *domain.VisibilityResult, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cell.SyncAttempts++
	cell.CheckedAt = &now
	cell.SubmittedAt = &now
	for i := range s.pending {
		if s.pending[i].ID == cell.ID {
			s.pending[i].SyncAttempts = cell.SyncAttempts
		}
	}
	return nil
}

func (s *visibilityWorkerStore) SaveVisibilityTask(_ context.Context, _ uuid.UUID, _ int, cellID uuid.UUID, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.pending {
		if s.pending[i].ID == cellID {
			s.pending[i].CloroTaskID = &id
		}
	}
	return nil
}

func (s *visibilityWorkerStore) SaveVisibilityResult(ctx context.Context, _ uuid.UUID, _ int, result domain.VisibilityResult) error {
	if s.save != nil {
		if err := s.save(ctx); err != nil {
			return err
		}
	}
	if s.saveErr != nil {
		return s.saveErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.saved = append(s.saved, result)
	for i, cell := range s.pending {
		if cell.ID == result.ID {
			s.pending = append(append([]domain.VisibilityResult(nil), s.pending[:i]...), s.pending[i+1:]...)
			break
		}
	}
	return nil
}

func (s *visibilityWorkerStore) EnqueueFirstCheckEmail(context.Context, uuid.UUID) error {
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
	submit func(context.Context, domain.VisibilityResult, string) (string, error)
	poll   func(context.Context, string, domain.EngineID) (*cloro.Answer, error)
	fetch  func(context.Context, domain.VisibilityResult, string) (*cloro.Answer, error)
}

func (p visibilityTestProvider) Fetch(ctx context.Context, cell domain.VisibilityResult, country string) (*cloro.Answer, error) {
	if p.fetch != nil {
		return p.fetch(ctx, cell, country)
	}
	return p.Poll(ctx, "", cell.Engine)
}

func (p visibilityTestProvider) Submit(ctx context.Context, cell domain.VisibilityResult, country string) (string, error) {
	if p.submit != nil {
		return p.submit(ctx, cell, country)
	}
	return cell.ID.String(), nil
}

func (p visibilityTestProvider) Poll(ctx context.Context, id string, engine domain.EngineID) (*cloro.Answer, error) {
	if p.poll != nil {
		return p.poll(ctx, id, engine)
	}
	return &cloro.Answer{
		Text: "Koi is recommended", Sources: []string{"https://koi.dev"},
		Markdown: "1. [Acme](https://acme.dev)",
		Entities: []domain.NamedEntity{{Type: "software", Name: "Acme"}},
	}, nil
}

func newVisibilityWorkerFixture() (*visibilityWorkerStore, *river.Job[database.VisibilityJobArgs]) {
	runID := uuid.New()
	brandID := uuid.New()
	createdAt := time.Now()
	store := &visibilityWorkerStore{
		run: &domain.VisibilityRun{
			BaseEntity: domain.BaseEntity{
				ID:        runID,
				CreatedAt: createdAt,
			},
			BrandID:      brandID,
			TotalResults: 4,
			BrandSnapshot: domain.BrandSnapshot{
				Name:    "Koi",
				Domain:  "koi.dev",
				Country: "IT",
			},
		},
	}
	for _, engine := range domain.VisibilityEngines() {
		store.pending = append(store.pending, domain.VisibilityResult{
			BaseEntity: domain.BaseEntity{
				ID: uuid.New(),
			},
			BrandID:      brandID,
			RunCreatedAt: createdAt,
			RunID:        runID,
			PromptID:     uuid.New(),
			PromptText:   "Best tools?",
			Engine:       engine,
			Collector:    domain.CollectorCloro,
			Sources:      []string{},
		})
	}
	job := &river.Job[database.VisibilityJobArgs]{
		JobRow: &rivertype.JobRow{
			ID:          1,
			Attempt:     1,
			MaxAttempts: 3,
		},
		Args: database.VisibilityJobArgs{
			RunID: runID,
		},
	}
	return store, job
}

func TestVisibilityWorkerSubmitsThenResumesPolling(t *testing.T) {
	store, job := newVisibilityWorkerFixture()
	submissions := 0
	provider := visibilityTestProvider{
		submit: func(_ context.Context, cell domain.VisibilityResult, country string) (string, error) {
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
		require.Equal(t, domain.OutcomeCited, *result.Outcome)
		require.Equal(t, "1. [Acme](https://acme.dev)", result.AnswerMarkdown)
		require.Equal(t, []domain.NamedEntity{{Type: "software", Name: "Acme"}}, result.NamedEntities)
	}
}

func TestVisibilityWorkerFastStartRetriesOnce(t *testing.T) {
	store, job := newVisibilityWorkerFixture()
	store.run.FastStart = true
	var attempts atomic.Int32
	worker := &visibilityWorker{store: store, provider: visibilityTestProvider{
		fetch: func(context.Context, domain.VisibilityResult, string) (*cloro.Answer, error) {
			attempts.Add(1)
			return nil, &cloro.Error{Code: "cloro_unavailable", Retryable: true}
		},
	}}
	var snooze *river.JobSnoozeError
	require.ErrorAs(t, worker.Work(t.Context(), job), &snooze)
	require.Equal(t, int32(4), attempts.Load())
	require.Empty(t, store.saved)
	require.ErrorAs(t, worker.Work(t.Context(), job), &snooze)
	require.Equal(t, int32(8), attempts.Load())
	require.Len(t, store.saved, 4)
	require.NoError(t, worker.Work(t.Context(), job))
	require.True(t, store.finished)
	require.Equal(t, int32(8), attempts.Load())
}

func TestVisibilityWorkerFastStartSavesAnswers(t *testing.T) {
	store, job := newVisibilityWorkerFixture()
	store.run.FastStart = true
	worker := &visibilityWorker{store: store, provider: visibilityTestProvider{}}
	var snooze *river.JobSnoozeError
	require.ErrorAs(t, worker.Work(t.Context(), job), &snooze)
	require.Len(t, store.saved, 4)
	for _, cell := range store.saved {
		require.Equal(t, domain.OutcomeCited, *cell.Outcome)
		require.Equal(t, 1, cell.SyncAttempts)
	}
	require.NoError(t, worker.Work(t.Context(), job))
	require.True(t, store.finished)
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
					poll: func(context.Context, string, domain.EngineID) (*cloro.Answer, error) { return nil, tc.err },
				},
			}
			err := worker.Work(t.Context(), job)
			if tc.terminal {
				require.NoError(t, err)
				require.True(t, store.finished)
				require.Len(t, store.saved, 4)
				for _, cell := range store.saved {
					require.Equal(t, domain.OutcomeError, *cell.Outcome)
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
					provider.submit = func(ctx context.Context, _ domain.VisibilityResult, _ string) (string, error) {
						<-ctx.Done()
						return "", ctx.Err()
					}
				} else {
					id := "task"
					store.pending[0].CloroTaskID = &id
					if stage == "poll" {
						provider.poll = func(ctx context.Context, _ string, _ domain.EngineID) (*cloro.Answer, error) {
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

type mentionJudgeFunc func(context.Context, domain.MentionInput) (domain.MentionJudgment, error)

func (f mentionJudgeFunc) JudgeMention(ctx context.Context, input domain.MentionInput) (domain.MentionJudgment, error) {
	return f(ctx, input)
}

func namedBrandProvider(text string, sources ...string) visibilityTestProvider {
	return visibilityTestProvider{
		poll: func(context.Context, string, domain.EngineID) (*cloro.Answer, error) {
			return &cloro.Answer{
				Text:    text,
				Sources: sources,
			}, nil
		},
	}
}

// finishVisibilityWork runs passes until the run is done, as River would after each snooze.
func finishVisibilityWork(t *testing.T, worker *visibilityWorker, store *visibilityWorkerStore, job *river.Job[database.VisibilityJobArgs]) {
	t.Helper()

	for range 4 {
		var snooze *river.JobSnoozeError
		err := worker.Work(t.Context(), job)
		if err != nil {
			require.ErrorAs(t, err, &snooze)
		}
		if store.finished {
			return
		}
	}

	t.Fatal("visibility run did not finish")
}

func TestVisibilityWorkerJudgesNamedBrandOnly(t *testing.T) {
	cost := 0.0002
	for _, fastStart := range []bool{false, true} {
		for _, test := range []struct {
			name        string
			answer      string
			sources     []string
			verdict     domain.MentionVerdict
			judgeErr    error
			wantCalls   int32
			wantOutcome domain.Outcome
			wantVerdict *domain.MentionVerdict
			wantCost    *float64
		}{
			{
				name:        "real mention stays mentioned",
				answer:      "Koi tracks AI visibility.",
				verdict:     domain.MentionVerdictMention,
				wantCalls:   4,
				wantOutcome: domain.OutcomeMentioned,
				wantVerdict: new(domain.MentionVerdictMention),
				wantCost:    &cost,
			},
			{
				name:        "unknown brand becomes a miss",
				answer:      "I couldn't reliably identify Koi.",
				verdict:     domain.MentionVerdictNoInfo,
				wantCalls:   4,
				wantOutcome: domain.OutcomeMiss,
				wantVerdict: new(domain.MentionVerdictNoInfo),
				wantCost:    &cost,
			},
			{
				name:        "other company becomes a miss",
				answer:      "Koi Security protects endpoints.",
				verdict:     domain.MentionVerdictOtherCompany,
				wantCalls:   4,
				wantOutcome: domain.OutcomeMiss,
				wantVerdict: new(domain.MentionVerdictOtherCompany),
				wantCost:    &cost,
			},
			{
				name:        "other meaning becomes a miss",
				answer:      "Koi are ornamental carp.",
				verdict:     domain.MentionVerdictOtherMeaning,
				wantCalls:   4,
				wantOutcome: domain.OutcomeMiss,
				wantVerdict: new(domain.MentionVerdictOtherMeaning),
				wantCost:    &cost,
			},
			{
				name:        "citation skips the judge",
				answer:      "Koi tracks AI visibility.",
				sources:     []string{"https://koi.dev"},
				wantOutcome: domain.OutcomeCited,
			},
			{
				name:        "absent name skips the judge",
				answer:      "Try Elmo.",
				wantOutcome: domain.OutcomeMiss,
			},
			{
				name:        "failed judgment keeps the name match",
				answer:      "Koi tracks AI visibility.",
				judgeErr:    errors.New("provider unavailable"),
				wantCalls:   4,
				wantOutcome: domain.OutcomeMentioned,
			},
		} {
			t.Run(fmt.Sprintf("%s fastStart=%t", test.name, fastStart), func(t *testing.T) {
				store, job := newVisibilityWorkerFixture()
				store.run.FastStart = fastStart

				var calls atomic.Int32
				worker := &visibilityWorker{
					store:    store,
					provider: namedBrandProvider(test.answer, test.sources...),
					judge: mentionJudgeFunc(func(_ context.Context, input domain.MentionInput) (domain.MentionJudgment, error) {
						calls.Add(1)
						require.Equal(t, "Koi", input.Brand.Name)
						require.Equal(t, "Best tools?", input.Prompt)
						require.Equal(t, test.answer, input.Answer)

						if test.judgeErr != nil {
							return domain.MentionJudgment{}, test.judgeErr
						}

						return domain.MentionJudgment{
							Verdict: test.verdict,
							CostUSD: &cost,
						}, nil
					}),
				}

				finishVisibilityWork(t, worker, store, job)

				require.Equal(t, test.wantCalls, calls.Load())
				require.Len(t, store.saved, 4)
				for _, result := range store.saved {
					require.NoError(t, result.Validate())
					require.Equal(t, test.wantOutcome, *result.Outcome)
					require.Equal(t, test.wantVerdict, result.MentionVerdict)
					require.Equal(t, test.wantCost, result.MentionCostUSD)
				}
			})
		}
	}
}

func TestVisibilityWorkerWithoutJudgeKeepsNameMatch(t *testing.T) {
	store, job := newVisibilityWorkerFixture()
	worker := &visibilityWorker{
		store:    store,
		provider: namedBrandProvider("I couldn't reliably identify Koi."),
	}

	finishVisibilityWork(t, worker, store, job)

	require.Len(t, store.saved, 4)
	for _, result := range store.saved {
		require.Equal(t, domain.OutcomeMentioned, *result.Outcome)
		require.Nil(t, result.MentionVerdict)
	}
}

func TestVisibilityWorkerCancelledJudgmentDoesNotSave(t *testing.T) {
	for _, fastStart := range []bool{false, true} {
		t.Run(fmt.Sprintf("fastStart=%t", fastStart), func(t *testing.T) {
			store, job := newVisibilityWorkerFixture()
			store.run.FastStart = fastStart

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			worker := &visibilityWorker{
				store:    store,
				provider: namedBrandProvider("Koi tracks AI visibility."),
				judge: mentionJudgeFunc(func(context.Context, domain.MentionInput) (domain.MentionJudgment, error) {
					cancel()

					return domain.MentionJudgment{}, errors.New("request aborted")
				}),
			}

			if !fastStart {
				var snooze *river.JobSnoozeError
				require.ErrorAs(t, worker.Work(ctx, job), &snooze)
			}

			require.ErrorIs(t, worker.Work(ctx, job), context.Canceled)
			require.Empty(t, store.saved)
			require.False(t, store.finished)
		})
	}
}
