package domain

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
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
	if len(password) < minPasswordLen {
		return uuid.Nil, Validation(FieldError{Field: "password", Code: FieldMinLength})
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

		user.PasswordHash = hash

		return s.UserRepo().Update(ctx, user)
	})
	if err != nil {
		return uuid.Nil, err
	}

	return userID, nil
}
