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
			return ctx.Err()
		}
		if time.Since(passStarted) >= 40*time.Second {
			break
		}

		now := time.Now().UTC()
		code := ""
		var answer *cloro.Answer
		switch {
		case cell.Collector != koidomain.CollectorCloro:
			code = "sample_run_retired"
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
	terminal := job.Attempt >= job.MaxAttempts
	if err := w.store.RecordVisibilityFailure(ctx, job.Args.RunID, job.ID, job.Attempt, "storage_error", terminal); err != nil {
		return errors.Join(cause, err)
	}

	return cause
}

type visibilityDispatchArgs struct{}

func (visibilityDispatchArgs) Kind() string { return "koi_visibility_dispatch" }

type visibilityDispatchWorker struct {
	river.WorkerDefaults[visibilityDispatchArgs]
	store *koidatabase.Store
}

func (w *visibilityDispatchWorker) Work(ctx context.Context, _ *river.Job[visibilityDispatchArgs]) error {
	if err := w.store.ReconcileVisibilityRuns(ctx); err != nil {
		return err
	}

	return w.store.DispatchVisibility(ctx, time.Now().UTC())
}
