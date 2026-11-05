package app

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/koihq/koi/pkg/database"
	"github.com/koihq/koi/pkg/domain"
)

type visibilityStore interface {
	BeginVisibilityAttempt(context.Context, uuid.UUID, int64, int) (*domain.VisibilityRun, []domain.VisibilityResult, error)
	ContactVisibilityTask(context.Context, uuid.UUID, int, *domain.VisibilityResult, time.Time) error
	BeginSyncVisibilityTask(context.Context, uuid.UUID, int, *domain.VisibilityResult, time.Time) error
	SaveVisibilityTask(context.Context, uuid.UUID, int, uuid.UUID, string) error
	SaveVisibilityResult(context.Context, uuid.UUID, int, domain.VisibilityResult) error
	FinishVisibilityAttempt(context.Context, uuid.UUID, int) error
	RecordVisibilityFailure(context.Context, uuid.UUID, int64, int, string, bool) error
}

type visibilityProvider interface {
	Submit(context.Context, domain.VisibilityResult, string) (string, error)
	Poll(context.Context, string, domain.EngineID) (*domain.CollectedAnswer, error)
	Fetch(context.Context, domain.VisibilityResult, string) (*domain.CollectedAnswer, error)
}

type visibilityWorker struct {
	river.WorkerDefaults[database.VisibilityJobArgs]
	store     visibilityStore
	provider  visibilityProvider
	syncSlots chan struct{}
}

func (w *visibilityWorker) Timeout(*river.Job[database.VisibilityJobArgs]) time.Duration {
	return 7 * time.Minute
}

func (w *visibilityWorker) Work(ctx context.Context, job *river.Job[database.VisibilityJobArgs]) error {
	run, cells, err := w.store.BeginVisibilityAttempt(ctx, job.Args.RunID, job.ID, job.Attempt)
	if err != nil {
		return w.fail(ctx, job, err)
	}
	if run == nil {
		return nil
	}
	if run.FastStart {
		return w.workSync(ctx, job, run, cells)
	}

	passStarted := time.Now()
	pending := len(cells)
	accountError := ""
	if run.ErrorCode != nil && (*run.ErrorCode == "cloro_credits" || *run.ErrorCode == "cloro_authentication") {
		accountError = *run.ErrorCode
	}
	for _, cell := range cells {
		if ctx.Err() != nil {
			return w.fail(ctx, job, ctx.Err())
		}
		if time.Since(passStarted) >= 40*time.Second {
			break
		}

		now := time.Now().UTC()
		code := ""
		var answer *domain.CollectedAnswer
		switch {
		case now.Sub(run.CreatedAt) >= 23*time.Hour:
			code = "cloro_deadline"
		case cell.CloroTaskID == nil && accountError != "":
			code = accountError
		default:
			if err := w.store.ContactVisibilityTask(ctx, run.ID, job.Attempt, &cell, now); err != nil {
				return w.fail(ctx, job, err)
			}

			if cell.CloroTaskID == nil {
				id, err := w.provider.Submit(ctx, cell, run.BrandSnapshot.Country)
				if err == nil {
					if err := w.store.SaveVisibilityTask(ctx, run.ID, job.Attempt, cell.ID, id); err != nil {
						return w.fail(ctx, job, err)
					}
					continue
				}

				if ctx.Err() != nil {
					return w.fail(ctx, job, ctx.Err())
				}
				failure := collectionFailure(err)
				if failure.Retryable {
					continue
				}
				code = failure.Code
				if failure.AccountWide {
					accountError = code
					if err := w.store.RecordVisibilityFailure(ctx, run.ID, job.ID, job.Attempt, code, false); err != nil {
						return w.fail(ctx, job, err)
					}
				}
			} else {
				var err error
				answer, err = w.provider.Poll(ctx, *cell.CloroTaskID, cell.Engine)
				if err != nil {
					if ctx.Err() != nil {
						return w.fail(ctx, job, ctx.Err())
					}
					failure := collectionFailure(err)
					if failure.Retryable {
						continue
					}
					code = failure.Code
					if failure.AccountWide {
						accountError = code
						if err := w.store.RecordVisibilityFailure(ctx, run.ID, job.ID, job.Attempt, code, false); err != nil {
							return w.fail(ctx, job, err)
						}
					}
				} else if answer == nil {
					continue
				}
			}
		}

		cell.FetchedAt = &now
		cell.Sources = []string{}
		outcome := domain.OutcomeError
		if code != "" {
			cell.ErrorCode = &code
		} else {
			cell.RawAnswer = answer.Text
			cell.AnswerMarkdown = answer.Markdown
			cell.NamedEntities = answer.Entities
			cell.ProviderCredits = answer.CreditsCharged
			if answer.Sources != nil {
				cell.Sources = answer.Sources
			}
			outcome = domain.ScoreVisibility(run.BrandSnapshot, answer.Text, answer.Sources)
		}
		cell.Outcome = &outcome
		if err := w.store.SaveVisibilityResult(ctx, run.ID, job.Attempt, cell); err != nil {
			return w.fail(ctx, job, err)
		}
		pending--
	}

	if pending > 0 {
		// Snoozing does not consume River attempts. Known task IDs resume after restart.
		return river.JobSnooze(10 * time.Second)
	}
	if err := w.store.FinishVisibilityAttempt(ctx, run.ID, job.Attempt); err != nil {
		return w.fail(ctx, job, err)
	}

	return nil
}

