package koiapi

import (
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apiresponse "github.com/koihq/koi/pkg/koiapi/response"
	"github.com/koihq/koi/pkg/koidomain"
)

type VisibilityHandler struct {
	querier   koidomain.VisibilityQuerier
	analytics *koidomain.VisibilityAnalyticsQuerier
}

func NewVisibilityHandler(q koidomain.VisibilityQuerier, analytics ...*koidomain.VisibilityAnalyticsQuerier) *VisibilityHandler {
	h := &VisibilityHandler{querier: q}
	if len(analytics) > 0 {
		h.analytics = analytics[0]
	}
	return h
}

func visibilityScope(r *http.Request) (int, []koidomain.EngineID, error) {
	days := 30
	if raw := r.URL.Query().Get("days"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value != 7 && value != 30 && value != 90 {
			return 0, nil, koidomain.Validation(koidomain.FieldError{Field: "days", Code: koidomain.FieldInvalid})
		}
		days = value
	}
	var engines []koidomain.EngineID
	if raw := r.URL.Query().Get("engines"); raw != "" {
		for _, part := range strings.Split(raw, ",") {
			engine := koidomain.EngineID(part)
			if !slices.Contains(koidomain.VisibilityEngines(), engine) {
				return 0, nil, koidomain.Validation(koidomain.FieldError{Field: "engines", Code: koidomain.FieldInvalid})
			}
			if !slices.Contains(engines, engine) {
				engines = append(engines, engine)
			}
		}
	}
	return days, engines, nil
}

func (h *VisibilityHandler) Summary(w http.ResponseWriter, r *http.Request) {
	_, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}
	days, engines, err := visibilityScope(r)
	if err != nil {
		apiresponse.WriteDomainError(w, r, err)
		return
	}
	report, err := h.analytics.Summary(r.Context(), brandID, days, engines)
	if err != nil {
		apiresponse.WriteDomainError(w, r, err)
		return
	}
	apiresponse.JSON(w, r, http.StatusOK, report)
}

func (h *VisibilityHandler) PromptVisibility(w http.ResponseWriter, r *http.Request) {
	_, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}
	promptID, ok := pathID(w, r, "promptID")
	if !ok {
		return
	}
	days, engines, err := visibilityScope(r)
	if err != nil {
		apiresponse.WriteDomainError(w, r, err)
		return
	}
	if _, err := h.querier.ListPromptRuns(r.Context(), brandID, promptID, koidomain.PageParams{Page: 1, PageSize: 1}); err != nil {
		apiresponse.WriteDomainError(w, r, err)
		return
	}
	byPrompt, _, err := h.analytics.Aggregate(r.Context(), brandID, []uuid.UUID{promptID}, days, engines)
	if err != nil {
		apiresponse.WriteDomainError(w, r, err)
		return
	}
	apiresponse.JSON(w, r, http.StatusOK, byPrompt[promptID])
}

func (h *VisibilityHandler) Routes() func(chi.Router) {
	return func(r chi.Router) {
		r.Get("/", h.List)
		r.Get("/{runID}", h.Get)
		r.Get("/{runID}/results", h.ListResults)
	}
}

type visibilityStatus struct {
	ID               uuid.UUID                 `json:"id"`
	BrandID          uuid.UUID                 `json:"brandId"`
	Engines          []koidomain.EngineID      `json:"engines"`
	State            koidomain.VisibilityState `json:"state"`
	TotalResults     int64                     `json:"totalResults"`
	CompletedResults int64                     `json:"completedResults"`
	CreatedAt        time.Time                 `json:"createdAt"`
	StartedAt        *time.Time                `json:"startedAt"`
	FinishedAt       *time.Time                `json:"finishedAt"`
	ErrorCode        *string                   `json:"errorCode"`
	StatusURL        string                    `json:"statusUrl"`
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
		StatusURL:        "/api/v1/runs/" + run.ID.String(),
	}
}

func (h *VisibilityHandler) Tracking(w http.ResponseWriter, r *http.Request) {
	_, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}
	status, err := h.querier.Tracking(r.Context(), brandID)
	if err != nil {
		apiresponse.WriteDomainError(w, r, err)
		return
	}
	apiresponse.JSON(w, r, http.StatusOK, TrackingStatusToResponse(status))
}

