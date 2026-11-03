package koidatabase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

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
		report.Competitors[i].Name = competitor.Name
		report.Competitors[i].Domain = competitor.Domain
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

func applyVisibilityCell(report *koidomain.VisibilityReport, cell koidomain.VisibilityResult, day int, competitorOutcomes map[uuid.UUID]koidomain.Outcome) {
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

func competitorOutcomes(appearances []koidomain.CompetitorAppearance) map[uuid.UUID]koidomain.Outcome {
	outcomes := map[uuid.UUID]koidomain.Outcome{}
	for _, appearance := range appearances {
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
		outcomes[*appearance.CompetitorID] = outcome
	}

	return outcomes
}

var analyticsOutcomes = []koidomain.Outcome{
	koidomain.OutcomeCited,
	koidomain.OutcomeMentioned,
	koidomain.OutcomeMiss,
}

// visibilityWindow filters results by the denormalized run identity, without joining runs.
func (r *GormVisibilityRepository) visibilityWindow(ctx context.Context, brandID uuid.UUID, ids []uuid.UUID, engines []koidomain.EngineID, from, to time.Time) (*gorm.DB, error) {
	results := newScopedRepository[koidomain.VisibilityResult](r.db, childOwnerScope("visibility_results"))
	query, err := results.query(ctx)
	if err != nil {
		return nil, err
	}

	query = query.Model(&koidomain.VisibilityResult{}).
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
func (r *GormVisibilityRepository) applyPreviousVisibility(ctx context.Context, overall *koidomain.VisibilityReport, byPrompt map[uuid.UUID]*koidomain.VisibilityReport, query *gorm.DB) error {
	var counts []struct {
		PromptID uuid.UUID
		Outcome  koidomain.Outcome
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

func (r *GormVisibilityRepository) runSnapshots(ctx context.Context, brandID uuid.UUID, from, to time.Time) (map[uuid.UUID]koidomain.BrandSnapshot, error) {
	query, err := r.query(ctx)
	if err != nil {
		return nil, err
	}

	var runs []struct {
		ID            uuid.UUID
		BrandSnapshot koidomain.BrandSnapshot `gorm:"type:jsonb;serializer:json"`
	}
	if err := query.Model(&koidomain.VisibilityRun{}).
		Select("id, brand_snapshot").
		Where("brand_id = ? AND created_at >= ? AND created_at < ?", brandID, from, to).
		Scan(&runs).Error; err != nil {
		return nil, mapDBError(err)
	}

	snapshots := make(map[uuid.UUID]koidomain.BrandSnapshot, len(runs))
	for _, run := range runs {
		snapshots[run.ID] = run.BrandSnapshot
	}

	return snapshots, nil
}

// AggregateVisibility counts the previous period in SQL and matches current cells once per request.
// Current competitor identities are applied at read time.
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
	var cells []koidomain.VisibilityResult
	if err := current.Select(columns).Find(&cells).Error; err != nil {
		return nil, nil, mapDBError(err)
	}

	var matcher *koidomain.AppearanceMatcher
	var snapshots map[uuid.UUID]koidomain.BrandSnapshot
	if len(competitors) > 0 {
		matcher = koidomain.NewAppearanceMatcher(competitors)
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
		outcomes := map[uuid.UUID]koidomain.Outcome{}
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

var _ koidomain.VisibilityAnalyticsRepository = (*GormVisibilityRepository)(nil)
