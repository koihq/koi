package database

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/domain"
)

func newVisibilityReport(start time.Time, days int, competitors []domain.Competitor) *domain.VisibilityReport {
	report := &domain.VisibilityReport{
		Series:      make([]domain.VisibilityDay, days),
		Engines:     make(map[domain.EngineID]domain.VisibilityMetric),
		Competitors: make([]domain.CompetitorVisibility, len(competitors)),
	}
	for day := range report.Series {
		date := start.AddDate(0, 0, day).Format("2006-01-02")
		report.Series[day].Date = date
	}
	for i, competitor := range competitors {
		report.Competitors[i].ID = competitor.ID
		report.Competitors[i].Name = competitor.Name
		report.Competitors[i].Domain = competitor.Domain
		report.Competitors[i].Series = make([]domain.VisibilityDay, days)
		for day := range report.Series {
			report.Competitors[i].Series[day].Date = report.Series[day].Date
		}
	}
	return report
}

func validVisibilityOutcome(outcome *domain.Outcome) bool {
	return outcome != nil && (*outcome == domain.OutcomeCited || *outcome == domain.OutcomeMentioned || *outcome == domain.OutcomeMiss)
}

func applyVisibilityCell(report *domain.VisibilityReport, cell domain.VisibilityResult, day int, competitorOutcomes map[uuid.UUID]domain.Outcome) {
	report.Metric.AddForAnalytics(*cell.Outcome)
	report.Series[day].Metric.AddForAnalytics(*cell.Outcome)

	engine := report.Engines[cell.Engine]
	engine.AddForAnalytics(*cell.Outcome)
	report.Engines[cell.Engine] = engine

	for i := range report.Competitors {
		competitor := &report.Competitors[i]
		outcome := competitorOutcomes[competitor.ID]
		if outcome == "" {
			outcome = domain.OutcomeMiss
		}
		competitor.Metric.AddForAnalytics(outcome)
		competitor.Series[day].Metric.AddForAnalytics(outcome)
	}
}

func competitorOutcomes(appearances []domain.CompetitorAppearance) map[uuid.UUID]domain.Outcome {
	outcomes := map[uuid.UUID]domain.Outcome{}
	for _, appearance := range appearances {
		if appearance.CompetitorID == nil {
			continue
		}

		outcome := domain.OutcomeMentioned
		for _, evidence := range appearance.Evidence {
			if evidence.Kind == "citation" {
				outcome = domain.OutcomeCited
				break
			}
		}
		outcomes[*appearance.CompetitorID] = outcome
	}

	return outcomes
}

var analyticsOutcomes = []domain.Outcome{
	domain.OutcomeCited,
	domain.OutcomeMentioned,
	domain.OutcomeMiss,
}

// visibilityWindow filters results by the denormalized run identity, without joining runs.
func (r *GormVisibilityRepository) visibilityWindow(ctx context.Context, brandID uuid.UUID, ids []uuid.UUID, engines []domain.EngineID, from, to time.Time) (*gorm.DB, error) {
	results := newBrandChildRepository[domain.VisibilityResult](r.db)
	query, err := results.query(ctx)
	if err != nil {
		return nil, err
	}

	query = query.Model(&domain.VisibilityResult{}).
		Where("brand_id = ? AND run_created_at >= ? AND run_created_at < ?", brandID, from, to).
		Where("outcome IN ?", analyticsOutcomes)
	if ids != nil {
		query = query.Where("prompt_id IN ?", ids)
	}
	if len(engines) > 0 {
		query = query.Where("engine IN ?", engines)
	}

	return query, nil
}

// applyPreviousVisibility counts the previous period in SQL; it needs no answer text.
func (r *GormVisibilityRepository) applyPreviousVisibility(ctx context.Context, overall *domain.VisibilityReport, byPrompt map[uuid.UUID]*domain.VisibilityReport, query *gorm.DB) error {
	var counts []struct {
		PromptID uuid.UUID
		Outcome  domain.Outcome
		Count    int
	}
	if err := query.Select("prompt_id, outcome, count(*) AS count").Group("prompt_id, outcome").Scan(&counts).Error; err != nil {
		return mapDBError(err)
	}

	for _, count := range counts {
		overall.Previous.AddOutcomes(count.Outcome, count.Count)
		if report := byPrompt[count.PromptID]; report != nil {
			report.Previous.AddOutcomes(count.Outcome, count.Count)
		}
	}

	return nil
}

