package database

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/koihq/koi/pkg/domain"
)

type GormUserRepository struct {
	*Repository[domain.User]
}

func NewUserRepository(db *gorm.DB) *GormUserRepository {
	return &GormUserRepository{Repository: newRepository[domain.User](db, "password_hash")}
}

func (r *GormUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	entity := new(domain.User)
	err := r.db.WithContext(ctx).Where("email = ?", email).Take(entity).Error
	if err != nil {
		return nil, mapDBError(err)
	}

	return entity, nil
}

func (r *GormUserRepository) LockForUpdate(ctx context.Context, id uuid.UUID) error {
	var user domain.User
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").Where("id = ?", id).Take(&user).Error

	return mapDBError(err)
}

// signupLockKey is an arbitrary constant that names this lock among Postgres advisory locks.
const signupLockKey = 7246310

func (r *GormUserRepository) LockAndCount(ctx context.Context) (int64, error) {
	db := r.db.WithContext(ctx)
	if err := db.Exec("SELECT pg_advisory_xact_lock(?)", signupLockKey).Error; err != nil {
		return 0, err
	}

	var count int64
	err := db.Model(&domain.User{}).Count(&count).Error

	return count, err
}