func (w *visibilityWorker) workSync(ctx context.Context, job *river.Job[database.VisibilityJobArgs], run *domain.VisibilityRun, cells []domain.VisibilityResult) error {
	if len(cells) == 0 {
		return w.store.FinishVisibilityAttempt(ctx, run.ID, job.Attempt)
	}
	if len(cells) > 4 {
		cells = cells[:4]
	}

	var group sync.WaitGroup
	errorsFound := make(chan error, len(cells))
	for _, cell := range cells {
		group.Add(1)
		go func() {
			defer group.Done()
			if err := w.syncCell(ctx, job, run, cell); err != nil {
				errorsFound <- err
			}
		}()
	}
	group.Wait()
	close(errorsFound)
	for err := range errorsFound {
		return w.fail(ctx, job, err)
	}
	return river.JobSnooze(10 * time.Second)
}

func (w *visibilityWorker) syncCell(ctx context.Context, job *river.Job[database.VisibilityJobArgs], run *domain.VisibilityRun, cell domain.VisibilityResult) error {
	now := time.Now().UTC()
	code := ""
	var answer *domain.CollectedAnswer
	if now.Sub(run.CreatedAt) >= 23*time.Hour {
		code = "cloro_deadline"
	} else if run.ErrorCode != nil && (*run.ErrorCode == "cloro_credits" || *run.ErrorCode == "cloro_authentication") {
		code = *run.ErrorCode
	} else if cell.SyncAttempts >= 2 {
		code = "cloro_unavailable"
	} else {
		if w.syncSlots != nil {
			select {
			case w.syncSlots <- struct{}{}:
				defer func() { <-w.syncSlots }()
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		if err := w.store.ContactVisibilityTask(ctx, run.ID, job.Attempt, &cell, now); err != nil {
			return err
		}
		if err := w.store.BeginSyncVisibilityTask(ctx, run.ID, job.Attempt, &cell, now); err != nil {
			return err
		}
		var err error
		answer, err = w.provider.Fetch(ctx, cell, run.BrandSnapshot.Country)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			failure := collectionFailure(err)
			if failure.Retryable && cell.SyncAttempts < 2 {
				return nil
			}
			code = failure.Code
			if failure.AccountWide {
				if err := w.store.RecordVisibilityFailure(ctx, run.ID, job.ID, job.Attempt, code, false); err != nil {
					return err
				}
			}
		}
	}

	fetchedAt := time.Now().UTC()
	cell.FetchedAt = &fetchedAt
	cell.Sources = []string{}
	outcome := domain.OutcomeError
	if code != "" {
		cell.ErrorCode = &code
	} else {
		cell.RawAnswer = answer.Text
		cell.AnswerMarkdown = answer.Markdown
		cell.NamedEntities = answer.Entities
		cell.ProviderCredits = answer.CreditsCharged
		if answer.Sources != nil {
			cell.Sources = answer.Sources
		}
		outcome = domain.ScoreVisibility(run.BrandSnapshot, answer.Text, answer.Sources)
	}
	cell.Outcome = &outcome
	return w.store.SaveVisibilityResult(ctx, run.ID, job.Attempt, cell)
}

func (w *visibilityWorker) fail(ctx context.Context, job *river.Job[database.VisibilityJobArgs], cause error) error {
	if errors.Is(ctx.Err(), context.Canceled) {
		return ctx.Err()
	}

	code := "storage_error"
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		if job.Attempt < job.MaxAttempts {
			return errors.Join(cause, ctx.Err())
		}

		cause = errors.Join(cause, ctx.Err())
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		ctx = cleanupCtx
		code = "job_timeout"
	}

	terminal := job.Attempt >= job.MaxAttempts
	if err := w.store.RecordVisibilityFailure(ctx, job.Args.RunID, job.ID, job.Attempt, code, terminal); err != nil {
		return errors.Join(cause, err)
	}

	return cause
}

func collectionFailure(err error) domain.ProviderFailure {
	var classified domain.ProviderError
	if errors.As(err, &classified) {
		return classified.Failure()
	}
	return domain.ProviderFailure{Code: "cloro_unavailable", Retryable: true}
}