func (r *GormVisibilityRepository) runSnapshots(ctx context.Context, brandID uuid.UUID, from, to time.Time) (map[uuid.UUID]domain.BrandSnapshot, error) {
	query, err := r.query(ctx)
	if err != nil {
		return nil, err
	}

	var runs []struct {
		ID            uuid.UUID
		BrandSnapshot domain.BrandSnapshot `gorm:"type:jsonb;serializer:json"`
	}
	if err := query.Model(&domain.VisibilityRun{}).
		Select("id, brand_snapshot").
		Where("brand_id = ? AND created_at >= ? AND created_at < ?", brandID, from, to).
		Scan(&runs).Error; err != nil {
		return nil, mapDBError(err)
	}

	snapshots := make(map[uuid.UUID]domain.BrandSnapshot, len(runs))
	for _, run := range runs {
		snapshots[run.ID] = run.BrandSnapshot
	}

	return snapshots, nil
}

// AggregateVisibility counts the previous period in SQL and matches current cells once per request.
// Current competitor identities are applied at read time.
func (r *GormVisibilityRepository) AggregateVisibility(ctx context.Context, brandID uuid.UUID, ids []uuid.UUID, days int, engines []domain.EngineID, now time.Time) (map[uuid.UUID]*domain.VisibilityReport, *domain.VisibilityReport, error) {
	query, err := NewCompetitorRepository(r.db).query(ctx)
	if err != nil {
		return nil, nil, err
	}
	var competitors []domain.Competitor
	if err := query.Where("brand_id = ? AND confirmed = true", brandID).Order("created_at, id").Find(&competitors).Error; err != nil {
		return nil, nil, mapDBError(err)
	}

	today := now.UTC().Truncate(24 * time.Hour)
	start := today.AddDate(0, 0, 1-days)
	previousStart := start.AddDate(0, 0, -days)
	end := today.AddDate(0, 0, 1)

	overall := newVisibilityReport(start, days, competitors)
	byPrompt := make(map[uuid.UUID]*domain.VisibilityReport, len(ids))
	for _, id := range ids {
		byPrompt[id] = newVisibilityReport(start, days, competitors)
	}

	if ids != nil && len(ids) == 0 {
		return byPrompt, overall, nil
	}

	previous, err := r.visibilityWindow(ctx, brandID, ids, engines, previousStart, start)
	if err != nil {
		return nil, nil, err
	}
	if err := r.applyPreviousVisibility(ctx, overall, byPrompt, previous); err != nil {
		return nil, nil, err
	}

	current, err := r.visibilityWindow(ctx, brandID, ids, engines, start, end)
	if err != nil {
		return nil, nil, err
	}
	columns := "prompt_id, engine, outcome, run_created_at"
	if len(competitors) > 0 {
		columns += ", run_id, raw_answer, sources, observations"
	}
	var cells []domain.VisibilityResult
	if err := current.Select(columns).Find(&cells).Error; err != nil {
		return nil, nil, mapDBError(err)
	}

	var matcher *domain.AppearanceMatcher
	var snapshots map[uuid.UUID]domain.BrandSnapshot
	if len(competitors) > 0 {
		matcher = domain.NewAppearanceMatcher(competitors)
		snapshots, err = r.runSnapshots(ctx, brandID, start, end)
		if err != nil {
			return nil, nil, err
		}
	}

	for _, cell := range cells {
		if !validVisibilityOutcome(cell.Outcome) {
			continue
		}

		day := int(cell.RunCreatedAt.UTC().Truncate(24*time.Hour).Sub(start) / (24 * time.Hour))
		outcomes := map[uuid.UUID]domain.Outcome{}
		if matcher != nil {
			outcomes = competitorOutcomes(matcher.Match(cell, snapshots[cell.RunID]))
		}

		applyVisibilityCell(overall, cell, day, outcomes)
		if report := byPrompt[cell.PromptID]; report != nil {
			applyVisibilityCell(report, cell, day, outcomes)
		}
	}

	return byPrompt, overall, nil
}

var _ domain.VisibilityAnalyticsRepository = (*GormVisibilityRepository)(nil)
