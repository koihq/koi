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

type passwordResetSender interface {
	SendPasswordReset(ctx context.Context, to, token string) error
}

type passwordResetIssuer interface {
	IssuePasswordReset(ctx context.Context, email string) (*domain.User, string, error)
}

// passwordResetWorker creates the token only when sending, so the job table never holds one.
type passwordResetWorker struct {
	river.WorkerDefaults[database.PasswordResetJobArgs]
	issuer passwordResetIssuer
	sender passwordResetSender
}

func (w *passwordResetWorker) Timeout(*river.Job[database.PasswordResetJobArgs]) time.Duration {
	return 30 * time.Second
}

func (w *passwordResetWorker) Work(ctx context.Context, job *river.Job[database.PasswordResetJobArgs]) error {
	user, token, err := w.issuer.IssuePasswordReset(ctx, job.Args.Email)
	if errors.Is(err, domain.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	return w.sender.SendPasswordReset(ctx, user.Email, token)
}

type passwordChangedSender interface {
	SendPasswordChanged(ctx context.Context, userID uuid.UUID, to string) error
}

type passwordChangedWorker struct {
	river.WorkerDefaults[database.PasswordChangedJobArgs]
	store  domain.Store
	sender passwordChangedSender
}

func (w *passwordChangedWorker) Timeout(*river.Job[database.PasswordChangedJobArgs]) time.Duration {
	return 30 * time.Second
}

func (w *passwordChangedWorker) Work(ctx context.Context, job *river.Job[database.PasswordChangedJobArgs]) error {
	user, err := w.store.UserRepo().Get(ctx, job.Args.UserID)
	if errors.Is(err, domain.ErrNotFound) {
		return river.JobCancel(err)
	}
	if err != nil {
		return err
	}

	return w.sender.SendPasswordChanged(ctx, user.ID, user.Email)
}
