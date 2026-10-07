package koiapi

import (
	"net/http"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/koihq/koi/pkg/koiapi/api"
	"github.com/koihq/koi/pkg/koiapi/middlewares"
	"github.com/koihq/koi/pkg/koidomain"
)

type GenerationHandler struct {
	querier   koidomain.GenerationQuerier
	commander koidomain.GenerationCommander
	sess      *scs.SessionManager
}

func NewGenerationHandler(q koidomain.GenerationQuerier, c koidomain.GenerationCommander, sess *scs.SessionManager) *GenerationHandler {
	return &GenerationHandler{querier: q, commander: c, sess: sess}
}

func (h *GenerationHandler) Routes() func(chi.Router) {
	return func(r chi.Router) {
		r.Use(Authorizer(h.sess))
		r.With(middlewares.DecodeBody[generationRequest]()).Post("/", h.Create)
		r.Get("/", h.List)
		r.Get("/{generationID}", h.Get)
	}
}

type generationRequest struct {
	Mode koidomain.GenerationMode `json:"mode"`
}

type generationAccepted struct {
	ID        uuid.UUID                 `json:"id"`
	Mode      koidomain.GenerationMode  `json:"mode"`
	State     koidomain.GenerationState `json:"state"`
	StatusURL string                    `json:"status_url"`
}

type generationStatus struct {
	ID              uuid.UUID                 `json:"id"`
	Mode            koidomain.GenerationMode  `json:"mode"`
	State           koidomain.GenerationState `json:"state"`
	Attempt         int                       `json:"attempt"`
	PromptCount     int                       `json:"prompt_count"`
	CompetitorCount int                       `json:"competitor_count"`
	CreatedAt       time.Time                 `json:"created_at"`
	StartedAt       *time.Time                `json:"started_at,omitempty"`
	FinishedAt      *time.Time                `json:"finished_at,omitempty"`
	NextRetryAt     *time.Time                `json:"next_retry_at,omitempty"`
	ErrorCode       *string                   `json:"error_code,omitempty"`
}

func generationStatusURL(brandID, runID uuid.UUID) string {
	return "/api/v1/brands/" + brandID.String() + "/generations/" + runID.String()
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
	_, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}

	input := middlewares.Body[generationRequest](r)
	var key *string
	if values, present := r.Header[http.CanonicalHeaderKey("Idempotency-Key")]; present {
		if len(values) != 1 || !validIdempotencyKey(values[0]) {
			api.WriteDomainError(w, r, koidomain.Validation(koidomain.FieldError{Field: "idempotency_key", Code: koidomain.FieldInvalid}))

			return
		}

		key = &values[0]
	}

	run, terminalReplay, err := h.commander.Create(r.Context(), brandID, input.Mode, key)
	if err != nil {
		status, env := api.MapError(err)
		if domainErr, ok := err.(*koidomain.Error); ok && domainErr.RunID != uuid.Nil {
			env.Error.StatusURL = generationStatusURL(brandID, domainErr.RunID)
		}

		if status == http.StatusInternalServerError {
			api.WriteDomainError(w, r, err)

			return
		}

		api.JSON(w, r, status, env)

		return
	}

	status := http.StatusAccepted
	if terminalReplay {
		status = http.StatusOK
	}

	api.JSON(w, r, status, generationAccepted{
		ID:        run.ID,
		Mode:      run.Mode,
		State:     run.State,
		StatusURL: generationStatusURL(brandID, run.ID),
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
	_, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}

	runID, ok := pathID(w, r, "generationID")
	if !ok {
		return
	}

	run, err := h.querier.Get(r.Context(), brandID, runID)
	if err != nil {
		api.WriteDomainError(w, r, err)

		return
	}

	api.JSON(w, r, http.StatusOK, publicGenerationStatus(run))
}

func (h *GenerationHandler) List(w http.ResponseWriter, r *http.Request) {
	_, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}

	page, err := pageParams(r)
	if err != nil {
		api.WriteDomainError(w, r, err)

		return
	}

	runs, err := h.querier.List(r.Context(), brandID, page)
	if err != nil {
		api.WriteDomainError(w, r, err)

		return
	}

	items := make([]generationStatus, 0, len(runs.Items))

	for i := range runs.Items {
		items = append(items, publicGenerationStatus(&runs.Items[i]))
	}

	api.JSON(w, r, http.StatusOK, koidomain.Page[generationStatus]{
		Items:       items,
		TotalItems:  runs.TotalItems,
		TotalPages:  runs.TotalPages,
		CurrentPage: runs.CurrentPage,
		HasNext:     runs.HasNext,
		HasPrev:     runs.HasPrev,
	})
}
