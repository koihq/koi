package koidomain

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type EngineID string

const (
	EngineChatGPT           EngineID = "chatgpt"
	EnginePerplexity        EngineID = "perplexity"
	EngineGemini            EngineID = "gemini"
	EngineGoogleAIOverviews EngineID = "google_ai_overviews"
)

func VisibilityEngines() []EngineID {
	return []EngineID{EngineChatGPT, EnginePerplexity, EngineGemini, EngineGoogleAIOverviews}
}

type CollectorID string

const (
	CollectorStub  CollectorID = "stub"
	CollectorCloro CollectorID = "cloro"
)

type Outcome string

const (
	OutcomeCited     Outcome = "cited"
	OutcomeMentioned Outcome = "mentioned"
	OutcomeMiss      Outcome = "miss"
	OutcomeError     Outcome = "error"
)

type VisibilityState string

const (
	VisibilityQueued    VisibilityState = "queued"
	VisibilityRunning   VisibilityState = "running"
	VisibilitySucceeded VisibilityState = "succeeded"
	VisibilityFailed    VisibilityState = "failed"
)

type VisibilityRun struct {
	BaseEntity
	BrandID          uuid.UUID       `json:"-" gorm:"type:uuid;not null"`
	BrandSnapshot    BrandSnapshot   `json:"-" gorm:"type:jsonb;serializer:json;not null"`
	Engines          []EngineID      `json:"-" gorm:"type:jsonb;serializer:json;not null"`
	State            VisibilityState `json:"-" gorm:"type:varchar(16);not null;check:chk_visibility_state,state IN ('queued','running','succeeded','failed')"`
	TotalResults     int64           `json:"-" gorm:"not null"`
	CompletedResults int64           `json:"-" gorm:"->;-:migration"`
	RiverJobID       *int64          `json:"-"`
	Attempt          int             `json:"-" gorm:"not null;default:0"`
	StartedAt        *time.Time      `json:"-"`
	FinishedAt       *time.Time      `json:"-"`
	ErrorCode        *string         `json:"-"`
	Brand            *Brand          `json:"-" gorm:"foreignKey:BrandID;constraint:OnDelete:CASCADE"`
}

func (VisibilityRun) TableName() string { return "visibility_runs" }

func (r VisibilityRun) Validate() error {
	if r.BrandID == uuid.Nil || len(r.Engines) == 0 || r.TotalResults <= 0 || r.State != VisibilityQueued {
		return Validation(FieldError{Field: "run", Code: FieldInvalid})
	}

	return nil
}

type VisibilityResult struct {
	BaseEntity
	RunID uuid.UUID `json:"run_id" gorm:"type:uuid;not null"`
	// PromptID is a snapshot identifier, deliberately independent of the live prompt row.
	PromptID       uuid.UUID      `json:"prompt_id" gorm:"type:uuid;not null"`
	PromptText     string         `json:"prompt_text" gorm:"not null"`
	PromptPosition int            `json:"prompt_position" gorm:"not null"`
	Engine         EngineID       `json:"engine" gorm:"type:varchar(32);not null"`
	Collector      CollectorID    `json:"collector" gorm:"type:varchar(16);not null"`
	FetchedAt      *time.Time     `json:"fetched_at"`
	Outcome        *Outcome       `json:"outcome" gorm:"type:varchar(16);check:chk_visibility_outcome,outcome IS NULL OR outcome IN ('cited','mentioned','miss','error')"`
	RawAnswer      string         `json:"raw_answer" gorm:"not null;default:''"`
	Sources        []string       `json:"sources" gorm:"type:jsonb;serializer:json;not null"`
	ErrorCode      *string        `json:"error_code"`
	Run            *VisibilityRun `json:"-" gorm:"foreignKey:RunID;constraint:OnDelete:CASCADE"`
}

func (VisibilityResult) TableName() string { return "visibility_results" }

func (r VisibilityResult) Validate() error {
	if r.RunID == uuid.Nil || r.PromptID == uuid.Nil || strings.TrimSpace(r.PromptText) == "" || r.PromptPosition < 0 {
		return Validation(FieldError{Field: "result", Code: FieldInvalid})
	}

	validEngine := false

	for _, engine := range VisibilityEngines() {
		validEngine = validEngine || r.Engine == engine
	}

	if !validEngine || r.Collector != CollectorStub && r.Collector != CollectorCloro || r.Sources == nil {
		return Validation(FieldError{Field: "result", Code: FieldInvalid})
	}

	if (r.FetchedAt == nil) != (r.Outcome == nil) {
		return Validation(FieldError{Field: "outcome", Code: FieldInvalid})
	}

	if r.Outcome != nil {
		switch *r.Outcome {
		case OutcomeCited, OutcomeMentioned, OutcomeMiss:
			if r.ErrorCode != nil {
				return Validation(FieldError{Field: "error_code", Code: FieldInvalid})
			}
		case OutcomeError:
			if r.ErrorCode == nil || *r.ErrorCode == "" {
				return Validation(FieldError{Field: "error_code", Code: FieldInvalid})
			}
		default:
			return Validation(FieldError{Field: "outcome", Code: FieldInvalid})
		}
	}

	return nil
}

