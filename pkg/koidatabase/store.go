package koidatabase

import (
	"context"
	"database/sql"
	"errors"

	"github.com/riverqueue/river"
	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/koidomain"
)

var errNoDatabase = errors.New("database not connected")

type Store struct {
	liveCollection bool
	cloudMode      bool
	db             *gorm.DB
	enqueueClient  *river.Client[*sql.Tx]
}

func NewStore(db *gorm.DB, enqueueClient ...*river.Client[*sql.Tx]) *Store {
	s := &Store{db: db}
	if len(enqueueClient) > 0 {
		s.enqueueClient = enqueueClient[0]
	}

	return s
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
		return fn(&Store{db: tx, enqueueClient: s.enqueueClient, liveCollection: s.liveCollection, cloudMode: s.cloudMode})
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

func (s *Store) PromptRepo() koidomain.PromptRepository {
	return NewGormPromptRepository(s.db, s.cloudMode)
}

func (s *Store) CompetitorRepo() koidomain.CompetitorRepository {
	return NewGormCompetitorRepository(s.db)
}

func (s *Store) GenerationRepo() koidomain.GenerationRepository {
	return NewGormGenerationRepository(s.db, s.enqueueClient, s.cloudMode)
}

// WithLiveCollection enables paid collection only when the operator configured Cloro.
func (s *Store) WithLiveCollection(enabled bool) *Store {
	s.liveCollection = enabled
	return s
}

func (s *Store) WithCloudMode(enabled bool) *Store {
	s.cloudMode = enabled

	return s
}

func (s *Store) VisibilityRepo() koidomain.VisibilityRepository {
	return NewGormVisibilityRepository(s.db, s.enqueueClient, s.liveCollection, s.cloudMode)
}
