package koiapp

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/koihq/koi/pkg/koidatabase"
	"github.com/koihq/koi/pkg/koidomain"
)

type welcomeSender interface {
	SendWelcome(context.Context, uuid.UUID, string) error
}

type welcomeWorker struct {
	river.WorkerDefaults[koidatabase.WelcomeJobArgs]
	store  koidomain.Store
	sender welcomeSender
}

func (w *welcomeWorker) Timeout(*river.Job[koidatabase.WelcomeJobArgs]) time.Duration {
	return 30 * time.Second
}

func (w *welcomeWorker) Work(ctx context.Context, job *river.Job[koidatabase.WelcomeJobArgs]) error {
	user, err := w.store.UserRepo().Get(ctx, job.Args.UserID)
	if errors.Is(err, koidomain.ErrNotFound) {
		return river.JobCancel(err)
	}
	if err != nil {
		return err
	}

	return w.sender.SendWelcome(ctx, user.ID, user.Email)
}
