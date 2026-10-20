package koiapp

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/koihq/koi/pkg/koidatabase"
	"github.com/koihq/koi/pkg/koidomain"
	"github.com/koihq/koi/pkg/koiproviders/cloro"
)

type visibilityStore interface {
	BeginVisibilityAttempt(context.Context, uuid.UUID, int64, int) (*koidomain.VisibilityRun, []koidomain.VisibilityResult, error)
	ContactVisibilityTask(context.Context, uuid.UUID, int, *koidomain.VisibilityResult, time.Time) error
	SaveVisibilityTask(context.Context, uuid.UUID, int, uuid.UUID, string) error
	SaveVisibilityResult(context.Context, uuid.UUID, int, koidomain.VisibilityResult) error
	FinishVisibilityAttempt(context.Context, uuid.UUID, int) error
	RecordVisibilityFailure(context.Context, uuid.UUID, int64, int, string, bool) error
}

type visibilityProvider interface {
	Submit(context.Context, koidomain.VisibilityResult, string) (string, error)
	Poll(context.Context, string, koidomain.EngineID) (*cloro.Answer, error)
}

type visibilityWorker struct {
	river.WorkerDefaults[koidatabase.VisibilityJobArgs]
	store    visibilityStore
	provider visibilityProvider
}

func (w *visibilityWorker) Timeout(*river.Job[koidatabase.VisibilityJobArgs]) time.Duration {
	return time.Minute
}

func (w *visibilityWorker) Work(ctx context.Context, job *river.Job[koidatabase.VisibilityJobArgs]) error {
	run, cells, err := w.store.BeginVisibilityAttempt(ctx, job.Args.RunID, job.ID, job.Attempt)
	if err != nil {
		return w.fail(ctx, job, err)
	}
	if run == nil {
		return nil
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
		var answer *cloro.Answer
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
				failure := cloro.Failure(err)
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
					failure := cloro.Failure(err)
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
		outcome := koidomain.OutcomeError
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
			outcome = koidomain.ScoreVisibility(run.BrandSnapshot, answer.Text, answer.Sources)
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

func (w *visibilityWorker) fail(ctx context.Context, job *river.Job[koidatabase.VisibilityJobArgs], cause error) error {
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
