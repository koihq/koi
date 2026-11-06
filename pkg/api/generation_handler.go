package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/koihq/koi/pkg/api/middlewares"
	"github.com/koihq/koi/pkg/api/response"
	"github.com/koihq/koi/pkg/domain"
)

type GenerationHandler struct {
	querier   domain.GenerationQuerier
	commander domain.GenerationCommander
}

func NewGenerationHandler(q domain.GenerationQuerier, c domain.GenerationCommander) *GenerationHandler {
	return &GenerationHandler{querier: q, commander: c}
}

func (h *GenerationHandler) Routes() func(chi.Router) {
	return func(r chi.Router) {
		r.With(middlewares.DecodeBody[CreateGenerationReq]()).Post("/", h.Create)
		r.Get("/", List(brandFilter, h.querier.List, GenerationToRes))
		r.Get("/{id}", Get(h.querier.Get, GenerationToRes))
	}
}

type CreateGenerationReq struct {
	BrandRequestScope
	StartTracking bool `json:"startTracking"`
}

type GenerationSummaryRes struct {
	Product   string   `json:"product"`
	Category  string   `json:"category"`
	Audience  string   `json:"audience"`
	Functions []string `json:"functions"`
}

type GenerationAcceptedRes struct {
	ID        uuid.UUID              `json:"id"`
	State     domain.GenerationState `json:"state"`
	StatusURL string                 `json:"statusUrl"`
}

type GenerationRes struct {
	ID              uuid.UUID              `json:"id"`
	State           domain.GenerationState `json:"state"`
	Attempt         int                    `json:"attempt"`
	PromptCount     int                    `json:"promptCount"`
	CompetitorCount int                    `json:"competitorCount"`
	CreatedAt       time.Time              `json:"createdAt"`
	StartedAt       *time.Time             `json:"startedAt,omitempty"`
	FinishedAt      *time.Time             `json:"finishedAt,omitempty"`
	NextRetryAt     *time.Time             `json:"nextRetryAt,omitempty"`
	ErrorCode       *string                `json:"errorCode,omitempty"`
	StartTracking   bool                   `json:"startTracking"`
	Summary         *GenerationSummaryRes  `json:"summary,omitempty"`
}

func generationStatusURL(runID uuid.UUID) string {
	return "/api/v1/generations/" + runID.String()
}

func GenerationToRes(run *domain.GenerationRun) GenerationRes {
	var summary *GenerationSummaryRes
	if run.Summary != nil {
		summary = &GenerationSummaryRes{
			Product:   run.Summary.Product,
			Category:  run.Summary.Category,
			Audience:  run.Summary.Audience,
			Functions: run.Summary.Functions,
		}
	}

	return GenerationRes{
		ID:              run.ID,
		State:           run.State,
		Attempt:         run.Attempt,
		PromptCount:     run.PromptCount,
		CompetitorCount: run.CompetitorCount,
		CreatedAt:       run.CreatedAt,
		StartedAt:       run.StartedAt,
		FinishedAt:      run.FinishedAt,
		NextRetryAt:     run.NextRetryAt,
		ErrorCode:       run.ErrorCode,
		StartTracking:   run.StartTracking,
		Summary:         summary,
	}
}

func (h *GenerationHandler) Create(w http.ResponseWriter, r *http.Request) {
	input := middlewares.Body[CreateGenerationReq](r)
	if err := requireBrand(input.BrandID); err != nil {
		response.WriteDomainError(w, r, err)
		return
	}

	var key *string
	if values, present := r.Header[http.CanonicalHeaderKey("Idempotency-Key")]; present {
		if len(values) != 1 || !validIdempotencyKey(values[0]) {
			response.WriteDomainError(w, r, domain.Validation(domain.FieldError{Field: "idempotency_key", Code: domain.FieldInvalid}))

			return
		}

		key = &values[0]
	}

	run, terminalReplay, err := h.commander.Create(r.Context(), input.BrandID, key, input.StartTracking)
	if err != nil {
		response.WriteDomainError(w, r, err)

		return
	}

	status := http.StatusAccepted
	if terminalReplay {
		status = http.StatusOK
	}

	w.Header().Set("Location", generationStatusURL(run.ID))
	response.JSON(w, r, status, GenerationAcceptedRes{
		ID:        run.ID,
		State:     run.State,
		StatusURL: generationStatusURL(run.ID),
	})
}

func validIdempotencyKey(key string) bool {
	if len(key) == 0 || len(key) > 255 {
		return false
	}

	for i := range len(key) {
		if key[i] < '!' || key[i] > '~' {
			return false
		}
	}

	return true
}
