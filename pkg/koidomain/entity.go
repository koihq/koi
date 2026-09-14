package koidomain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound           = errors.New("not found")
	ErrEmailTaken         = errors.New("email taken")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrValidation         = errors.New("validation")
)

type Entity interface {
	GetID() uuid.UUID
}

type BaseEntity struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	CreatedAt time.Time `json:"-" gorm:"not null;autoCreateTime"`
	UpdatedAt time.Time `json:"-" gorm:"not null;autoUpdateTime"`
}

func (b BaseEntity) GetID() uuid.UUID { return b.ID }

type BaseEntityQuerier[T Entity] interface {
	Get(ctx context.Context, id uuid.UUID) (*T, error)
}

type BaseEntityRepository[T Entity] interface {
	BaseEntityQuerier[T]
	Create(ctx context.Context, entity *T) error
}
