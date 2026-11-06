package database

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/domain"
)

var errNoDatabase = errors.New("database not connected")

type Store struct {
	liveCollection      bool
	cloudMode           bool
	welcomeEmail        bool
	passwordReset       bool
	firstLookDailyLimit int
	db                  *gorm.DB
	enqueueClient       *river.Client[*sql.Tx]
}

func NewStore(db *gorm.DB) *Store {
	return &Store{db: db, firstLookDailyLimit: domain.DefaultFirstLookDailyLimit}
}

func (s *Store) WithEnqueueClient(client *river.Client[*sql.Tx]) *Store {
	s.enqueueClient = client

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

func (s *Store) Atomic(ctx context.Context, fn func(domain.Store) error) error {
	if s == nil || s.db == nil {
		return errNoDatabase
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		bound := *s
		bound.db = tx

		return fn(&bound)
	})
}

type WelcomeJobArgs struct {
	UserID uuid.UUID `json:"user_id"`
}

func (WelcomeJobArgs) Kind() string { return "koi_welcome_email" }

func (s *Store) WithWelcomeEmail(enabled bool) *Store {
	s.welcomeEmail = enabled
	return s
}

func (s *Store) EnqueueWelcomeEmail(ctx context.Context, userID uuid.UUID) error {
	if !s.welcomeEmail || s.enqueueClient == nil {
		return nil
	}

	_, err := s.enqueueClient.Insert(ctx, WelcomeJobArgs{UserID: userID}, &river.InsertOpts{
		Queue:       "email",
		MaxAttempts: 3,
	})

	return err
}

// FirstCheckEmailJobArgs names the run; the worker reads the address and brand when sending.
type FirstCheckEmailJobArgs struct {
	RunID uuid.UUID `json:"run_id"`
}

func (FirstCheckEmailJobArgs) Kind() string { return "koi_first_check_email" }

// EnqueueFirstCheckEmail tells the owner their first check is ready. Each run sends at most once.
func (s *Store) EnqueueFirstCheckEmail(ctx context.Context, runID uuid.UUID) error {
	if !s.welcomeEmail || s.enqueueClient == nil {
		return nil
	}

	_, err := s.enqueueClient.Insert(ctx, FirstCheckEmailJobArgs{RunID: runID}, &river.InsertOpts{
		Queue:       "email",
		MaxAttempts: 3,
		UniqueOpts:  river.UniqueOpts{ByArgs: true},
	})

	return err
}

// FirstCheckEmail returns the owner's address, the brand name and how many real answers the run
// collected. Workers have no request identity, so this joins through the run itself.
func (s *Store) FirstCheckEmail(ctx context.Context, runID uuid.UUID) (string, string, int, error) {
	var row struct {
		Email   string
		Name    string
		Answers int
	}

	err := s.db.WithContext(ctx).Raw(`SELECT users.email, brands.name,
			(SELECT count(*) FROM visibility_results AS results
			 WHERE results.run_id = runs.id AND results.outcome IN ('cited', 'mentioned', 'miss')) AS answers
		FROM visibility_runs AS runs
		JOIN brands ON brands.id = runs.brand_id
		JOIN memberships ON memberships.organization_id = brands.organization_id AND memberships.role = ?
		JOIN users ON users.id = memberships.user_id
		WHERE runs.id = ?`, domain.RoleOwner, runID).Scan(&row).Error

	return row.Email, row.Name, row.Answers, err
}

// PasswordResetJobArgs holds an address, never a token: the worker creates that when sending.
type PasswordResetJobArgs struct {
	Email string `json:"email"`
}

func (PasswordResetJobArgs) Kind() string { return "koi_password_reset_email" }

func (s *Store) WithPasswordResetEmail(enabled bool) *Store {
	s.passwordReset = enabled
	return s
}

// EnqueuePasswordResetEmail sends at most one email per address per minute.
func (s *Store) EnqueuePasswordResetEmail(ctx context.Context, email string) error {
	if !s.passwordReset || s.enqueueClient == nil {
		return domain.ErrFeatureUnavailable
	}

	_, err := s.enqueueClient.Insert(ctx, PasswordResetJobArgs{Email: email}, &river.InsertOpts{
		Queue:       "email",
		MaxAttempts: 3,
		UniqueOpts: river.UniqueOpts{
			ByArgs:   true,
			ByPeriod: time.Minute,
		},
	})

	return err
}

func (s *Store) PasswordResetRepo() domain.PasswordResetRepository {
	return NewPasswordResetRepository(s.db)
}

func (s *Store) UserRepo() domain.UserRepository {
	return NewUserRepository(s.db)
}

func (s *Store) OrganizationRepo() domain.OrganizationRepository {
	return NewOrganizationRepository(s.db)
}

func (s *Store) MembershipRepo() domain.MembershipRepository {
	return NewMembershipRepository(s.db)
}

func (s *Store) CredentialRepo() domain.CredentialRepository {
	return NewCredentialRepository(s.db)
}

func (s *Store) BillingRepo() domain.BillingRepository {
	return NewBillingRepository(s.db, s.cloudMode)
}

func (s *Store) BrandRepo() domain.BrandRepository {
	return NewBrandRepository(s.db)
}

func (s *Store) PromptRepo() domain.PromptRepository {
	return NewPromptRepository(s.db)
}

func (s *Store) CompetitorRepo() domain.CompetitorRepository {
	return NewCompetitorRepository(s.db)
}

func (s *Store) GenerationRepo() domain.GenerationRepository {
	return NewGenerationRepository(s.db, s.enqueueClient, s.cloudMode)
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

func (s *Store) OnboardingRepo() domain.OnboardingRepository {
	return NewOnboardingRepository(s.db)
}

func (s *Store) WithFirstLookDailyLimit(limit int) *Store {
	s.firstLookDailyLimit = limit

	return s
}

func (s *Store) VisibilityRepo() domain.VisibilityRepository {
	repository := NewVisibilityRepository(s.db, s.enqueueClient, s.liveCollection, s.cloudMode)
	repository.firstLookDailyLimit = s.firstLookDailyLimit

	return repository
}
