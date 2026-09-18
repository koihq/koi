package koidatabase

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/koidomain"
)

var errNoDatabase = errors.New("database not connected")

type Store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Ping(ctx context.Context) error {
	if s == nil || s.db == nil {
		return errNoDatabase
	}
	sqlDB, err := s.db.WithContext(ctx).DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

func (s *Store) Atomic(ctx context.Context, fn func(koidomain.Store) error) error {
	if s == nil || s.db == nil {
		return errNoDatabase
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&Store{db: tx})
	})
}

func (s *Store) UserRepo() koidomain.UserRepository {
	return NewGormUserRepository(s.db)
}

func (s *Store) OrganizationRepo() koidomain.OrganizationRepository {
	return NewGormOrganizationRepository(s.db)
}

func (s *Store) MembershipRepo() koidomain.MembershipRepository {
	return NewGormMembershipRepository(s.db)
}

func (s *Store) BrandRepo() koidomain.BrandRepository {
	return NewGormBrandRepository(s.db)
}
