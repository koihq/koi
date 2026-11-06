package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/koihq/koi/pkg/database"
	"github.com/koihq/koi/pkg/domain"
	"github.com/koihq/koi/pkg/providers/openrouter"
)

const maxGenerationContextBytes = 128 << 10

type generationStore interface {
	BeginGenerationAttempt(context.Context, uuid.UUID, int64, int) (*domain.GenerationRun, domain.GenerationInput, error)
	FinishGenerationAttempt(context.Context, uuid.UUID, int, domain.GenerationResult) (int, int, error)
	RecordGenerationFailure(context.Context, uuid.UUID, int, string, *time.Time) error
	SaveGenerationSummary(context.Context, uuid.UUID, domain.GenerationSummary) error
	StartTrackingAsOwner(context.Context, uuid.UUID) error
}

type generationProvider interface {
	Generate(context.Context, string, domain.GenerationInput, func(domain.GenerationSummary)) (domain.GenerationResult, error)
}

type generationWorker struct {
	river.WorkerDefaults[database.GenerationJobArgs]
	store    generationStore
	provider generationProvider
	timeout  time.Duration
}

func retryTime(job *river.Job[database.GenerationJobArgs]) time.Time {
	return time.Now().UTC().Add(time.Duration(job.Attempt*job.Attempt*5) * time.Second)
}

func (w *generationWorker) NextRetry(job *river.Job[database.GenerationJobArgs]) time.Time {
	return retryTime(job)
}

func (w *generationWorker) Timeout(*river.Job[database.GenerationJobArgs]) time.Duration {
	return w.timeout
}

func (w *generationWorker) Work(ctx context.Context, job *river.Job[database.GenerationJobArgs]) error {
	started := time.Now()
	run, input, err := w.store.BeginGenerationAttempt(ctx, job.Args.RunID, job.ID, job.Attempt)
	if err != nil {
		if errors.Is(err, domain.ErrBillingRequired) {
			return w.fail(ctx, job, "billing_required", false, err)
		}
		return w.fail(ctx, job, "storage_error", true, fmt.Errorf("begin generation: %w", err))
	}

	if run == nil {
		return nil
	}

	var fits bool
	input, fits = openrouter.BoundInput(input, run.ModelSlug, maxGenerationContextBytes)
	if !fits {
		return w.fail(ctx, job, "request_too_large", false, errors.New("brand snapshot exceeds request limit"))
	}

	result, err := w.provider.Generate(ctx, run.ModelSlug, input, func(summary domain.GenerationSummary) {
		if err := w.store.SaveGenerationSummary(ctx, run.ID, summary); err != nil {
			slog.Warn("generation summary not saved", "run_id", run.ID, "err", err)
		}
	})
	if err != nil {
		if ctx.Err() != nil {
			return w.fail(ctx, job, "job_timeout", true, ctx.Err())
		}

		var classified domain.ProviderError
		if errors.As(err, &classified) {
			providerErr := classified.Failure()
			slog.Info("generation provider result", "run_id", run.ID, "attempt", job.Attempt, "status", providerErr.Status, "duration_ms", time.Since(started).Milliseconds())

			return w.fail(ctx, job, providerErr.Code, providerErr.Retryable, err)
		}

		return w.fail(ctx, job, "provider_unavailable", true, err)
	}

	prompts, competitors, err := w.store.FinishGenerationAttempt(ctx, run.ID, job.Attempt, result)
	if err != nil {
		if errors.Is(err, domain.ErrBillingRequired) {
			return w.fail(ctx, job, "billing_required", false, err)
		}
		return w.fail(ctx, job, "storage_error", true, err)
	}

	slog.Info("generation completed", "run_id", run.ID, "attempt", job.Attempt, "prompt_count", prompts, "competitor_count", competitors, "duration_ms", time.Since(started).Milliseconds())

	// The draft already succeeded, so a failed start must not retry it. The page offers the start.
	if run.StartTracking {
		if err := w.store.StartTrackingAsOwner(ctx, run.BrandID); err != nil {
			slog.Warn("tracking did not start after the draft", "run_id", run.ID, "err", err)
		}
	}

	return nil
}

func (w *generationWorker) fail(ctx context.Context, job *river.Job[database.GenerationJobArgs], code string, retryable bool, cause error) error {
	var next *time.Time
	if errors.Is(ctx.Err(), context.Canceled) {
		return ctx.Err()
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		cause = errors.Join(cause, ctx.Err())
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		ctx = cleanupCtx
		code = "job_timeout"
	}

	if retryable && job.Attempt < job.MaxAttempts {
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
