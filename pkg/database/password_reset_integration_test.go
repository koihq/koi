package database

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/internal/testdb"
	"github.com/koihq/koi/pkg/domain"
)

func TestPasswordResetFlow(t *testing.T) {
	db, err := Open(testdb.URL(t))
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

	require.NoError(t, Migrate(db))

	store := NewStore(db)
	ctx := t.Context()
	auth := domain.NewAuthCommander(store, domain.AuthOptions{OpenSignup: true})
	const email = "you@example.com"

	me, err := auth.Signup(ctx, domain.SignupParams{Email: email, Password: "old-password"})
	require.NoError(t, err)

	t.Run("unknown email is not found and creates nothing", func(t *testing.T) {
		_, _, err := auth.IssuePasswordReset(ctx, "nobody@example.com")
		require.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("stores only the hash", func(t *testing.T) {
		_, raw, err := auth.IssuePasswordReset(ctx, email)
		require.NoError(t, err)

		var stored domain.PasswordResetToken
		require.NoError(t, db.Where("user_id = ?", me.User.ID).Take(&stored).Error)
		require.NotEqual(t, raw, stored.TokenHash)
		require.Equal(t, domain.HashCredential(raw), stored.TokenHash)
	})

	t.Run("a short password leaves the link usable", func(t *testing.T) {
		_, raw, err := auth.IssuePasswordReset(ctx, email)
		require.NoError(t, err)

		_, err = auth.ResetPassword(ctx, raw, "short")
		require.ErrorIs(t, err, domain.ErrValidation)

		userID, err := auth.ResetPassword(ctx, raw, "new-password-1")
		require.NoError(t, err)
		require.Equal(t, me.User.ID, userID)
	})

	t.Run("changes the password and works once", func(t *testing.T) {
		_, raw, err := auth.IssuePasswordReset(ctx, email)
		require.NoError(t, err)

		_, err = auth.ResetPassword(ctx, raw, "new-password-2")
		require.NoError(t, err)

		_, err = auth.Login(ctx, email, "new-password-1")
		require.ErrorIs(t, err, domain.ErrInvalidCredentials)

		_, err = auth.Login(ctx, email, "new-password-2")
		require.NoError(t, err)

		_, err = auth.ResetPassword(ctx, raw, "new-password-3")
		require.ErrorIs(t, err, domain.ErrPasswordResetInvalid)
	})

	t.Run("a newer link replaces the older one", func(t *testing.T) {
		_, older, err := auth.IssuePasswordReset(ctx, email)
		require.NoError(t, err)

		_, newer, err := auth.IssuePasswordReset(ctx, email)
		require.NoError(t, err)

		_, err = auth.ResetPassword(ctx, older, "new-password-4")
		require.ErrorIs(t, err, domain.ErrPasswordResetInvalid)

		_, err = auth.ResetPassword(ctx, newer, "new-password-4")
		require.NoError(t, err)
	})

	t.Run("an expired link is rejected", func(t *testing.T) {
		_, raw, err := auth.IssuePasswordReset(ctx, email)
		require.NoError(t, err)

		require.NoError(t, db.Model(&domain.PasswordResetToken{}).
			Where("user_id = ?", me.User.ID).
			Update("expires_at", time.Now().Add(-time.Minute)).Error)

		_, err = auth.ResetPassword(ctx, raw, "new-password-5")
		require.ErrorIs(t, err, domain.ErrPasswordResetInvalid)

		_, err = auth.Login(ctx, email, "new-password-4")
		require.NoError(t, err)
	})

	t.Run("without email configured the request is unavailable", func(t *testing.T) {
		require.ErrorIs(t, auth.RequestPasswordReset(ctx, email), domain.ErrFeatureUnavailable)
	})

	t.Run("a malformed address is a validation error", func(t *testing.T) {
		require.ErrorIs(t, auth.RequestPasswordReset(ctx, "not-an-email"), domain.ErrValidation)
	})
}
