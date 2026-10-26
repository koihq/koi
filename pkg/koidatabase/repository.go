package koidatabase

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/koiauth"
	"github.com/koihq/koi/pkg/koidomain"
)

const pgUniqueViolation = "23505"

type Repository[T any] struct {
	db    *gorm.DB
	scope func(koiauth.Identity, *gorm.DB) *gorm.DB
}

func NewRepository[T any](db *gorm.DB) *Repository[T] {
	return &Repository[T]{db: db}
}

func newScopedRepository[T any](db *gorm.DB, scope func(koiauth.Identity, *gorm.DB) *gorm.DB) *Repository[T] {
	return &Repository[T]{db: db, scope: scope}
}

func (r *Repository[T]) query(ctx context.Context) (*gorm.DB, error) {
	db := r.db.WithContext(ctx)
	if r.scope == nil {
		return db, nil
	}

	identity, ok := koiauth.FromContext(ctx)
	if !ok || identity.OrganizationID == uuid.Nil {
		return nil, koidomain.ErrUnauthorized
	}

	return r.scope(identity, db), nil
}

func brandOwnerScope(identity koiauth.Identity, db *gorm.DB) *gorm.DB {
	return db.Where("brands.organization_id = ?", identity.OrganizationID)
}

func childOwnerScope(table string) func(koiauth.Identity, *gorm.DB) *gorm.DB {
	return func(identity koiauth.Identity, db *gorm.DB) *gorm.DB {
		return db.Where("EXISTS (SELECT 1 FROM brands WHERE brands.id = "+table+".brand_id AND brands.organization_id = ?)", identity.OrganizationID)
	}
}

func (r *Repository[T]) Create(ctx context.Context, entity *T) error {
	if err := r.db.WithContext(ctx).Create(entity).Error; err != nil {
		return mapDBError(err)
	}

	return nil
}

func (r *Repository[T]) Get(ctx context.Context, id uuid.UUID) (*T, error) {
	return r.getBy(ctx, func(db *gorm.DB) *gorm.DB { return db.Where("id = ?", id) })
}

func (r *Repository[T]) getBy(ctx context.Context, filter func(*gorm.DB) *gorm.DB) (*T, error) {
	db, err := r.query(ctx)
	if err != nil {
		return nil, err
	}

	entity := new(T)
	err = filter(db).Take(entity).Error
	if err != nil {
		return nil, mapDBError(err)
	}

	return entity, nil
}

func (r *Repository[T]) listBy(ctx context.Context, filter func(*gorm.DB) *gorm.DB, params koidomain.PageParams, order string) (*koidomain.Page[T], error) {
	db, err := r.query(ctx)
	if err != nil {
		return nil, err
	}

	return paginate[T](filter(db), params, order)
}

func (r *Repository[T]) updateBy(ctx context.Context, filter func(*gorm.DB) *gorm.DB, values any, columns []string, requireMatch bool) error {
	db, err := r.query(ctx)
	if err != nil {
		return err
	}

	return updated(filter(db.Model(new(T))).Select(columns).Updates(values), requireMatch)
}

func (r *Repository[T]) deleteBy(ctx context.Context, filter func(*gorm.DB) *gorm.DB, requireMatch bool) error {
	db, err := r.query(ctx)
	if err != nil {
		return err
	}

	return updated(filter(db).Delete(new(T)), requireMatch)
}

func (r *Repository[T]) Count(ctx context.Context) (int64, error) {
	db, err := r.query(ctx)
	if err != nil {
		return 0, err
	}

	var count int64
	err = db.Model(new(T)).Count(&count).Error

	return count, mapDBError(err)
}

func paginate[T any](db *gorm.DB, params koidomain.PageParams, order string) (*koidomain.Page[T], error) {
	query := db.Model(new(T))
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, mapDBError(err)
	}

	page := &koidomain.Page[T]{
		Items:       make([]T, 0),
		TotalItems:  total,
		CurrentPage: params.Page,
		HasPrev:     params.Page > 1,
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
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		switch pgErr.ConstraintName {
		case "fk_brands_organization", "fk_memberships_user", "fk_memberships_organization", "fk_billing_subscriptions_organization", "fk_billing_checkout_sessions_organization", "fk_billing_change_requests_organization":
			if strings.Contains(pgErr.Detail, "still referenced") {
				return koidomain.ErrResourceConflict
			}
			return koidomain.ErrNotFound
		}
	}
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		if pgErr.ConstraintName == "idx_prompts_brand_text" {
			return koidomain.Validation(koidomain.FieldError{Field: "text", Code: koidomain.FieldTaken})
		}

		if pgErr.ConstraintName == "idx_competitors_brand_domain" {
			return koidomain.Validation(koidomain.FieldError{Field: "domain", Code: koidomain.FieldTaken})
		}

		if pgErr.ConstraintName == "idx_competitors_brand_github_repo" {
			return koidomain.Validation(koidomain.FieldError{Field: "github_repo", Code: koidomain.FieldTaken})
		}

		if strings.Contains(strings.ToLower(pgErr.ConstraintName), "email") ||
			strings.EqualFold(pgErr.ColumnName, "email") {
			return koidomain.ErrEmailTaken
		}
	}

	return err
}

func updated(result *gorm.DB, requireMatch bool) error {
	if err := mapDBError(result.Error); err != nil {
		return err
	}

	if requireMatch && result.RowsAffected == 0 {
		return koidomain.ErrNotFound
	}

	return nil
}
