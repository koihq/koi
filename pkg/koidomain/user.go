package koidomain

import (
	"context"
	"fmt"
	"strings"
)

type User struct {
	BaseEntity
	Email        string `json:"email" gorm:"uniqueIndex;not null"`
	PasswordHash string `json:"-" gorm:"not null"`
}

func (User) TableName() string { return "users" }

func (u User) Validate() error {
	if !validEmail(u.Email) {
		return fmt.Errorf("%w: email", ErrValidation)
	}
	if u.PasswordHash == "" {
		return fmt.Errorf("%w: password", ErrValidation)
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
	GetByEmail(ctx context.Context, email string) (*User, error)
}
