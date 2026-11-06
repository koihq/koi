package api

import (
	"context"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/koihq/koi/pkg/domain"
)

type VisibilityRunRes struct {
	ID               uuid.UUID              `json:"id"`
	BrandID          uuid.UUID              `json:"brandId"`
	Engines          []domain.EngineID      `json:"engines"`
	State            domain.VisibilityState `json:"state"`
	TotalResults     int64                  `json:"totalResults"`
	CompletedResults int64                  `json:"completedResults"`
	CreatedAt        time.Time              `json:"createdAt"`
	StartedAt        *time.Time             `json:"startedAt"`
	FinishedAt       *time.Time             `json:"finishedAt"`
	ErrorCode        *string                `json:"errorCode"`
	StatusURL        string                 `json:"statusUrl"`
}

func VisibilityRunToRes(run *domain.VisibilityRun) VisibilityRunRes {
	return VisibilityRunRes{
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

type VisibilityResultRes struct {
	ID             uuid.UUID                 `json:"id"`
	RunID          uuid.UUID                 `json:"runId"`
	PromptID       uuid.UUID                 `json:"promptId"`
	PromptText     string                    `json:"promptText"`
	PromptPosition int                       `json:"promptPosition"`
	Engine         domain.EngineID           `json:"engine"`
	Collector      domain.CollectorID        `json:"collector"`
	FetchedAt      *time.Time                `json:"fetchedAt"`
	Outcome        *domain.Outcome           `json:"outcome"`
	RawAnswer      string                    `json:"rawAnswer"`
	Appearances    []CompetitorAppearanceRes `json:"appearances"`
	Sources        []string                  `json:"sources"`
	ErrorCode      *string                   `json:"errorCode"`
	MentionVerdict *domain.MentionVerdict    `json:"mentionVerdict"`
}

func VisibilityResultToRes(v *domain.VisibilityResult) VisibilityResultRes {
	return VisibilityResultRes{
		ID:             v.ID,
		RunID:          v.RunID,
		PromptID:       v.PromptID,
		PromptText:     v.PromptText,
		PromptPosition: v.PromptPosition,
		Engine:         v.Engine,
		Collector:      v.Collector,
		FetchedAt:      v.FetchedAt,
		Outcome:        v.Outcome,
		RawAnswer:      v.RawAnswer,
		Appearances:    mapItems(v.Appearances, CompetitorAppearanceToRes),
		Sources:        v.Sources,
		ErrorCode:      v.ErrorCode,
		MentionVerdict: v.MentionVerdict,
	}
}

type PromptHistoryRunRes struct {
	Run     VisibilityRunRes      `json:"run"`
	Results []VisibilityResultRes `json:"results"`
}

func PromptHistoryRunToRes(v *domain.PromptHistoryRun) PromptHistoryRunRes {
	return PromptHistoryRunRes{
		Run:     VisibilityRunToRes(&v.Run),
		Results: mapItems(v.Results, VisibilityResultToRes),
	}
}

type TrackingStatusRes struct {
	Confirmed     bool       `json:"confirmed"`
	StartedAt     *time.Time `json:"startedAt"`
	NextRunAt     *time.Time `json:"nextRunAt"`
	BlockedReason *string    `json:"blockedReason"`
}

func TrackingStatusToRes(v *domain.TrackingStatus) TrackingStatusRes {
	return TrackingStatusRes{
		Confirmed:     v.Confirmed,
		StartedAt:     v.StartedAt,
		NextRunAt:     v.NextRunAt,
		BlockedReason: v.BlockedReason,
	}
}

type VisibilityHandler struct {
	querier   domain.VisibilityQuerier
	tracker   domain.TrackingCommander
	analytics *domain.VisibilityAnalyticsQuerier
}

func NewVisibilityHandler(q domain.VisibilityQuerier, tracker domain.TrackingCommander, analytics *domain.VisibilityAnalyticsQuerier) *VisibilityHandler {
	return &VisibilityHandler{querier: q, tracker: tracker, analytics: analytics}
}

func (h *VisibilityHandler) Routes() func(chi.Router) {
	return func(r chi.Router) {
		r.Get("/", List(brandFilter, h.querier.List, VisibilityRunToRes))
		r.Get("/{id}", Get(h.querier.Get, VisibilityRunToRes))
		r.Get("/{id}/results", List(runIDParam, h.querier.ListResults, VisibilityResultToRes))
	}
}

// HistoryRoutes mounts a prompt's saved runs and visibility, read from the saved results.
func (h *VisibilityHandler) HistoryRoutes() func(chi.Router) {
	return func(r chi.Router) {
		r.Get("/{promptID}/runs", List(promptHistoryParams, h.listPromptRuns, PromptHistoryRunToRes))
		r.Get("/{promptID}/visibility", Read(promptVisibilityParams, h.analytics.PromptVisibility, identity[domain.VisibilityReport]))
	}
}

func (h *VisibilityHandler) Tracking() http.HandlerFunc {
	return Read(brandFilter, h.tracking, TrackingStatusToRes)
}

func (h *VisibilityHandler) Summary() http.HandlerFunc {
	return Read(visibilityScopeParams, h.analytics.Summary, identity[domain.VisibilitySummary])
}

func (h *VisibilityHandler) StartTracking() http.HandlerFunc {
	return Command(h.startTracking)
}

func (h *VisibilityHandler) startTracking(ctx context.Context, req BrandRequestScope) error {
	if err := requireBrand(req.BrandID); err != nil {
		return err
	}

	return h.tracker.Start(ctx, req.BrandID)
}

func (h *VisibilityHandler) tracking(ctx context.Context, filter domain.BrandFilter) (*domain.TrackingStatus, error) {
	return h.querier.Tracking(ctx, filter.BrandID)
}

func (h *VisibilityHandler) listPromptRuns(ctx context.Context, params promptHistoryKey, page domain.PageParams) (*domain.Page[domain.PromptHistoryRun], error) {
	return h.querier.ListPromptRuns(ctx, params.brandID, params.promptID, page)
}

func runIDParam(r *http.Request) (uuid.UUID, error) {
	return pathUUID(r, "id")
}

type promptHistoryKey struct {
	brandID  uuid.UUID
	promptID uuid.UUID
}

func promptHistoryParams(r *http.Request) (promptHistoryKey, error) {
	brand, err := brandFilter(r)
	if err != nil {
		return promptHistoryKey{}, err
	}

	promptID, err := pathUUID(r, "promptID")
	if err != nil {
		return promptHistoryKey{}, err
	}

	return promptHistoryKey{brandID: brand.BrandID, promptID: promptID}, nil
}

func promptVisibilityParams(r *http.Request) (domain.PromptVisibilityScope, error) {
	key, err := promptHistoryParams(r)
	if err != nil {
		return domain.PromptVisibilityScope{}, err
	}

	days, engines, err := visibilityScope(r)
	if err != nil {
		return domain.PromptVisibilityScope{}, err
	}

	return domain.PromptVisibilityScope{
		VisibilityScope: domain.VisibilityScope{
			BrandFilter: domain.BrandFilter{BrandID: key.brandID},
			Days:        days,
			Engines:     engines,
		},
		PromptID: key.promptID,
	}, nil
}

func visibilityScopeParams(r *http.Request) (domain.VisibilityScope, error) {
	brand, err := brandFilter(r)
	if err != nil {
		return domain.VisibilityScope{}, err
	}

	days, engines, err := visibilityScope(r)
	if err != nil {
		return domain.VisibilityScope{}, err
	}

	return domain.VisibilityScope{
		BrandFilter: brand,
		Days:        days,
		Engines:     engines,
	}, nil
}

func visibilityScope(r *http.Request) (int, []domain.EngineID, error) {
	days := 0
	if raw := r.URL.Query().Get("days"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value == 0 {
			return 0, nil, domain.Validation(domain.FieldError{Field: "days", Code: domain.FieldInvalid})
		}

		days = value
	}

	var engines []domain.EngineID
	if raw := r.URL.Query().Get("engines"); raw != "" {
		for _, part := range strings.Split(raw, ",") {
			engines = append(engines, domain.EngineID(part))
		}
	}

	return visibilityScopeValues(days, engines)
}

func visibilityScopeValues(days int, selected []domain.EngineID) (int, []domain.EngineID, error) {
	if days == 0 {
		days = 30
	}

	if days != 7 && days != 30 && days != 90 {
		return 0, nil, domain.Validation(domain.FieldError{Field: "days", Code: domain.FieldInvalid})
	}

	var engines []domain.EngineID
	for _, engine := range selected {
		if !slices.Contains(domain.VisibilityEngines(), engine) {
			return 0, nil, domain.Validation(domain.FieldError{Field: "engines", Code: domain.FieldInvalid})
		}

		if !slices.Contains(engines, engine) {
			engines = append(engines, engine)
		}
	}

	return days, engines, nil
}
