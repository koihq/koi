package koiapp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/koihq/koi/pkg/koidatabase"
	"github.com/koihq/koi/pkg/koidomain"
	"github.com/koihq/koi/pkg/koiproviders/openrouter"
	"github.com/riverqueue/river"
)

const maxGenerationContextBytes = 128 << 10

type generationStore interface {
	BeginGenerationAttempt(context.Context, uuid.UUID, int64, int) (*koidomain.GenerationRun, koidomain.GenerationInput, error)
	FinishGenerationAttempt(context.Context, uuid.UUID, int, koidomain.GenerationResult) (int, int, error)
	RecordGenerationFailure(context.Context, uuid.UUID, int, string, *time.Time) error
}

type generationProvider interface {
	Generate(context.Context, string, koidomain.GenerationInput) (koidomain.GenerationResult, error)
}

type generationWorker struct {
	river.WorkerDefaults[koidatabase.GenerationJobArgs]
	store    generationStore
	provider generationProvider
}

func retryTime(job *river.Job[koidatabase.GenerationJobArgs]) time.Time {
	base := time.Now().UTC()
	if job.AttemptedAt != nil {
		base = *job.AttemptedAt
	}
	return base.Add(time.Duration(job.Attempt*job.Attempt*5) * time.Second)
}

func (w *generationWorker) NextRetry(job *river.Job[koidatabase.GenerationJobArgs]) time.Time {
	return retryTime(job)
}

func (w *generationWorker) Work(ctx context.Context, job *river.Job[koidatabase.GenerationJobArgs]) error {
	started := time.Now()
	run, input, err := w.store.BeginGenerationAttempt(ctx, job.Args.RunID, job.ID, job.Attempt)
	if err != nil {
		return fmt.Errorf("begin generation: %w", err)
	}
	if run == nil {
		return nil
	}
	var fits bool
	input, fits = openrouter.BoundInput(input, run.ModelSlug, maxGenerationContextBytes)
	if !fits {
		return w.fail(ctx, job, "request_too_large", false, errors.New("brand snapshot exceeds request limit"))
	}
	result, err := w.provider.Generate(ctx, run.ModelSlug, input)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		var providerErr *openrouter.Error
		if errors.As(err, &providerErr) {
			slog.Info("generation provider result", "run_id", run.ID, "attempt", job.Attempt, "status", providerErr.Status, "duration_ms", time.Since(started).Milliseconds())
			return w.fail(ctx, job, providerErr.Code, providerErr.Retryable, err)
		}
		return w.fail(ctx, job, "provider_unavailable", true, err)
	}
	prompts, competitors, err := w.store.FinishGenerationAttempt(ctx, run.ID, job.Attempt, result)
	if err != nil {
		return w.fail(ctx, job, "storage_error", true, err)
	}
	slog.Info("generation completed", "run_id", run.ID, "attempt", job.Attempt, "prompt_count", prompts, "competitor_count", competitors, "duration_ms", time.Since(started).Milliseconds())
	return nil
}

func (w *generationWorker) fail(ctx context.Context, job *river.Job[koidatabase.GenerationJobArgs], code string, retryable bool, cause error) error {
	var next *time.Time
	if retryable && job.Attempt < 3 {
		at := retryTime(job)
		next = &at
	}
	if err := w.store.RecordGenerationFailure(ctx, job.Args.RunID, job.Attempt, code, next); err != nil {
		return errors.Join(cause, err)
	}
	slog.Warn("generation failed", "run_id", job.Args.RunID, "attempt", job.Attempt, "error_code", code, "retry", next != nil)
	if next == nil {
		return river.JobCancel(cause)
	}
	return cause
}
