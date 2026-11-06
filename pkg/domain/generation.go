package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type GenerationState string

const (
	GenerationQueued    GenerationState = "queued"
	GenerationRunning   GenerationState = "running"
	GenerationSucceeded GenerationState = "succeeded"
	GenerationFailed    GenerationState = "failed"
)

const GenerationModelSlug = "deepseek/deepseek-v4.1-flash"

// MentionModelSlug was chosen by testing candidates against hand-labeled engine answers.
const MentionModelSlug = "deepseek/deepseek-v4.1-flash"

type BrandSnapshot struct {
	Name        string `json:"name"`
	Domain      string `json:"domain,omitempty"`
	GitHubRepo  string `json:"github_repo,omitempty"`
	Country     string `json:"country"`
	Description string `json:"description,omitempty"`
}

type CompetitorIdentity struct {
	Name       string `json:"name"`
	Domain     string `json:"domain"`
	GitHubRepo string `json:"github_repo"`
}

type GenerationInput struct {
	Brand               BrandSnapshot        `json:"brand"`
	PromptTarget        int                  `json:"prompt_target"`
	ExistingPrompts     []string             `json:"existing_prompts"`
	ExistingCompetitors []CompetitorIdentity `json:"existing_competitors"`
}

type GenerationResult struct {
	Prompts     []string             `json:"prompts"`
	Competitors []CompetitorIdentity `json:"competitors"`
}

// GenerationSummary is what the draft understood about the product, shown back to the customer
// while the draft continues.
type GenerationSummary struct {
	Product   string   `json:"product"`
	Category  string   `json:"category"`
	Audience  string   `json:"audience"`
	Functions []string `json:"functions"`
}

type GenerationRun struct {
	BaseEntity
	// Recent indexes use expressions for the inherited BaseEntity timestamps and ID.
	BrandID         uuid.UUID       `json:"-" gorm:"type:uuid;not null;uniqueIndex:idx_generation_runs_active,where:state = 'queued' OR state = 'running';index:idx_generation_runs_recent,priority:1;index:idx_generation_runs_recent,priority:2,expression:created_at,sort:desc;index:idx_generation_runs_recent,priority:3,expression:id,sort:desc;uniqueIndex:idx_generation_runs_key,priority:1,where:idempotency_key IS NOT NULL"`
	State           GenerationState `json:"-" gorm:"type:varchar(16);not null"`
	RiverJobID      *int64          `json:"-"`
	IdempotencyKey  *string         `json:"-" gorm:"type:varchar(255);uniqueIndex:idx_generation_runs_key,priority:2"`
	Attempt         int             `json:"-" gorm:"not null;default:0"`
	PromptCount     int             `json:"-" gorm:"not null;default:0"`
	CompetitorCount int             `json:"-" gorm:"not null;default:0"`
	StartedAt       *time.Time      `json:"-"`
	FinishedAt      *time.Time      `json:"-"`
	NextRetryAt     *time.Time      `json:"-"`
	ErrorCode       *string         `json:"-"`
	ModelSlug       string          `json:"-" gorm:"not null"`
	BrandSnapshot   BrandSnapshot   `json:"-" gorm:"type:jsonb;serializer:json;not null"`
	// StartTracking starts collection when the draft succeeds, so closing the page cannot stop it.
	StartTracking bool               `json:"-" gorm:"not null;default:false"`
	Summary       *GenerationSummary `json:"-" gorm:"type:jsonb;serializer:json"`
	Brand         *Brand             `json:"-" gorm:"foreignKey:BrandID;constraint:OnDelete:CASCADE"`
}

func (GenerationRun) TableName() string { return "generation_runs" }

func (r GenerationRun) GetBrandID() uuid.UUID { return r.BrandID }

type GenerationRepository interface {
	BaseEntityQuerier[GenerationRun]
	Lister[GenerationRun, BrandFilter]
	ByKey(context.Context, uuid.UUID, string) (*GenerationRun, error)
	Active(context.Context, uuid.UUID) (*GenerationRun, error)
	HasSucceeded(context.Context, uuid.UUID) (bool, error)
	CreateAndEnqueue(context.Context, *GenerationRun) error
}

type GenerationCommander interface {
	Create(ctx context.Context, brandID uuid.UUID, key *string, startTracking bool) (*GenerationRun, bool, error)
}

type GenerationQuerier interface {
	BaseEntityQuerier[GenerationRun]
	Lister[GenerationRun, BrandFilter]
}

type generationCommander struct {
	store   Store
	enabled bool
}

func NewGenerationCommander(store Store, enabled bool) GenerationCommander {
	return &generationCommander{store: store, enabled: enabled}
}

func NewGenerationQuerier(store Store) GenerationQuerier {
	return brandChildQuerier[GenerationRun, BrandFilter]{
		store: store,
		repo: func(s Store) brandChildRepository[GenerationRun, BrandFilter] {
			return s.GenerationRepo()
		},
	}
}

// Create starts the brand's one draft. A failed draft can run again until tracking starts.
func (s *generationCommander) Create(ctx context.Context, brandID uuid.UUID, key *string, startTracking bool) (*GenerationRun, bool, error) {
	var run *GenerationRun
	var terminalReplay bool

	err := withBrandLock(ctx, s.store, brandID, func(tx Store, brand *Brand) error {
		var err error

		if !s.enabled {
			return ErrFeatureUnavailable
		}

		repo := tx.GenerationRepo()
		if key != nil {
			run, err = repo.ByKey(ctx, brandID, *key)
			if err != nil && !errors.Is(err, ErrNotFound) {
				return err
			}

			if err == nil {
				terminalReplay = run.State == GenerationSucceeded || run.State == GenerationFailed

				return nil
			}
		}

		active, err := repo.Active(ctx, brandID)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}

		if err == nil {
			run = active

			return nil
		}

		if brand.TrackingConfirmedAt != nil {
			return ErrInitialAlreadyCompleted
		}

		succeeded, err := repo.HasSucceeded(ctx, brandID)
		if err != nil {
			return err
		}

		if succeeded {
			return ErrInitialAlreadyCompleted
		}

		run = &GenerationRun{
			BrandID:        brandID,
			State:          GenerationQueued,
			IdempotencyKey: key,
			ModelSlug:      GenerationModelSlug,
			BrandSnapshot:  brand.Snapshot(),
			StartTracking:  startTracking,
		}

		return repo.CreateAndEnqueue(ctx, run)
	})
	if err != nil {
		return nil, false, err
	}

	return run, terminalReplay, nil
}
