package koiapi

import (
	"net/http"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/koihq/koi/pkg/koiapi/api"
	"github.com/koihq/koi/pkg/koidomain"
)

type VisibilityHandler struct {
	querier   koidomain.VisibilityQuerier
	commander koidomain.VisibilityCommander
	sess      *scs.SessionManager
}

func NewVisibilityHandler(q koidomain.VisibilityQuerier, c koidomain.VisibilityCommander, sess *scs.SessionManager) *VisibilityHandler {
	return &VisibilityHandler{querier: q, commander: c, sess: sess}
}

func (h *VisibilityHandler) Routes() func(chi.Router) {
	return func(r chi.Router) {
		r.Use(Authorizer(h.sess))
		r.Post("/", h.Create)
		r.Get("/", h.List)
		r.Get("/{runID}", h.Get)
		r.Get("/{runID}/results", h.ListResults)
	}
}

type visibilityStatus struct {
	ID               uuid.UUID                 `json:"id"`
	BrandID          uuid.UUID                 `json:"brand_id"`
	Engines          []koidomain.EngineID      `json:"engines"`
	State            koidomain.VisibilityState `json:"state"`
	TotalResults     int64                     `json:"total_results"`
	CompletedResults int64                     `json:"completed_results"`
	CreatedAt        time.Time                 `json:"created_at"`
	StartedAt        *time.Time                `json:"started_at"`
	FinishedAt       *time.Time                `json:"finished_at"`
	ErrorCode        *string                   `json:"error_code"`
	StatusURL        string                    `json:"status_url"`
}

func publicVisibilityStatus(run *koidomain.VisibilityRun) visibilityStatus {
	return visibilityStatus{
		ID:               run.ID,
		BrandID:          run.BrandID,
		Engines:          run.Engines,
		State:            run.State,
		TotalResults:     run.TotalResults,
		CompletedResults: run.CompletedResults,
		CreatedAt:        run.CreatedAt,
		StartedAt:        run.StartedAt,
		FinishedAt:       run.FinishedAt,
		ErrorCode:        run.ErrorCode,
		StatusURL:        "/api/v1/brands/" + run.BrandID.String() + "/runs/" + run.ID.String(),
	}
}

func (h *VisibilityHandler) Create(w http.ResponseWriter, r *http.Request) {
	_, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}

	run, err := h.commander.Create(r.Context(), brandID)
	if err != nil {
		api.WriteDomainError(w, r, err)

		return
	}

	api.JSON(w, r, http.StatusAccepted, publicVisibilityStatus(run))
}

func (h *VisibilityHandler) Get(w http.ResponseWriter, r *http.Request) {
	_, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}

	runID, ok := pathID(w, r, "runID")
	if !ok {
		return
	}

	run, err := h.querier.Get(r.Context(), brandID, runID)
	if err != nil {
		api.WriteDomainError(w, r, err)

		return
	}

	api.JSON(w, r, http.StatusOK, publicVisibilityStatus(run))
}

func (h *VisibilityHandler) List(w http.ResponseWriter, r *http.Request) {
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

	items := make([]visibilityStatus, 0, len(runs.Items))

	for i := range runs.Items {
		items = append(items, publicVisibilityStatus(&runs.Items[i]))
	}

	api.JSON(w, r, http.StatusOK, koidomain.Page[visibilityStatus]{
		Items:       items,
		TotalItems:  runs.TotalItems,
		TotalPages:  runs.TotalPages,
		CurrentPage: runs.CurrentPage,
		HasNext:     runs.HasNext,
		HasPrev:     runs.HasPrev,
	})
}

func (h *VisibilityHandler) ListResults(w http.ResponseWriter, r *http.Request) {
	_, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}

	runID, ok := pathID(w, r, "runID")
	if !ok {
		return
	}

	page, err := pageParams(r)
	if err != nil {
		api.WriteDomainError(w, r, err)

		return
	}

	results, err := h.querier.ListResults(r.Context(), brandID, runID, page)
	if err != nil {
		api.WriteDomainError(w, r, err)

		return
	}

	api.JSON(w, r, http.StatusOK, results)
}
