package app

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/database"
	"github.com/koihq/koi/pkg/domain"
)

type sentEmail struct {
	id    uuid.UUID
	to    string
	brand string
}

type fakeAccountSender struct {
	sent []sentEmail
	err  error
}

func (f *fakeAccountSender) SendWelcome(_ context.Context, userID uuid.UUID, to string) error {
	f.sent = append(f.sent, sentEmail{id: userID, to: to})

	return f.err
}

func (f *fakeAccountSender) SendPasswordChanged(_ context.Context, userID uuid.UUID, to string) error {
	f.sent = append(f.sent, sentEmail{id: userID, to: to})

	return f.err
}

func (f *fakeAccountSender) SendFirstCheckReady(_ context.Context, runID uuid.UUID, to, brand string) error {
	f.sent = append(f.sent, sentEmail{id: runID, to: to, brand: brand})

	return f.err
}

// TestAccountEmailWorkersReadTheCurrentAddress covers the workers that look a user up when the job
// runs: they mail the address the user has now, and cancel the job when the user is gone.
func TestAccountEmailWorkersReadTheCurrentAddress(t *testing.T) {
	fixture := newAdminFixture(t)
	me := fixture.account(t)

	workers := map[string]func(sender *fakeAccountSender, userID uuid.UUID) error{
		"welcome": func(sender *fakeAccountSender, userID uuid.UUID) error {
			worker := &welcomeWorker{store: fixture.store, sender: sender}

			return worker.Work(t.Context(), &river.Job[database.WelcomeJobArgs]{
				Args: database.WelcomeJobArgs{UserID: userID},
			})
		},
		"password changed": func(sender *fakeAccountSender, userID uuid.UUID) error {
			worker := &passwordChangedWorker{store: fixture.store, sender: sender}

			return worker.Work(t.Context(), &river.Job[database.PasswordChangedJobArgs]{
				Args: database.PasswordChangedJobArgs{UserID: userID},
			})
		},
	}

	for name, work := range workers {
		t.Run(name, func(t *testing.T) {
			sender := &fakeAccountSender{}

			require.NoError(t, work(sender, me.User.ID))
			require.Equal(t, []sentEmail{{id: me.User.ID, to: me.User.Email}}, sender.sent)
		})

		t.Run(name+" for a deleted user is cancelled", func(t *testing.T) {
			sender := &fakeAccountSender{}

			err := work(sender, uuid.New())

			var cancel *river.JobCancelError
			require.ErrorAs(t, err, &cancel)
			require.ErrorIs(t, err, domain.ErrNotFound)
			require.Empty(t, sender.sent)
		})

		t.Run(name+" retries a failed send", func(t *testing.T) {
			failure := errors.New("resend is down")

			require.ErrorIs(t, work(&fakeAccountSender{err: failure}, me.User.ID), failure)
		})
	}
}

type fakeFirstCheckStore struct {
	to, brand string
	answers   int
	err       error
}

func (f fakeFirstCheckStore) FirstCheckEmail(context.Context, uuid.UUID) (string, string, int, error) {
	return f.to, f.brand, f.answers, f.err
}

func TestFirstCheckEmailIsSentOnlyWhenThereIsSomethingToShow(t *testing.T) {
	runID := uuid.New()
	lookupFailure := errors.New("database down")

	for name, tc := range map[string]struct {
		store fakeFirstCheckStore
		sent  []sentEmail
		err   error
	}{
		"answers ready": {
			store: fakeFirstCheckStore{to: "you@example.com", brand: "Acme", answers: 3},
			sent:  []sentEmail{{id: runID, to: "you@example.com", brand: "Acme"}},
		},
		"no answers":     {store: fakeFirstCheckStore{to: "you@example.com", brand: "Acme"}},
		"no address":     {store: fakeFirstCheckStore{brand: "Acme", answers: 3}},
		"lookup failure": {store: fakeFirstCheckStore{err: lookupFailure}, err: lookupFailure},
	} {
		t.Run(name, func(t *testing.T) {
			sender := &fakeAccountSender{}
			worker := &firstCheckEmailWorker{store: tc.store, sender: sender}

			err := worker.Work(t.Context(), &river.Job[database.FirstCheckEmailJobArgs]{
				Args: database.FirstCheckEmailJobArgs{RunID: runID},
			})

			require.ErrorIs(t, err, tc.err)
			require.Equal(t, tc.sent, sender.sent)
		})
	}
}
