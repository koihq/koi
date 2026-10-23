package koidatabase

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/koihq/koi/pkg/koidomain"
)

func newVisibilityReport(start time.Time, days int, competitors []koidomain.Competitor) *koidomain.VisibilityReport {
	report := &koidomain.VisibilityReport{
		Series:      make([]koidomain.VisibilityDay, days),
		Engines:     make(map[koidomain.EngineID]koidomain.VisibilityMetric),
		Competitors: make([]koidomain.CompetitorVisibility, len(competitors)),
	}
	for day := range report.Series {
		date := start.AddDate(0, 0, day).Format("2006-01-02")
		report.Series[day].Date = date
	}
	for i, competitor := range competitors {
		report.Competitors[i].ID = competitor.ID
		report.Competitors[i].Series = make([]koidomain.VisibilityDay, days)
		for day := range report.Series {
			report.Competitors[i].Series[day].Date = report.Series[day].Date
		}
	}
	return report
}

func validVisibilityOutcome(outcome *koidomain.Outcome) bool {
	return outcome != nil && (*outcome == koidomain.OutcomeCited || *outcome == koidomain.OutcomeMentioned || *outcome == koidomain.OutcomeMiss)
}

func applyVisibilityCell(report *koidomain.VisibilityReport, cell koidomain.VisibilityResult, day int, current bool, competitorOutcomes map[uuid.UUID]koidomain.Outcome) {
	if !current {
		report.Previous.AddForAnalytics(*cell.Outcome)
		return
	}

	report.Metric.AddForAnalytics(*cell.Outcome)
	report.Series[day].Metric.AddForAnalytics(*cell.Outcome)
	engine := report.Engines[cell.Engine]
	engine.AddForAnalytics(*cell.Outcome)
	report.Engines[cell.Engine] = engine
	for i := range report.Competitors {
		competitor := &report.Competitors[i]
		outcome := competitorOutcomes[competitor.ID]
		if outcome == "" {
			outcome = koidomain.OutcomeMiss
		}
		competitor.Metric.AddForAnalytics(outcome)
		competitor.Series[day].Metric.AddForAnalytics(outcome)
	}
}

// AggregateVisibility reads both periods in one cell query. Current identities are applied at read time.
func (r *GormVisibilityRepository) AggregateVisibility(ctx context.Context, brandID uuid.UUID, ids []uuid.UUID, days int, engines []koidomain.EngineID, now time.Time) (map[uuid.UUID]*koidomain.VisibilityReport, *koidomain.VisibilityReport, error) {
	query, err := NewGormCompetitorRepository(r.db).query(ctx)
	if err != nil {
		return nil, nil, err
	}
	var competitors []koidomain.Competitor
	if err := query.Where("brand_id = ? AND confirmed = true", brandID).Order("created_at, id").Find(&competitors).Error; err != nil {
		return nil, nil, mapDBError(err)
	}

	today := now.UTC().Truncate(24 * time.Hour)
	start := today.AddDate(0, 0, 1-days)
	previousStart := start.AddDate(0, 0, -days)
	end := today.AddDate(0, 0, 1)
	overall := newVisibilityReport(start, days, competitors)
	byPrompt := make(map[uuid.UUID]*koidomain.VisibilityReport, len(ids))
	for _, id := range ids {
		byPrompt[id] = newVisibilityReport(start, days, competitors)
	}

	if ids != nil && len(ids) == 0 {
		return byPrompt, overall, nil
	}
	cellsQuery, err := r.query(ctx)
	if err != nil {
		return nil, nil, err
	}

	cellsQuery = cellsQuery.Model(&koidomain.VisibilityResult{}).
		Joins("JOIN visibility_runs ON visibility_runs.id = visibility_results.run_id").
		Where("visibility_runs.brand_id = ? AND visibility_runs.created_at >= ? AND visibility_runs.created_at < ?", brandID, previousStart, end).
		Where("visibility_results.outcome IN ?", []koidomain.Outcome{koidomain.OutcomeCited, koidomain.OutcomeMentioned, koidomain.OutcomeMiss}).
		Select("visibility_results.*, visibility_runs.created_at AS run_created_at, visibility_runs.brand_snapshot AS run_snapshot")
	if ids != nil {
		cellsQuery = cellsQuery.Where("visibility_results.prompt_id IN ?", ids)
	}
	if len(engines) > 0 {
		cellsQuery = cellsQuery.Where("visibility_results.engine IN ?", engines)
	}
	var cells []struct {
		koidomain.VisibilityResult
		RunCreatedAt time.Time
		RunSnapshot  koidomain.BrandSnapshot `gorm:"type:jsonb;serializer:json"`
	}
	if err := cellsQuery.Find(&cells).Error; err != nil {
		return nil, nil, mapDBError(err)
	}

	for _, row := range cells {
		cell := row.VisibilityResult
		if !validVisibilityOutcome(cell.Outcome) {
			continue
		}
		current := !row.RunCreatedAt.Before(start)
		day := int(row.RunCreatedAt.UTC().Truncate(24*time.Hour).Sub(start) / (24 * time.Hour))
		competitorOutcomes := map[uuid.UUID]koidomain.Outcome{}
		if current && len(competitors) > 0 {
			for _, appearance := range koidomain.MatchAppearances(cell, competitors, row.RunSnapshot) {
				if appearance.CompetitorID == nil {
					continue
				}
				outcome := koidomain.OutcomeMentioned
				for _, evidence := range appearance.Evidence {
					if evidence.Kind == "citation" {
						outcome = koidomain.OutcomeCited
						break
					}
				}
				competitorOutcomes[*appearance.CompetitorID] = outcome
			}
		}
		applyVisibilityCell(overall, cell, day, current, competitorOutcomes)
		if report := byPrompt[cell.PromptID]; report != nil {
			applyVisibilityCell(report, cell, day, current, competitorOutcomes)
		}
	}

	return byPrompt, overall, nil
}

var _ koidomain.VisibilityAnalyticsRepository = (*GormVisibilityRepository)(nil)
