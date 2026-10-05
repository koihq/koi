package koiapp

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/koihq/koi/pkg/koidatabase"
	"github.com/koihq/koi/pkg/koidomain"
)

type visibilityStore interface {
	BeginVisibilityAttempt(context.Context, uuid.UUID, int64, int) (*koidomain.VisibilityRun, []koidomain.VisibilityResult, error)
	SaveVisibilityResult(context.Context, uuid.UUID, int, koidomain.VisibilityResult) error
	FinishVisibilityAttempt(context.Context, uuid.UUID, int) error
	RecordVisibilityFailure(context.Context, uuid.UUID, int64, int, string, bool) error
}

type visibilityWorker struct {
	river.WorkerDefaults[koidatabase.VisibilityJobArgs]
	store   visibilityStore
	collect func(context.Context, koidomain.BrandSnapshot, koidomain.VisibilityResult) (koidomain.VisibilityResult, error)
}

func (w *visibilityWorker) Work(ctx context.Context, job *river.Job[koidatabase.VisibilityJobArgs]) error {
	run, results, err := w.store.BeginVisibilityAttempt(ctx, job.Args.RunID, job.ID, job.Attempt)
	if err != nil {
		return w.fail(ctx, job, err)
	}
	if run == nil {
		return nil
	}
	for _, pending := range results {
		if err := ctx.Err(); err != nil {
			return err
		}
		answer, err := w.collect(ctx, run.BrandSnapshot, pending)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// Copy only answer fields: collectors cannot change persisted prompt/engine identity.
		result := pending
		result.Outcome, result.RawAnswer, result.Sources, result.ErrorCode = answer.Outcome, answer.RawAnswer, answer.Sources, answer.ErrorCode
		if err != nil {
			outcome, code := koidomain.OutcomeError, "collector_error"
			result.Outcome, result.ErrorCode = &outcome, &code
			result.RawAnswer, result.Sources = "", []string{}
		}
		if result.Sources == nil {
			result.Sources = []string{}
		}
		now := time.Now().UTC()
		result.FetchedAt = &now
		if err := w.store.SaveVisibilityResult(ctx, run.ID, job.Attempt, result); err != nil {
			return w.fail(ctx, job, err)
		}
	}
	if err := w.store.FinishVisibilityAttempt(ctx, run.ID, job.Attempt); err != nil {
		return w.fail(ctx, job, err)
	}
	slog.InfoContext(ctx, "visibility completed", "run_id", run.ID, "attempt", job.Attempt, "results", run.TotalResults)
	return nil
}

func (w *visibilityWorker) fail(ctx context.Context, job *river.Job[koidatabase.VisibilityJobArgs], cause error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	terminal := job.Attempt >= job.MaxAttempts
	if err := w.store.RecordVisibilityFailure(ctx, job.Args.RunID, job.ID, job.Attempt, "storage_error", terminal); err != nil {
		return errors.Join(cause, err)
	}
	slog.WarnContext(ctx, "visibility failed", "run_id", job.Args.RunID, "attempt", job.Attempt, "terminal", terminal)
	if terminal {
		return river.JobCancel(cause)
	}
	return cause
}
