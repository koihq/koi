package domain

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

type User struct {
	BaseEntity
	Email        string `json:"email" gorm:"uniqueIndex;not null"`
	PasswordHash string `json:"-" gorm:"not null"`
}

func (User) TableName() string { return "users" }

func (u User) Validate() error {
	if !validEmail(u.Email) {
		return Validation(FieldError{Field: "email", Code: FieldInvalid})
	}

	if u.PasswordHash == "" {
		return Validation(FieldError{Field: "password", Code: FieldInvalid})
	}

	return nil
}

func validEmail(email string) bool {
	email = strings.TrimSpace(email)
	at := strings.IndexByte(email, '@')

	return at > 0 && at < len(email)-1
}

type UserRepository interface {
	BaseEntityRepository[User]
	MutableRepository[User]
	GetByEmail(ctx context.Context, email string) (*User, error)
	LockForUpdate(ctx context.Context, id uuid.UUID) error
	// LockAndCount serializes sign-ups until the transaction ends, so two cannot both be the first.
	LockAndCount(ctx context.Context) (int64, error)
}
