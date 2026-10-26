package koidomain

import (
	"context"
	"time"

	"github.com/google/uuid"
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

type PageParams struct {
	Page     int
	PageSize int
}

type Page[T any] struct {
	Items       []T   `json:"items"`
	TotalItems  int64 `json:"totalItems"`
	TotalPages  int64 `json:"totalPages"`
	CurrentPage int   `json:"currentPage"`
	HasNext     bool  `json:"hasNext"`
	HasPrev     bool  `json:"hasPrev"`
}
