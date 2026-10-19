package koidatabase

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/koidomain"
)

func TestVisibilityAnalyticsUsesCurrentCompetitorIdentity(t *testing.T) {
	db, store, ctx, brand := liveVisibilityFixture(t, true)
	prompts, err := store.PromptRepo().List(ctx, koidomain.PromptFilter{
		BrandID: brand.ID,
		Page:    koidomain.PageParams{Page: 1, PageSize: 10},
	})
	require.NoError(t, err)
	promptID := prompts.Items[0].ID
	run := finishVisibilityReviews(t, store, ctx, brand.ID)
	_, cells, err := store.BeginVisibilityAttempt(ctx, run.ID, *run.RiverJobID, 1)
	require.NoError(t, err)
	now := time.Now().UTC()

	first := appearanceCell(cells[0], "Acme", "https://acme.dev")
	first.Observations = koidomain.ExtractObservations(first)
	require.NoError(t, db.Save(&first).Error)
	second := appearanceCell(cells[1], "Shared", "")
	second.Observations = koidomain.ExtractObservations(second)
	require.NoError(t, db.Save(&second).Error)
	errorOutcome := koidomain.OutcomeError
	cells[2].Outcome = &errorOutcome
	cells[2].FetchedAt = &now
	errorCode := "provider_failed"
	cells[2].ErrorCode = &errorCode
	require.NoError(t, db.Save(&cells[2]).Error)
	require.NoError(t, db.Model(&koidomain.VisibilityRun{}).Where("id = ?", run.ID).
		Update("state", koidomain.VisibilityFailed).Error)

	previousRun := *run
	previousRun.ID = uuid.Nil
	previousRun.State = koidomain.VisibilitySucceeded
	previousRun.ScheduledFor = nil
	previousRun.RiverJobID = nil
	previousRun.CreatedAt = now.AddDate(0, 0, -10)
	previousRun.UpdatedAt = previousRun.CreatedAt
	require.NoError(t, db.Create(&previousRun).Error)
	previousCell := cells[0]
	previousCell.ID = uuid.Nil
	previousCell.RunID = previousRun.ID
	mentioned := koidomain.OutcomeMentioned
	previousCell.Outcome = &mentioned
	previousCell.FetchedAt = &now
	previousCell.RawAnswer = "Koi and Acme"
	previousCell.Sources = []string{}
	require.NoError(t, db.Create(&previousCell).Error)

	repo := store.VisibilityRepo().(koidomain.VisibilityAnalyticsRepository)
	read := func(engines []koidomain.EngineID) *koidomain.VisibilityReport {
		t.Helper()
		byPrompt, _, err := repo.AggregateVisibility(ctx, brand.ID, []uuid.UUID{promptID}, 7, engines, now)
		require.NoError(t, err)
		return byPrompt[promptID]
	}
	initial := read(nil)
	require.Equal(t, 2, initial.Metric.Completed)
	require.Equal(t, 0.0, *initial.Metric.Rate)
	require.Equal(t, 1, initial.Previous.Completed)
	require.Equal(t, 100.0, *initial.Previous.Rate)
	require.Nil(t, initial.Series[0].Metric.Rate)
	require.Empty(t, initial.Competitors)

	commander := koidomain.NewCompetitorCommander(store)
	acme, err := commander.Create(ctx, brand.ID, koidomain.CompetitorParams{Name: "Acme", Domain: "acme.dev"})
	require.NoError(t, err)
	shared, err := commander.Create(ctx, brand.ID, koidomain.CompetitorParams{Name: "Other", Domain: "other.dev", Aliases: []string{"Shared"}})
	require.NoError(t, err)
	report := read(nil)
	require.Equal(t, 1, report.Competitors[0].Metric.Cited)
	require.Equal(t, 1, report.Competitors[1].Metric.Mentioned)

	aliases := []string{"Shared"}
	_, err = commander.Patch(ctx, brand.ID, acme.ID, koidomain.CompetitorUpdate{Aliases: &aliases})
	require.NoError(t, err)
	report = read(nil)
	require.Equal(t, 1, report.Competitors[0].Metric.Cited)
	require.Zero(t, report.Competitors[0].Metric.Mentioned)
	require.Zero(t, report.Competitors[1].Metric.Mentioned)

	filtered := read([]koidomain.EngineID{cells[0].Engine})
	require.Equal(t, 1, filtered.Metric.Completed)
	require.Equal(t, 1, filtered.Competitors[0].Metric.Cited)
	secondPrompt, err := koidomain.NewPromptCommander(store).Create(ctx, brand.ID, koidomain.PromptParams{Text: "Another analytics prompt?"})
	require.NoError(t, err)
	var cellQueries atomic.Int32
	require.NoError(t, db.Callback().Query().Before("gorm:query").Register("analytics_cell_count", func(tx *gorm.DB) {
		if tx.Statement.Table == "visibility_results" {
			cellQueries.Add(1)
		}
	}))
	t.Cleanup(func() { _ = db.Callback().Query().Remove("analytics_cell_count") })
	_, _, err = repo.AggregateVisibility(ctx, brand.ID, []uuid.UUID{promptID, secondPrompt.ID}, 7, nil, now)
	require.NoError(t, err)
	require.EqualValues(t, 1, cellQueries.Load())
	require.NoError(t, db.Model(&koidomain.Prompt{}).Where("id = ?", promptID).
		Update("confirmed", false).Error)
	require.Equal(t, 2, read(nil).Metric.Completed)
	require.NoError(t, koidomain.NewPromptCommander(store).Delete(ctx, brand.ID, promptID))
	require.Equal(t, 2, read(nil).Metric.Completed)
	require.NoError(t, commander.Delete(ctx, brand.ID, acme.ID))
	require.NoError(t, commander.Delete(ctx, brand.ID, shared.ID))
	require.Empty(t, read(nil).Competitors)

	stranger := context.Background()
	_, _, err = koidomain.NewVisibilityAnalyticsQuerier(store).Aggregate(stranger, brand.ID, nil, 7, nil)
	require.Error(t, err)
}