func (h *VisibilityHandler) Get(w http.ResponseWriter, r *http.Request) {
	runID, ok := pathID(w, r, "runID")
	if !ok {
		return
	}

	run, err := h.querier.Get(r.Context(), runID)
	if err != nil {
		apiresponse.WriteDomainError(w, r, err)

		return
	}

	apiresponse.JSON(w, r, http.StatusOK, publicVisibilityStatus(run))
}

func (h *VisibilityHandler) List(w http.ResponseWriter, r *http.Request) {
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

	items := make([]visibilityStatus, 0, len(runs.Items))

	for i := range runs.Items {
		items = append(items, publicVisibilityStatus(&runs.Items[i]))
	}

	apiresponse.JSON(w, r, http.StatusOK, koidomain.Page[visibilityStatus]{
		Items:       items,
		TotalItems:  runs.TotalItems,
		TotalPages:  runs.TotalPages,
		CurrentPage: runs.CurrentPage,
		HasNext:     runs.HasNext,
		HasPrev:     runs.HasPrev,
	})
}

func (h *VisibilityHandler) ListResults(w http.ResponseWriter, r *http.Request) {
	runID, ok := pathID(w, r, "runID")
	if !ok {
		return
	}

	page, err := pageParams(r)
	if err != nil {
		apiresponse.WriteDomainError(w, r, err)

		return
	}

	results, err := h.querier.ListResults(r.Context(), runID, page)
	if err != nil {
		apiresponse.WriteDomainError(w, r, err)

		return
	}

	apiresponse.JSON(w, r, http.StatusOK, mapPage(results, VisibilityResultToResponse))
}

func (h *VisibilityHandler) ListHistoricalPrompts(w http.ResponseWriter, r *http.Request) {
	_, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}

	params, err := promptListQuery(r, false)
	if err != nil {
		apiresponse.WriteDomainError(w, r, err)
		return
	}

	result, err := h.querier.ListHistoricalPrompts(r.Context(), brandID, params)
	if err != nil {
		apiresponse.WriteDomainError(w, r, err)
		return
	}
	if h.analytics != nil {
		ids := make([]uuid.UUID, 0, len(result.Items))
		for _, prompt := range result.Items {
			ids = append(ids, prompt.PromptID)
		}
		byPrompt, _, err := h.analytics.Aggregate(r.Context(), brandID, ids, params.Days, params.Engines)
		if err != nil {
			apiresponse.WriteDomainError(w, r, err)
			return
		}
		for i := range result.Items {
			result.Items[i].Visibility = byPrompt[result.Items[i].PromptID]
		}
	}

	apiresponse.JSON(w, r, http.StatusOK, mapPage(result, HistoricalPromptToResponse))
}

type promptHistoryRunRes struct {
	Run     visibilityStatus      `json:"run"`
	Results []VisibilityResultRes `json:"results"`
}

func (h *VisibilityHandler) ListPromptRuns(w http.ResponseWriter, r *http.Request) {
	_, brandID, ok := requestOwnerAndBrand(w, r)
	if !ok {
		return
	}

	promptID, ok := pathID(w, r, "promptID")
	if !ok {
		return
	}

	page, err := pageParams(r)
	if err != nil {
		apiresponse.WriteDomainError(w, r, err)
		return
	}

	history, err := h.querier.ListPromptRuns(r.Context(), brandID, promptID, page)
	if err != nil {
		apiresponse.WriteDomainError(w, r, err)
		return
	}

	items := make([]promptHistoryRunRes, 0, len(history.Items))
	for i := range history.Items {
		items = append(items, promptHistoryRunRes{
			Run:     publicVisibilityStatus(&history.Items[i].Run),
			Results: mapItems(history.Items[i].Results, VisibilityResultToResponse),
		})
	}

	apiresponse.JSON(w, r, http.StatusOK, koidomain.Page[promptHistoryRunRes]{
		Items:       items,
		TotalItems:  history.TotalItems,
		TotalPages:  history.TotalPages,
		CurrentPage: history.CurrentPage,
		HasNext:     history.HasNext,
		HasPrev:     history.HasPrev,
	})
}
