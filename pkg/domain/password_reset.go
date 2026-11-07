package domain

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	passwordResetTTL      = time.Hour
	passwordResetTokenLen = 32
)

// PasswordResetToken lets its holder choose a new password once. Only the hash is stored.
type PasswordResetToken struct {
	BaseEntity
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
	User      *User     `json:"-" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	TokenHash string    `gorm:"uniqueIndex;not null"`
	ExpiresAt time.Time `gorm:"not null"`
}

func (PasswordResetToken) TableName() string { return "password_reset_tokens" }

type PasswordResetRepository interface {
	Create(context.Context, *PasswordResetToken) error
	DeleteForUser(context.Context, uuid.UUID) error
	// Consume removes a live token and returns its user, or ErrPasswordResetInvalid.
	Consume(ctx context.Context, tokenHash string, now time.Time) (uuid.UUID, error)
}

// RequestPasswordReset queues the email for any well-formed address. Whether an account exists
// is only looked up by the worker, so the response and its timing say nothing about it.
func (c *authCommander) RequestPasswordReset(ctx context.Context, email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if !validEmail(email) {
		return Validation(FieldError{Field: "email", Code: FieldInvalid})
	}

	return c.store.EnqueuePasswordResetEmail(ctx, email)
}

// IssuePasswordReset replaces the user's earlier token, so only the newest link works.
func (c *authCommander) IssuePasswordReset(ctx context.Context, email string) (*User, string, error) {
	user, err := c.store.UserRepo().GetByEmail(ctx, email)
	if err != nil {
		return nil, "", err
	}

	secret := make([]byte, passwordResetTokenLen)
	if _, err := rand.Read(secret); err != nil {
		return nil, "", err
	}

	raw := base64.RawURLEncoding.EncodeToString(secret)
	token := &PasswordResetToken{
		UserID:    user.ID,
		TokenHash: HashCredential(raw),
		ExpiresAt: time.Now().Add(passwordResetTTL),
	}

	err = c.store.Atomic(ctx, func(s Store) error {
		if err := s.PasswordResetRepo().DeleteForUser(ctx, user.ID); err != nil {
			return err
		}

		return s.PasswordResetRepo().Create(ctx, token)
	})
	if err != nil {
		return nil, "", err
	}

	return user, raw, nil
}

// ResetPassword checks the new password first, so a rejected one does not use up the link.
func (c *authCommander) ResetPassword(ctx context.Context, raw, password string) (uuid.UUID, error) {
	if err := validatePassword(password); err != nil {
		return uuid.Nil, err
	}

	if raw == "" {
		return uuid.Nil, ErrPasswordResetInvalid
	}

	hash, err := HashPassword(password)
	if err != nil {
		return uuid.Nil, err
	}

	var userID uuid.UUID

	err = c.store.Atomic(ctx, func(s Store) error {
		var err error

		userID, err = s.PasswordResetRepo().Consume(ctx, HashCredential(raw), time.Now())
		if err != nil {
			return err
		}

		user, err := s.UserRepo().Get(ctx, userID)
		if errors.Is(err, ErrNotFound) {
			return ErrPasswordResetInvalid
		}
		if err != nil {
			return err
		}

		return replacePassword(ctx, s, user, hash)
	})
	if err != nil {
		return uuid.Nil, err
	}

	// The password has already changed, so a lost notice must not report the reset as failed.
	if err := c.store.EnqueuePasswordChangedEmail(ctx, userID); err != nil {
		slog.Warn("password changed email enqueue failed", "user_id", userID, "err", err)
	}

	return userID, nil
}

func (c *authCommander) SetPassword(ctx context.Context, email, password string) (uuid.UUID, error) {
	if err := validatePassword(password); err != nil {
		return uuid.Nil, err
	}

	hash, err := HashPassword(password)
	if err != nil {
		return uuid.Nil, err
	}

	var userID uuid.UUID

	err = c.store.Atomic(ctx, func(s Store) error {
		user, err := s.UserRepo().GetByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
		if err != nil {
			return err
		}

		userID = user.ID

		return replacePassword(ctx, s, user, hash)
	})
	if err != nil {
		return uuid.Nil, err
	}

	return userID, nil
}

// replacePassword also ends every grant made with the old password. There is no email verification,
// so someone who registered an address first must lose access when its owner recovers the account.
func replacePassword(ctx context.Context, s Store, user *User, hash string) error {
	user.PasswordHash = hash
	if err := s.UserRepo().Update(ctx, user); err != nil {
		return err
	}

	if err := s.PasswordResetRepo().DeleteForUser(ctx, user.ID); err != nil {
		return err
	}

	return s.CredentialRepo().RevokeAllForUser(ctx, user.ID)
}
