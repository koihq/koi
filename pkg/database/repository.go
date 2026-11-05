package database

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/auth"
	"github.com/koihq/koi/pkg/domain"
)

const pgUniqueViolation = "23505"

// tenantScope limits a repository to the organization in the request identity.
// A scoped repository without authorizeCreate rejects every Create.
type tenantScope[T any] struct {
	read            func(auth.Identity, *gorm.DB) *gorm.DB
	authorizeCreate func(*gorm.DB, auth.Identity, *T) error
}

// Repository is the shared persistence for one entity. Reads, updates and deletes pass
// through the tenant scope, so a concrete repository declares only what differs.
type Repository[T domain.Entity] struct {
	db      *gorm.DB
	scope   *tenantScope[T]
	read    func(*gorm.DB) *gorm.DB
	columns []string
}

// newRepository is for entities without a tenant: users, organizations and memberships.
func newRepository[T domain.Entity](db *gorm.DB, columns ...string) *Repository[T] {
	return &Repository[T]{
		db:      db,
		columns: columns,
	}
}

func newScopedRepository[T domain.Entity](db *gorm.DB, scope tenantScope[T], columns ...string) *Repository[T] {
	return &Repository[T]{
		db:      db,
		scope:   &scope,
		columns: columns,
	}
}

// newBrandChildRepository scopes an entity through the organization of its brand.
func newBrandChildRepository[T domain.BrandChild](db *gorm.DB, columns ...string) *Repository[T] {
	var entity T
	table := entity.TableName()

	return newScopedRepository(db, tenantScope[T]{
		read: func(identity auth.Identity, db *gorm.DB) *gorm.DB {
			return db.Where("EXISTS (SELECT 1 FROM brands WHERE brands.id = "+table+".brand_id AND brands.organization_id = ?)", identity.OrganizationID)
		},
		authorizeCreate: func(db *gorm.DB, identity auth.Identity, entity *T) error {
			var brand domain.Brand
			err := db.Select("id").
				Where("id = ? AND organization_id = ?", (*entity).GetBrandID(), identity.OrganizationID).
				Take(&brand).Error

			return mapDBError(err)
		},
	}, columns...)
}

func requireTenant(ctx context.Context) (auth.Identity, error) {
	identity, ok := auth.FromContext(ctx)
	if !ok || identity.OrganizationID == uuid.Nil {
		return auth.Identity{}, domain.ErrUnauthorized
	}

	return identity, nil
}

func (r *Repository[T]) query(ctx context.Context) (*gorm.DB, error) {
	db := r.db.WithContext(ctx)
	if r.scope == nil {
		return db, nil
	}

	identity, err := requireTenant(ctx)
	if err != nil {
		return nil, err
	}

	return r.scope.read(identity, db), nil
}

// reader is query plus the read shape declared by the repository, if any.
func (r *Repository[T]) reader(ctx context.Context) (*gorm.DB, error) {
	db, err := r.query(ctx)
	if err != nil {
		return nil, err
	}

	if r.read != nil {
		db = r.read(db)
	}

	return db, nil
}

func (r *Repository[T]) whereID(db *gorm.DB, id uuid.UUID) *gorm.DB {
	var entity T

	return db.Where(entity.TableName()+".id = ?", id)
}

func (r *Repository[T]) Create(ctx context.Context, entity *T) error {
	db := r.db.WithContext(ctx)

	if r.scope != nil {
		identity, err := requireTenant(ctx)
		if err != nil {
			return err
		}

		if r.scope.authorizeCreate == nil {
			return domain.ErrUnauthorized
		}

		if err := r.scope.authorizeCreate(db, identity, entity); err != nil {
			return err
		}
	}

	return mapDBError(db.Create(entity).Error)
}

func (r *Repository[T]) Get(ctx context.Context, id uuid.UUID) (*T, error) {
	return r.getBy(ctx, func(db *gorm.DB) *gorm.DB { return r.whereID(db, id) })
}

func (r *Repository[T]) getBy(ctx context.Context, filter func(*gorm.DB) *gorm.DB) (*T, error) {
	db, err := r.reader(ctx)
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

// Update writes only the columns declared by the repository.
func (r *Repository[T]) Update(ctx context.Context, entity *T) error {
	return r.updateColumns(ctx, (*entity).GetID(), entity, r.columns)
}

func (r *Repository[T]) updateColumns(ctx context.Context, id uuid.UUID, values any, columns []string) error {
	db, err := r.query(ctx)
	if err != nil {
		return err
	}

	return requireRow(r.whereID(db.Model(new(T)), id).Select(columns).Updates(values))
}

func (r *Repository[T]) Delete(ctx context.Context, id uuid.UUID) error {
	db, err := r.query(ctx)
	if err != nil {
		return err
	}

	return requireRow(r.whereID(db, id).Delete(new(T)))
}

// ListRepository adds a typed filter and a stable order to a Repository.
type ListRepository[T domain.Entity, F any] struct {
	*Repository[T]
	filter func(*gorm.DB, F) *gorm.DB
	order  func(F) string
}

func newListRepository[T domain.Entity, F any](base *Repository[T], filter func(*gorm.DB, F) *gorm.DB, order func(F) string) *ListRepository[T, F] {
	return &ListRepository[T, F]{
		Repository: base,
		filter:     filter,
		order:      order,
	}
}

func (r *ListRepository[T, F]) List(ctx context.Context, filter F, page domain.PageParams) (*domain.Page[T], error) {
	db, err := r.reader(ctx)
	if err != nil {
		return nil, err
	}

	return paginate[T](r.filter(db, filter), page, r.order(filter))
}

func fixedOrder[F any](order string) func(F) string {
	return func(F) string { return order }
}

func filterByBrand(db *gorm.DB, filter domain.BrandFilter) *gorm.DB {
	return db.Where("brand_id = ?", filter.BrandID)
}

func paginate[T any](db *gorm.DB, params domain.PageParams, order string) (*domain.Page[T], error) {
	query := db.Model(new(T))
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, mapDBError(err)
	}

	page := &domain.Page[T]{
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
		return domain.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		switch pgErr.ConstraintName {
		case "fk_brands_organization", "fk_memberships_user", "fk_memberships_organization", "fk_billing_subscriptions_organization", "fk_billing_checkout_sessions_organization", "fk_billing_change_requests_organization":
			if strings.Contains(pgErr.Detail, "still referenced") {
				return domain.ErrResourceConflict
			}
			return domain.ErrNotFound
		}
	}
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		if pgErr.ConstraintName == "idx_prompts_brand_text" {
			return domain.Validation(domain.FieldError{Field: "text", Code: domain.FieldTaken})
		}

		if pgErr.ConstraintName == "idx_competitors_brand_domain" {
			return domain.Validation(domain.FieldError{Field: "domain", Code: domain.FieldTaken})
		}

		if pgErr.ConstraintName == "idx_competitors_brand_github_repo" {
			return domain.Validation(domain.FieldError{Field: "github_repo", Code: domain.FieldTaken})
		}

		if strings.Contains(strings.ToLower(pgErr.ConstraintName), "email") ||
			strings.EqualFold(pgErr.ColumnName, "email") {
			return domain.ErrEmailTaken
		}
	}

	return err
}

func requireRow(result *gorm.DB) error {
	if err := mapDBError(result.Error); err != nil {
		return err
	}

	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}
