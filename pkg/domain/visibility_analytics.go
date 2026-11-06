package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type VisibilityMetric struct {
	Completed int      `json:"completed"`
	Cited     int      `json:"cited"`
	Mentioned int      `json:"mentioned"`
	Rate      *float64 `json:"rate"`
}

func (m *VisibilityMetric) AddForAnalytics(outcome Outcome) {
	m.AddOutcomes(outcome, 1)
}

// AddOutcomes adds count cells with the same outcome and recomputes the rate once.
func (m *VisibilityMetric) AddOutcomes(outcome Outcome, count int) {
	if count <= 0 {
		return
	}

	m.Completed += count
	switch outcome {
	case OutcomeCited:
		m.Cited += count
	case OutcomeMentioned:
		m.Mentioned += count
	}

	rate := 100 * float64(m.Cited+m.Mentioned) / float64(m.Completed)
	m.Rate = &rate
}

type VisibilityDay struct {
	Date   string           `json:"date"`
	Metric VisibilityMetric `json:"metric"`
}

type CompetitorVisibility struct {
	ID     uuid.UUID        `json:"id"`
	Name   string           `json:"name"`
	Domain string           `json:"domain"`
	Metric VisibilityMetric `json:"metric"`
	Series []VisibilityDay  `json:"series"`
}

type VisibilityReport struct {
	Metric      VisibilityMetric              `json:"metric"`
	Previous    VisibilityMetric              `json:"previous"`
	Series      []VisibilityDay               `json:"series"`
	Engines     map[EngineID]VisibilityMetric `json:"engines"`
	Competitors []CompetitorVisibility        `json:"competitors"`
}

type VisibilitySummary struct {
	*VisibilityReport
	HasConfirmedPrompts bool `json:"hasConfirmedPrompts"`
}

// VisibilityScope selects the period and engines of a brand's report.
type VisibilityScope struct {
	BrandFilter
	Days    int
	Engines []EngineID
}

// PromptVisibilityScope narrows a report to one prompt snapshot.
type PromptVisibilityScope struct {
	VisibilityScope
	PromptID uuid.UUID
}

type VisibilityAnalyticsRepository interface {
	AggregateVisibility(context.Context, uuid.UUID, []uuid.UUID, int, []EngineID, time.Time) (map[uuid.UUID]*VisibilityReport, *VisibilityReport, error)
}

type VisibilityAnalyticsQuerier struct{ store Store }

func NewVisibilityAnalyticsQuerier(store Store) *VisibilityAnalyticsQuerier {
	return &VisibilityAnalyticsQuerier{store: store}
}

func (q *VisibilityAnalyticsQuerier) Aggregate(ctx context.Context, brandID uuid.UUID, ids []uuid.UUID, days int, engines []EngineID) (map[uuid.UUID]*VisibilityReport, *VisibilityReport, error) {
	if err := ownedBrand(ctx, q.store, brandID); err != nil {
		return nil, nil, err
	}

	return q.store.VisibilityRepo().AggregateVisibility(ctx, brandID, ids, days, engines, time.Now().UTC())
}

func (q *VisibilityAnalyticsQuerier) Summary(ctx context.Context, scope VisibilityScope) (*VisibilitySummary, error) {
	_, report, err := q.Aggregate(ctx, scope.BrandID, nil, scope.Days, scope.Engines)
	if err != nil {
		return nil, err
	}

	hasPrompts, err := q.store.PromptRepo().HasConfirmed(ctx, scope.BrandID)
	if err != nil {
		return nil, err
	}

	return &VisibilitySummary{
		VisibilityReport:    report,
		HasConfirmedPrompts: hasPrompts,
	}, nil
}

// PromptVisibility reports on a prompt that exists now or in the brand's history.
func (q *VisibilityAnalyticsQuerier) PromptVisibility(ctx context.Context, scope PromptVisibilityScope) (*VisibilityReport, error) {
	if err := ownedBrand(ctx, q.store, scope.BrandID); err != nil {
		return nil, err
	}

	_, err := q.store.VisibilityRepo().ListPromptRuns(ctx, scope.BrandID, scope.PromptID, PageParams{Page: 1, PageSize: 1})
	if err != nil {
		return nil, err
	}

	byPrompt, _, err := q.Aggregate(ctx, scope.BrandID, []uuid.UUID{scope.PromptID}, scope.Days, scope.Engines)
	if err != nil {
		return nil, err
	}

	return byPrompt[scope.PromptID], nil
}
