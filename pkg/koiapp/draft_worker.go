package koiapp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/koihq/koi/pkg/koidatabase"
	"github.com/koihq/koi/pkg/koidomain"
	"github.com/koihq/koi/pkg/koiproviders/openrouter"
	"github.com/riverqueue/river"
)

const maxDraftContextBytes = 128 << 10

type draftWorker struct {
	river.WorkerDefaults[koidatabase.GenerationJobArgs]
	store    *koidatabase.Store
	provider *openrouter.Client
}

func retryTime(job *river.Job[koidatabase.GenerationJobArgs]) time.Time {
	base := time.Now().UTC()
	if job.AttemptedAt != nil {
		base = *job.AttemptedAt
	}
	return base.Add(time.Duration(job.Attempt*job.Attempt*5) * time.Second)
}

func (w *draftWorker) NextRetry(job *river.Job[koidatabase.GenerationJobArgs]) time.Time {
	return retryTime(job)
}

func (w *draftWorker) Work(ctx context.Context, job *river.Job[koidatabase.GenerationJobArgs]) error {
	started := time.Now()
	run, input, err := w.store.BeginDraftAttempt(ctx, job.Args.RunID, job.ID, job.Attempt)
	if err != nil {
		return fmt.Errorf("begin draft: %w", err)
	}
	if run == nil {
		return nil
	}
	if run.PromptVersion != koidomain.GenerationPromptVersion {
		return w.fail(ctx, job, "unsupported_prompt_version", false, errors.New("unsupported draft prompt version"))
	}
	var fits bool
	input, fits = openrouter.BoundInput(input, run.ModelSlug, maxDraftContextBytes)
	if !fits {
		return w.fail(ctx, job, "request_too_large", false, errors.New("brand snapshot exceeds request limit"))
	}
	draft, err := w.provider.Generate(ctx, run.ModelSlug, input)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		var providerErr *openrouter.Error
		if errors.As(err, &providerErr) {
			slog.Info("draft provider result", "run_id", run.ID, "attempt", job.Attempt, "status", providerErr.Status, "duration_ms", time.Since(started).Milliseconds())
			return w.fail(ctx, job, providerErr.Code, providerErr.Retryable, err)
		}
		return w.fail(ctx, job, "provider_unavailable", true, err)
	}
	prompts, competitors, err := w.store.FinishDraftAttempt(ctx, run.ID, job.Attempt, draft)
	if err != nil {
		return w.fail(ctx, job, "storage_error", true, err)
	}
	slog.Info("draft generation completed", "run_id", run.ID, "attempt", job.Attempt, "prompt_count", prompts, "competitor_count", competitors, "duration_ms", time.Since(started).Milliseconds())
	return nil
}

func (w *draftWorker) fail(ctx context.Context, job *river.Job[koidatabase.GenerationJobArgs], code string, retryable bool, cause error) error {
	var next *time.Time
	if retryable && job.Attempt < 3 {
		at := retryTime(job)
		next = &at
	}
	if err := w.store.RecordDraftFailure(ctx, job.Args.RunID, job.Attempt, code, next); err != nil {
		return errors.Join(cause, err)
	}
	slog.Warn("draft generation failed", "run_id", job.Args.RunID, "attempt", job.Attempt, "error_code", code, "retry", next != nil)
	if next == nil {
		return river.JobCancel(cause)
	}
	return cause
}
