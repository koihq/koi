package koidomain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type GenerationMode string

const (
	GenerationInitial     GenerationMode = "initial"
	GenerationPrompts     GenerationMode = "prompts"
	GenerationCompetitors GenerationMode = "competitors"
)

type GenerationState string

const (
	GenerationQueued    GenerationState = "queued"
	GenerationRunning   GenerationState = "running"
	GenerationSucceeded GenerationState = "succeeded"
	GenerationFailed    GenerationState = "failed"
)

const (
	GenerationModelSlug     = "deepseek/deepseek-v4.1-flash"
	GenerationPromptVersion = "draft_v1"
)

type BrandSnapshot struct {
	Name        string `json:"name"`
	Domain      string `json:"domain,omitempty"`
	GitHubRepo  string `json:"github_repo,omitempty"`
	Country     string `json:"country"`
	Description string `json:"description,omitempty"`
}

type DraftCompany struct {
	Name   string `json:"name"`
	Domain string `json:"domain"`
}

type DraftInput struct {
	Brand               BrandSnapshot  `json:"brand"`
	Mode                GenerationMode `json:"mode"`
	ExistingPrompts     []string       `json:"existing_prompts"`
	ExistingCompetitors []DraftCompany `json:"existing_competitors"`
}

type DraftOutput struct {
	Prompts     []string       `json:"prompts"`
	Competitors []DraftCompany `json:"competitors"`
}

type GenerationRun struct {
	BaseEntity
	BrandID         uuid.UUID       `json:"-" gorm:"type:uuid;not null"`
	Mode            GenerationMode  `json:"-" gorm:"type:varchar(16);not null"`
	State           GenerationState `json:"-" gorm:"type:varchar(16);not null"`
	RiverJobID      *int64          `json:"-"`
	IdempotencyKey  *string         `json:"-" gorm:"type:varchar(255)"`
	Attempt         int             `json:"-" gorm:"not null;default:0"`
	PromptCount     int             `json:"-" gorm:"not null;default:0"`
	CompetitorCount int             `json:"-" gorm:"not null;default:0"`
	StartedAt       *time.Time      `json:"-"`
	FinishedAt      *time.Time      `json:"-"`
	NextRetryAt     *time.Time      `json:"-"`
	ErrorCode       *string         `json:"-"`
	ModelSlug       string          `json:"-" gorm:"not null"`
	PromptVersion   string          `json:"-" gorm:"not null"`
	BrandSnapshot   BrandSnapshot   `json:"-" gorm:"type:jsonb;serializer:json;not null"`
	Brand           *Brand          `json:"-" gorm:"foreignKey:BrandID;constraint:OnDelete:CASCADE"`
}

func (GenerationRun) TableName() string { return "generation_runs" }

type GenerationRepository interface {
	Get(context.Context, uuid.UUID, uuid.UUID) (*GenerationRun, error)
	List(context.Context, uuid.UUID, PageParams) (*Page[GenerationRun], error)
	ByKey(context.Context, uuid.UUID, string) (*GenerationRun, error)
	Active(context.Context, uuid.UUID) (*GenerationRun, error)
	HasSucceededInitial(context.Context, uuid.UUID) (bool, error)
	CreateAndEnqueue(context.Context, *GenerationRun) error
}

type GenerationCommander interface {
	Create(context.Context, uuid.UUID, GenerationMode, *string) (*GenerationRun, bool, error)
}

type GenerationQuerier interface {
	Get(context.Context, uuid.UUID, uuid.UUID) (*GenerationRun, error)
	List(context.Context, uuid.UUID, PageParams) (*Page[GenerationRun], error)
}

type generationService struct {
	store   Store
	enabled bool
}

func NewGenerationCommander(store Store, enabled bool) GenerationCommander {
	return &generationService{store: store, enabled: enabled}
}

func NewGenerationQuerier(store Store) GenerationQuerier {
	return &generationService{store: store}
}

func validGenerationMode(mode GenerationMode) bool {
	return mode == GenerationInitial || mode == GenerationPrompts || mode == GenerationCompetitors
}

func (s *generationService) Create(ctx context.Context, brandID uuid.UUID, mode GenerationMode, key *string) (*GenerationRun, bool, error) {
	if !validGenerationMode(mode) {
		return nil, false, Validation(FieldError{Field: "mode", Code: FieldInvalid})
	}
	var run *GenerationRun
	var terminalReplay bool
	err := s.store.Atomic(ctx, func(tx Store) error {
		brand, err := tx.BrandRepo().LockForUpdate(ctx, brandID)
		if err != nil {
			return err
		}
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
				if run.Mode != mode {
					return ErrIdempotencyConflict
				}
				terminalReplay = run.State == GenerationSucceeded || run.State == GenerationFailed
				return nil
			}
		}
		active, err := repo.Active(ctx, brandID)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
		if err == nil {
			if active.Mode == mode || active.Mode == GenerationInitial && mode != GenerationInitial {
				run = active
				return nil
			}
			return &Error{Code: CodeGenerationConflict, Message: ErrGenerationConflict.Message, RunID: active.ID}
		}
		if mode == GenerationInitial {
			succeeded, err := repo.HasSucceededInitial(ctx, brandID)
			if err != nil {
				return err
			}
			if succeeded {
				return ErrInitialAlreadyCompleted
			}
		}
		run = &GenerationRun{
			BrandID: brandID, Mode: mode, State: GenerationQueued, IdempotencyKey: key,
			ModelSlug: GenerationModelSlug, PromptVersion: GenerationPromptVersion,
			BrandSnapshot: BrandSnapshot{Name: brand.Name, Domain: brand.Domain, GitHubRepo: brand.GitHubRepo,
				Country: brand.Country, Description: brand.Description},
		}
		return repo.CreateAndEnqueue(ctx, run)
	})
	if err != nil {
		return nil, false, err
	}
	return run, terminalReplay, nil
}

func (s *generationService) Get(ctx context.Context, brandID, runID uuid.UUID) (*GenerationRun, error) {
	if err := ownedBrand(ctx, s.store, brandID); err != nil {
		return nil, err
	}
	return s.store.GenerationRepo().Get(ctx, brandID, runID)
}

func (s *generationService) List(ctx context.Context, brandID uuid.UUID, page PageParams) (*Page[GenerationRun], error) {
	if err := ownedBrand(ctx, s.store, brandID); err != nil {
		return nil, err
	}
	return s.store.GenerationRepo().List(ctx, brandID, page)
}
