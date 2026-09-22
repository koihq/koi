package koidatabase

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/koidomain"
)

const pgUniqueViolation = "23505"

type Repository[T any] struct {
	db *gorm.DB
}

func NewRepository[T any](db *gorm.DB) *Repository[T] {
	return &Repository[T]{db: db}
}

func (r *Repository[T]) Create(ctx context.Context, entity *T) error {
	if err := r.db.WithContext(ctx).Create(entity).Error; err != nil {
		return mapDBError(err)
	}
	return nil
}

func (r *Repository[T]) Get(ctx context.Context, id uuid.UUID) (*T, error) {
	entity := new(T)
	err := r.db.WithContext(ctx).Where("id = ?", id).Take(entity).Error
	if err != nil {
		return nil, mapDBError(err)
	}
	return entity, nil
}

func paginate[T any](db *gorm.DB, params koidomain.PageParams, order string) (*koidomain.Page[T], error) {
	query := db.Model(new(T))
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, mapDBError(err)
	}
	page := &koidomain.Page[T]{
		Items: make([]T, 0), TotalItems: total, CurrentPage: params.Page, HasPrev: params.Page > 1,
	}
	if total > 0 {
		page.TotalPages = (total-1)/int64(params.PageSize) + 1
	}
	page.HasNext = int64(params.Page) < page.TotalPages
	if int64(params.Page) > page.TotalPages {
		return page, nil
	}
	if err := query.Order(order).Limit(params.PageSize).Offset((params.Page - 1) * params.PageSize).Find(&page.Items).Error; err != nil {
		return nil, mapDBError(err)
	}
	return page, nil
}

func mapDBError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return koidomain.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		if strings.Contains(strings.ToLower(pgErr.ConstraintName), "email") ||
			strings.EqualFold(pgErr.ColumnName, "email") {
			return koidomain.ErrEmailTaken
		}
	}
	return err
}
