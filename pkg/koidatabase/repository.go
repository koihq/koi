package koidatabase

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/koidomain"
)

type Repository[T any] struct {
	db *gorm.DB
}

func NewRepository[T any](db *gorm.DB) *Repository[T] {
	return &Repository[T]{db: db}
}

func (r *Repository[T]) Create(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

func (r *Repository[T]) Get(ctx context.Context, id uuid.UUID) (*T, error) {
	entity := new(T)
	err := r.db.WithContext(ctx).Where("id = ?", id).Take(entity).Error
	if err != nil {
		return nil, mapNotFound(err)
	}
	return entity, nil
}

func mapNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return koidomain.ErrNotFound
	}
	return err
}
