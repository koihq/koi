package koiapi

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/koihq/koi/pkg/koiapi/middlewares"
	apiresponse "github.com/koihq/koi/pkg/koiapi/response"
	"github.com/koihq/koi/pkg/koidomain"
)

type GenerationHandler struct {
	querier   koidomain.GenerationQuerier
	commander koidomain.GenerationCommander
}

func NewGenerationHandler(q koidomain.GenerationQuerier, c koidomain.GenerationCommander) *GenerationHandler {
	return &GenerationHandler{querier: q, commander: c}
}

func (h *GenerationHandler) Routes() func(chi.Router) {
	return func(r chi.Router) {
		r.With(middlewares.DecodeBody[generationRequest]()).Post("/", h.Create)
		r.Get("/", h.List)
		r.Get("/{generationID}", h.Get)
	}
}

type generationRequest struct {
	BrandRequestScope
	Mode koidomain.GenerationMode `json:"mode"`
}

type generationAccepted struct {
	ID        uuid.UUID                 `json:"id"`
	Mode      koidomain.GenerationMode  `json:"mode"`
	State     koidomain.GenerationState `json:"state"`
	StatusURL string                    `json:"statusUrl"`
}

type generationStatus struct {
	ID              uuid.UUID                 `json:"id"`
	Mode            koidomain.GenerationMode  `json:"mode"`
	State           koidomain.GenerationState `json:"state"`
	Attempt         int                       `json:"attempt"`
	PromptCount     int                       `json:"promptCount"`
	CompetitorCount int                       `json:"competitorCount"`
	CreatedAt       time.Time                 `json:"createdAt"`
	StartedAt       *time.Time                `json:"startedAt,omitempty"`
	FinishedAt      *time.Time                `json:"finishedAt,omitempty"`
	NextRetryAt     *time.Time                `json:"nextRetryAt,omitempty"`
	ErrorCode       *string                   `json:"errorCode,omitempty"`
}

func generationStatusURL(runID uuid.UUID) string {
	return "/api/v1/generations/" + runID.String()
}

func publicGenerationStatus(run *koidomain.GenerationRun) generationStatus {
	return generationStatus{
		ID:              run.ID,
		Mode:            run.Mode,
		State:           run.State,
		Attempt:         run.Attempt,
		PromptCount:     run.PromptCount,
		CompetitorCount: run.CompetitorCount,
		CreatedAt:       run.CreatedAt,
		StartedAt:       run.StartedAt,
		FinishedAt:      run.FinishedAt,
		NextRetryAt:     run.NextRetryAt,
		ErrorCode:       run.ErrorCode,
	}
}

func (h *GenerationHandler) Create(w http.ResponseWriter, r *http.Request) {
	brandID := middlewares.Body[generationRequest](r).BrandID
	if brandID == uuid.Nil {
		apiresponse.WriteDomainError(w, r, koidomain.Validation(koidomain.FieldError{Field: "brandId", Code: koidomain.FieldInvalid}))
		return
	}

	input := middlewares.Body[generationRequest](r)
	var key *string
	if values, present := r.Header[http.CanonicalHeaderKey("Idempotency-Key")]; present {
		if len(values) != 1 || !validIdempotencyKey(values[0]) {
			apiresponse.WriteDomainError(w, r, koidomain.Validation(koidomain.FieldError{Field: "idempotency_key", Code: koidomain.FieldInvalid}))

			return
		}

		key = &values[0]
	}

	run, terminalReplay, err := h.commander.Create(r.Context(), brandID, input.Mode, key)
	if err != nil {
		status, env := apiresponse.MapError(err)
		if domainErr, ok := err.(*koidomain.Error); ok && domainErr.RunID != uuid.Nil {
			env.Error.StatusURL = generationStatusURL(domainErr.RunID)
		}

		if status == http.StatusInternalServerError {
			apiresponse.WriteDomainError(w, r, err)

			return
		}

		apiresponse.JSON(w, r, status, env)

		return
	}

	status := http.StatusAccepted
	if terminalReplay {
		status = http.StatusOK
	}

	w.Header().Set("Location", generationStatusURL(run.ID))
	apiresponse.JSON(w, r, status, generationAccepted{
		ID:        run.ID,
		Mode:      run.Mode,
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

func (h *GenerationHandler) Get(w http.ResponseWriter, r *http.Request) {
	runID, ok := pathID(w, r, "generationID")
	if !ok {
		return
	}

	run, err := h.querier.Get(r.Context(), runID)
	if err != nil {
		apiresponse.WriteDomainError(w, r, err)

		return
	}

	apiresponse.JSON(w, r, http.StatusOK, publicGenerationStatus(run))
}

func (h *GenerationHandler) List(w http.ResponseWriter, r *http.Request) {
	_, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}

	page, err := pageParams(r)
	if err != nil {
		apiresponse.WriteDomainError(w, r, err)

		return
	}

	runs, err := h.querier.List(r.Context(), brandID, page)
	if err != nil {
		apiresponse.WriteDomainError(w, r, err)

		return
	}

	items := make([]generationStatus, 0, len(runs.Items))

	for i := range runs.Items {
		items = append(items, publicGenerationStatus(&runs.Items[i]))
	}

	apiresponse.JSON(w, r, http.StatusOK, koidomain.Page[generationStatus]{
		Items:       items,
		TotalItems:  runs.TotalItems,
		TotalPages:  runs.TotalPages,
		CurrentPage: runs.CurrentPage,
		HasNext:     runs.HasNext,
		HasPrev:     runs.HasPrev,
	})
}
