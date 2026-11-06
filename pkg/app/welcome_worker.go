package app

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/koihq/koi/pkg/database"
	"github.com/koihq/koi/pkg/domain"
)

type welcomeSender interface {
	SendWelcome(context.Context, uuid.UUID, string) error
}

type welcomeWorker struct {
	river.WorkerDefaults[database.WelcomeJobArgs]
	store  domain.Store
	sender welcomeSender
}

func (w *welcomeWorker) Timeout(*river.Job[database.WelcomeJobArgs]) time.Duration {
	return 30 * time.Second
}

func (w *welcomeWorker) Work(ctx context.Context, job *river.Job[database.WelcomeJobArgs]) error {
	user, err := w.store.UserRepo().Get(ctx, job.Args.UserID)
	if errors.Is(err, domain.ErrNotFound) {
		return river.JobCancel(err)
	}
	if err != nil {
		return err
	}

	return w.sender.SendWelcome(ctx, user.ID, user.Email)
}

type firstCheckEmailStore interface {
	FirstCheckEmail(context.Context, uuid.UUID) (string, string, int, error)
}

type firstCheckSender interface {
	SendFirstCheckReady(ctx context.Context, runID uuid.UUID, to, brand string) error
}

type firstCheckEmailWorker struct {
	river.WorkerDefaults[database.FirstCheckEmailJobArgs]
	store  firstCheckEmailStore
	sender firstCheckSender
}

func (w *firstCheckEmailWorker) Timeout(*river.Job[database.FirstCheckEmailJobArgs]) time.Duration {
	return 30 * time.Second
}

func (w *firstCheckEmailWorker) Work(ctx context.Context, job *river.Job[database.FirstCheckEmailJobArgs]) error {
	to, brand, answers, err := w.store.FirstCheckEmail(ctx, job.Args.RunID)
	if err != nil {
		return err
	}

	// A check with no real answers has nothing to show, so it should not bring anyone back.
	if to == "" || answers == 0 {
		return nil
	}

	return w.sender.SendFirstCheckReady(ctx, job.Args.RunID, to, brand)
}
