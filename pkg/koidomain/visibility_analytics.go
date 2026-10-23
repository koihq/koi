package koidomain

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
	m.Completed++
	switch outcome {
	case OutcomeCited:
		m.Cited++
	case OutcomeMentioned:
		m.Mentioned++
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
