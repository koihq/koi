package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Entity interface {
	GetID() uuid.UUID
	TableName() string
}

// BrandChild is an entity whose tenant is the organization of its brand.
type BrandChild interface {
	Entity
	GetBrandID() uuid.UUID
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

type Lister[T Entity, F any] interface {
	List(ctx context.Context, filter F, page PageParams) (*Page[T], error)
}

type BaseEntityRepository[T Entity] interface {
	BaseEntityQuerier[T]
	Create(ctx context.Context, entity *T) error
}

type MutableRepository[T Entity] interface {
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// BrandFilter selects the rows of one brand. Resource filters embed it.
type BrandFilter struct {
	BrandID uuid.UUID
}

func (f BrandFilter) Brand() uuid.UUID { return f.BrandID }

// BrandScopedFilter is any list filter that belongs to one brand.
type BrandScopedFilter interface {
	Brand() uuid.UUID
}

// brandChildRepository is the read side a brandChildQuerier needs.
type brandChildRepository[T Entity, F BrandScopedFilter] interface {
	BaseEntityQuerier[T]
	Lister[T, F]
}

// brandChildQuerier serves the reads shared by everything owned by a brand. List checks the
// brand first, so a brand of another organization is 404 rather than an empty page.
type brandChildQuerier[T Entity, F BrandScopedFilter] struct {
	store Store
	repo  func(Store) brandChildRepository[T, F]
}

func (q brandChildQuerier[T, F]) Get(ctx context.Context, id uuid.UUID) (*T, error) {
	return q.repo(q.store).Get(ctx, id)
}

func (q brandChildQuerier[T, F]) List(ctx context.Context, filter F, page PageParams) (*Page[T], error) {
	if err := ownedBrand(ctx, q.store, filter.Brand()); err != nil {
		return nil, err
	}

	return q.repo(q.store).List(ctx, filter, page)
}

func ownedBrand(ctx context.Context, store Store, brandID uuid.UUID) error {
	_, err := store.BrandRepo().Get(ctx, brandID)

	return err
}

// withBrandLock runs fn in one transaction holding the brand row. Every change to what a brand
// tracks goes through this lock, so reviews and generation workers see a consistent set.
func withBrandLock(ctx context.Context, store Store, brandID uuid.UUID, fn func(tx Store, brand *Brand) error) error {
	return store.Atomic(ctx, func(tx Store) error {
		brand, err := tx.BrandRepo().LockForUpdate(ctx, brandID)
		if err != nil {
			return err
		}

		return fn(tx, brand)
	})
}

// withLockedChild resolves a child's brand, takes its lock and reads the child again under it,
// so the command starts from the row no other writer can change until it commits.
func withLockedChild[T BrandChild](ctx context.Context, store Store, id uuid.UUID, repo func(Store) BaseEntityQuerier[T], fn func(tx Store, brand *Brand, child *T) error) error {
	current, err := repo(store).Get(ctx, id)
	if err != nil {
		return err
	}

	return withBrandLock(ctx, store, (*current).GetBrandID(), func(tx Store, brand *Brand) error {
		child, err := repo(tx).Get(ctx, id)
		if err != nil {
			return err
		}

		return fn(tx, brand, child)
	})
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