type VisibilityInputs struct {
	Prompts                []Prompt
	UnconfirmedCompetitors int64
}

type VisibilityRepository interface {
	Get(context.Context, uuid.UUID, uuid.UUID) (*VisibilityRun, error)
	List(context.Context, uuid.UUID, PageParams) (*Page[VisibilityRun], error)
	Active(context.Context, uuid.UUID) (*VisibilityRun, error)
	Inputs(context.Context, uuid.UUID) (VisibilityInputs, error)
	CreateAndEnqueue(context.Context, *VisibilityRun, []VisibilityResult) error
	ListResults(context.Context, uuid.UUID, uuid.UUID, PageParams) (*Page[VisibilityResult], error)
}

type VisibilityCommander interface {
	Create(context.Context, uuid.UUID) (*VisibilityRun, error)
}

type VisibilityQuerier interface {
	Get(context.Context, uuid.UUID, uuid.UUID) (*VisibilityRun, error)
	List(context.Context, uuid.UUID, PageParams) (*Page[VisibilityRun], error)
	ListResults(context.Context, uuid.UUID, uuid.UUID, PageParams) (*Page[VisibilityResult], error)
}

type visibilityService struct{ store Store }

func NewVisibilityCommander(store Store) VisibilityCommander { return &visibilityService{store: store} }

func NewVisibilityQuerier(store Store) VisibilityQuerier { return &visibilityService{store: store} }

func (s *visibilityService) Create(ctx context.Context, brandID uuid.UUID) (*VisibilityRun, error) {
	var run *VisibilityRun

	err := s.store.Atomic(ctx, func(tx Store) error {
		brand, err := tx.BrandRepo().LockForUpdate(ctx, brandID)
		if err != nil {
			return err
		}

		repo := tx.VisibilityRepo()
		run, err = repo.Active(ctx, brandID)
		if err == nil {
			return nil
		}

		if !errors.Is(err, ErrNotFound) {
			return err
		}

		inputs, err := repo.Inputs(ctx, brandID)
		if err != nil {
			return err
		}

		var fields []FieldError
		if len(inputs.Prompts) == 0 {
			fields = append(fields, FieldError{Field: "prompts", Code: "empty"})
		}

		for _, prompt := range inputs.Prompts {
			if !prompt.Confirmed {
				fields = append(fields, FieldError{Field: "prompts", Code: "unconfirmed"})
				break
			}
		}

		if inputs.UnconfirmedCompetitors > 0 {
			fields = append(fields, FieldError{Field: "competitors", Code: "unconfirmed"})
		}

		if len(fields) > 0 {
			return &Error{Code: CodeRunNotReady, Message: "run not ready", Fields: fields}
		}

		run = &VisibilityRun{
			BrandID: brandID,
			BrandSnapshot: BrandSnapshot{
				Name:        brand.Name,
				Domain:      brand.Domain,
				GitHubRepo:  brand.GitHubRepo,
				Country:     brand.Country,
				Description: brand.Description,
			},
			Engines:      VisibilityEngines(),
			State:        VisibilityQueued,
			TotalResults: int64(len(inputs.Prompts) * len(VisibilityEngines())),
		}
		if err := run.Validate(); err != nil {
			return err
		}

		results := make([]VisibilityResult, 0, run.TotalResults)

		for position, prompt := range inputs.Prompts {
			for _, engine := range run.Engines {
				results = append(results, VisibilityResult{
					PromptID:       prompt.ID,
					PromptText:     prompt.Text,
					PromptPosition: position,
					Engine:         engine,
					Collector:      CollectorStub,
					Sources:        []string{},
				})
			}
		}

		return repo.CreateAndEnqueue(ctx, run, results)
	})
	if err != nil {
		return nil, err
	}

	return run, nil
}

func (s *visibilityService) Get(ctx context.Context, brandID, runID uuid.UUID) (*VisibilityRun, error) {
	if err := ownedBrand(ctx, s.store, brandID); err != nil {
		return nil, err
	}

	return s.store.VisibilityRepo().Get(ctx, brandID, runID)
}

func (s *visibilityService) List(ctx context.Context, brandID uuid.UUID, page PageParams) (*Page[VisibilityRun], error) {
	if err := ownedBrand(ctx, s.store, brandID); err != nil {
		return nil, err
	}

	return s.store.VisibilityRepo().List(ctx, brandID, page)
}

func (s *visibilityService) ListResults(ctx context.Context, brandID, runID uuid.UUID, page PageParams) (*Page[VisibilityResult], error) {
	if _, err := s.Get(ctx, brandID, runID); err != nil {
		return nil, err
	}

	return s.store.VisibilityRepo().ListResults(ctx, brandID, runID, page)
}
