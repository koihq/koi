package app

import (
	"context"
	"errors"
	"testing"

	"github.com/riverqueue/river"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/database"
	"github.com/koihq/koi/pkg/domain"
)

type fakeResetIssuer struct {
	user  *domain.User
	token string
	err   error
}

func (f fakeResetIssuer) IssuePasswordReset(context.Context, string) (*domain.User, string, error) {
	return f.user, f.token, f.err
}

type fakeResetSender struct {
	to, token string
	err       error
}

func (f *fakeResetSender) SendPasswordReset(_ context.Context, to, token string) error {
	f.to, f.token = to, token

	return f.err
}

func resetJob() *river.Job[database.PasswordResetJobArgs] {
	return &river.Job[database.PasswordResetJobArgs]{
		Args: database.PasswordResetJobArgs{Email: "you@example.com"},
	}
}

func TestPasswordResetWorkerSendsTheIssuedToken(t *testing.T) {
	sender := &fakeResetSender{}
	worker := &passwordResetWorker{
		issuer: fakeResetIssuer{user: &domain.User{Email: "you@example.com"}, token: "raw"},
		sender: sender,
	}

	require.NoError(t, worker.Work(t.Context(), resetJob()))
	require.Equal(t, "you@example.com", sender.to)
	require.Equal(t, "raw", sender.token)
}

func TestPasswordResetWorkerIgnoresUnknownEmail(t *testing.T) {
	sender := &fakeResetSender{}
	worker := &passwordResetWorker{
		issuer: fakeResetIssuer{err: domain.ErrNotFound},
		sender: sender,
	}

	require.NoError(t, worker.Work(t.Context(), resetJob()))
	require.Empty(t, sender.to)
}

func TestPasswordResetWorkerRetriesFailures(t *testing.T) {
	failure := errors.New("resend is down")
	worker := &passwordResetWorker{
		issuer: fakeResetIssuer{user: &domain.User{Email: "you@example.com"}, token: "raw"},
		sender: &fakeResetSender{err: failure},
	}

	require.ErrorIs(t, worker.Work(t.Context(), resetJob()), failure)
}
