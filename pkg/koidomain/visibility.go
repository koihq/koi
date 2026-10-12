package koidomain

import (
	"context"
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
	ScheduledFor *time.Time `json:"-" gorm:"uniqueIndex:idx_visibility_runs_slot,priority:2"`
	BaseEntity
	// Recent indexes use expressions for the inherited BaseEntity timestamps and ID.
	BrandID          uuid.UUID       `json:"-" gorm:"type:uuid;not null;uniqueIndex:idx_visibility_runs_active,where:state = 'queued' OR state = 'running';index:idx_visibility_runs_recent,priority:1;index:idx_visibility_runs_recent,priority:2,expression:created_at,sort:desc;index:idx_visibility_runs_recent,priority:3,expression:id,sort:desc;uniqueIndex:idx_visibility_runs_slot,priority:1,where:scheduled_for IS NOT NULL"`
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
	CheckedAt   *time.Time `json:"-"`
	CloroTaskID *string    `json:"-"`
	SubmittedAt *time.Time `json:"-"`
	BaseEntity
	RunID uuid.UUID `json:"run_id" gorm:"type:uuid;not null;uniqueIndex:idx_visibility_results_cell,priority:1;index:idx_visibility_results_order,priority:1"`
	// PromptID is a snapshot identifier, deliberately independent of the live prompt row.
	PromptID       uuid.UUID      `json:"prompt_id" gorm:"type:uuid;not null;uniqueIndex:idx_visibility_results_cell,priority:2"`
	PromptText     string         `json:"prompt_text" gorm:"not null"`
	PromptPosition int            `json:"prompt_position" gorm:"not null;index:idx_visibility_results_order,priority:2"`
	Engine         EngineID       `json:"engine" gorm:"type:varchar(32);not null;uniqueIndex:idx_visibility_results_cell,priority:3;index:idx_visibility_results_order,priority:3"`
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
	Prompts []Prompt
}

type VisibilityRepository interface {
	FinishReview(context.Context, uuid.UUID, string) error
	Tracking(context.Context, uuid.UUID) (*TrackingStatus, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (*VisibilityRun, error)
	List(context.Context, uuid.UUID, PageParams) (*Page[VisibilityRun], error)
	Active(context.Context, uuid.UUID) (*VisibilityRun, error)
	Inputs(context.Context, uuid.UUID) (VisibilityInputs, error)
	CreateAndEnqueue(context.Context, *VisibilityRun, []VisibilityResult) error
	ListResults(context.Context, uuid.UUID, uuid.UUID, PageParams) (*Page[VisibilityResult], error)
}

type TrackingStatus struct {
	PromptsReviewed     bool       `json:"prompts_reviewed"`
	CompetitorsReviewed bool       `json:"competitors_reviewed"`
	StartedAt           *time.Time `json:"started_at"`
	NextRunAt           *time.Time `json:"next_run_at"`
	BlockedReason       *string    `json:"blocked_reason"`
}

type VisibilityQuerier interface {
	Get(context.Context, uuid.UUID, uuid.UUID) (*VisibilityRun, error)
	List(context.Context, uuid.UUID, PageParams) (*Page[VisibilityRun], error)
	ListResults(context.Context, uuid.UUID, uuid.UUID, PageParams) (*Page[VisibilityResult], error)
	Tracking(context.Context, uuid.UUID) (*TrackingStatus, error)
}

type visibilityService struct{ store Store }

func NewVisibilityQuerier(store Store) VisibilityQuerier { return &visibilityService{store: store} }

func (s *visibilityService) Tracking(ctx context.Context, brandID uuid.UUID) (*TrackingStatus, error) {
	return s.store.VisibilityRepo().Tracking(ctx, brandID)
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
