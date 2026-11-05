package database

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/domain"
)

func TestVisibilityAnalyticsUsesCurrentCompetitorIdentity(t *testing.T) {
	db, store, ctx, brand := liveVisibilityFixture(t, true)
	prompts, err := store.PromptRepo().List(ctx, domain.PromptFilter{
		BrandFilter: domain.BrandFilter{BrandID: brand.ID},
	}, domain.PageParams{Page: 1, PageSize: 10})
	require.NoError(t, err)
	promptID := prompts.Items[0].ID
	run := finishVisibilityReviews(t, store, ctx, brand.ID)
	_, cells, err := store.BeginVisibilityAttempt(ctx, run.ID, *run.RiverJobID, 1)
	require.NoError(t, err)
	var persistedRun domain.VisibilityRun
	require.NoError(t, db.First(&persistedRun, "id = ?", run.ID).Error)
	for _, cell := range cells {
		require.Equal(t, brand.ID, cell.BrandID)
		require.True(t, cell.RunCreatedAt.Equal(persistedRun.CreatedAt))
	}
	now := time.Now().UTC()

	first := appearanceCell(cells[0], "Acme", "https://acme.dev")
	first.Observations = domain.ExtractObservations(first)
	require.NoError(t, db.Save(&first).Error)
	second := appearanceCell(cells[1], "Shared", "")
	second.Observations = domain.ExtractObservations(second)
	require.NoError(t, db.Save(&second).Error)
	errorOutcome := domain.OutcomeError
	cells[2].Outcome = &errorOutcome
	cells[2].FetchedAt = &now
	errorCode := "provider_failed"
	cells[2].ErrorCode = &errorCode
	require.NoError(t, db.Save(&cells[2]).Error)
	require.NoError(t, db.Model(&domain.VisibilityRun{}).Where("id = ?", run.ID).
		Update("state", domain.VisibilityFailed).Error)

	previousRun := *run
	previousRun.ID = uuid.Nil
	previousRun.State = domain.VisibilitySucceeded
	previousRun.ScheduledFor = nil
	previousRun.RiverJobID = nil
	previousRun.CreatedAt = now.AddDate(0, 0, -10)
	previousRun.UpdatedAt = previousRun.CreatedAt
	require.NoError(t, db.Create(&previousRun).Error)
	previousCell := cells[0]
	previousCell.ID = uuid.Nil
	previousCell.RunID = previousRun.ID
	previousCell.RunCreatedAt = previousRun.CreatedAt
	mentioned := domain.OutcomeMentioned
	previousCell.Outcome = &mentioned
	previousCell.FetchedAt = &now
	previousCell.RawAnswer = "Koi and Acme"
	previousCell.Sources = []string{}
	require.NoError(t, db.Create(&previousCell).Error)

	repo := store.VisibilityRepo()
	read := func(engines []domain.EngineID) *domain.VisibilityReport {
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

	commander := domain.NewCompetitorCommander(store)
	acme, err := commander.Create(ctx, brand.ID, domain.CreateCompetitorParams{Name: "Acme", Domain: "acme.dev"})
	require.NoError(t, err)
	shared, err := commander.Create(ctx, brand.ID, domain.CreateCompetitorParams{Name: "Other", Domain: "other.dev", Aliases: []string{"Shared"}})
	require.NoError(t, err)
	report := read(nil)
	require.Equal(t, "Acme", report.Competitors[0].Name)
	require.Equal(t, "acme.dev", report.Competitors[0].Domain)
	require.Equal(t, 1, report.Competitors[0].Metric.Cited)
	require.Equal(t, 1, report.Competitors[1].Metric.Mentioned)

	aliases := []string{"Shared"}
	_, err = commander.Update(ctx, acme.ID, domain.UpdateCompetitorParams{Aliases: &aliases})
	require.NoError(t, err)
	report = read(nil)
	require.Equal(t, 1, report.Competitors[0].Metric.Cited)
	require.Zero(t, report.Competitors[0].Metric.Mentioned)
	require.Zero(t, report.Competitors[1].Metric.Mentioned)
	newName := "Acme renamed"
	_, err = commander.Update(ctx, acme.ID, domain.UpdateCompetitorParams{Name: &newName})
	require.NoError(t, err)
	require.Equal(t, newName, read(nil).Competitors[0].Name)

	filtered := read([]domain.EngineID{cells[0].Engine})
	require.Equal(t, 1, filtered.Metric.Completed)
	require.Equal(t, 1, filtered.Competitors[0].Metric.Cited)
	require.Equal(t, 1, filtered.Previous.Completed)
	require.Equal(t, 1, filtered.Previous.Mentioned)
	otherEngine := read([]domain.EngineID{cells[1].Engine})
	require.Equal(t, 1, otherEngine.Metric.Completed)
	require.Zero(t, otherEngine.Previous.Completed)
	require.Nil(t, otherEngine.Previous.Rate)
	secondPrompt, err := domain.NewPromptCommander(store).Create(ctx, brand.ID, domain.CreatePromptParams{Text: "Another analytics prompt?"})
	require.NoError(t, err)
	var cellQueries atomic.Int32
	var totalQueries atomic.Int32
	countQuery := func(tx *gorm.DB) {
		totalQueries.Add(1)
		if tx.Statement.Table == "visibility_results" {
			cellQueries.Add(1)
		}
	}
	require.NoError(t, db.Callback().Query().Before("gorm:query").Register("analytics_cell_count", countQuery))
	require.NoError(t, db.Callback().Row().Before("gorm:row").Register("analytics_cell_count", countQuery))
	t.Cleanup(func() {
		_ = db.Callback().Query().Remove("analytics_cell_count")
		_ = db.Callback().Row().Remove("analytics_cell_count")
	})
	_, _, err = repo.AggregateVisibility(ctx, brand.ID, []uuid.UUID{promptID, secondPrompt.ID}, 7, nil, now)
	require.NoError(t, err)
	// Competitors, previous-period counts, current cells, and run snapshots: no per-prompt queries.
	require.EqualValues(t, 2, cellQueries.Load())
	require.EqualValues(t, 4, totalQueries.Load())
	require.NoError(t, db.Model(&domain.Prompt{}).Where("id = ?", promptID).
		Update("confirmed", false).Error)
	require.Equal(t, 2, read(nil).Metric.Completed)
	require.NoError(t, store.PromptRepo().Delete(ctx, promptID))
	require.Equal(t, 2, read(nil).Metric.Completed)
	require.NoError(t, store.PromptRepo().Delete(ctx, secondPrompt.ID))
	summary, err := domain.NewVisibilityAnalyticsQuerier(store).Summary(ctx, domain.VisibilityScope{BrandFilter: domain.BrandFilter{BrandID: brand.ID}, Days: 7})
	require.NoError(t, err)
	require.False(t, summary.HasConfirmedPrompts)
	require.Equal(t, 2, summary.Metric.Completed)
	require.NoError(t, commander.Delete(ctx, acme.ID))
	require.NoError(t, commander.Delete(ctx, shared.ID))
	require.Empty(t, read(nil).Competitors)

	stranger := context.Background()
	_, _, err = domain.NewVisibilityAnalyticsQuerier(store).Aggregate(stranger, brand.ID, nil, 7, nil)
	require.Error(t, err)
}

func TestVisibilitySummaryIncludesAllConfirmedCompetitorsAndPromptState(t *testing.T) {
	db, store, ctx, brand := liveVisibilityFixture(t, false)
	competitors := make([]domain.Competitor, 122)
	for i := range competitors {
		competitors[i] = domain.Competitor{
			BrandID:   brand.ID,
			Name:      fmt.Sprintf("Competitor %d", i),
			Domain:    fmt.Sprintf("competitor%d.example", i),
			Source:    domain.CompetitorSourceManual,
			Confirmed: i < 121,
		}
	}
	require.NoError(t, db.Create(&competitors).Error)
	querier := domain.NewVisibilityAnalyticsQuerier(store)

	for _, tt := range []struct {
		name      string
		confirmed bool
		inactive  bool
	}{
		{name: "confirmed", confirmed: true},
		{name: "paused", confirmed: true, inactive: true},
		{name: "unconfirmed"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, db.Model(&domain.Prompt{}).Where("brand_id = ?", brand.ID).
				Updates(map[string]any{"confirmed": tt.confirmed, "inactive": tt.inactive}).Error)

			summary, err := querier.Summary(ctx, domain.VisibilityScope{BrandFilter: domain.BrandFilter{BrandID: brand.ID}, Days: 7})
			require.NoError(t, err)
			require.Equal(t, tt.confirmed, summary.HasConfirmedPrompts)
			require.Len(t, summary.Competitors, 121)
			for _, competitor := range summary.Competitors {
				require.NotEmpty(t, competitor.Name)
				require.NotEmpty(t, competitor.Domain)
				require.Nil(t, competitor.Metric.Rate)
				require.Len(t, competitor.Series, 7)
			}
		})
	}

	require.NoError(t, db.Where("brand_id = ?", brand.ID).Delete(&domain.Prompt{}).Error)
	summary, err := querier.Summary(ctx, domain.VisibilityScope{BrandFilter: domain.BrandFilter{BrandID: brand.ID}, Days: 90, Engines: []domain.EngineID{domain.EngineChatGPT}})
	require.NoError(t, err)
	require.False(t, summary.HasConfirmedPrompts)
	require.Len(t, summary.Series, 90)
}

func TestVisibilityAnalyticsRepositoryScopesCellsWithoutDomainGuard(t *testing.T) {
	db, store, ctx, brand := liveVisibilityFixture(t, true)
	run := finishVisibilityReviews(t, store, ctx, brand.ID)
	_, cells, err := store.BeginVisibilityAttempt(ctx, run.ID, *run.RiverJobID, 1)
	require.NoError(t, err)
	now := time.Now().UTC()
	outcome := domain.OutcomeMentioned
	cell := cells[0]
	cell.Outcome = &outcome
	cell.FetchedAt = &now
	cell.RawAnswer = "Private brand answer"
	require.NoError(t, db.Save(&cell).Error)

	user := &domain.User{Email: uuid.NewString() + "@example.com", PasswordHash: "hash"}
	require.NoError(t, store.UserRepo().Create(t.Context(), user))
	testOrganizationForUser(t, store, user.ID)
	foreignContext := testIdentity(t, store, user.ID)
	hasPrompts, err := store.PromptRepo().HasConfirmed(foreignContext, brand.ID)
	require.NoError(t, err)
	require.False(t, hasPrompts)

	_, ownerReport, err := store.VisibilityRepo().AggregateVisibility(ctx, brand.ID, nil, 7, nil, now)
	require.NoError(t, err)
	require.Equal(t, 1, ownerReport.Metric.Completed)

	byPrompt, foreignReport, err := store.VisibilityRepo().AggregateVisibility(foreignContext, brand.ID, []uuid.UUID{cell.PromptID}, 7, nil, now)
	require.NoError(t, err)
	require.Zero(t, foreignReport.Metric.Completed)
	require.Nil(t, foreignReport.Metric.Rate)
	require.Zero(t, byPrompt[cell.PromptID].Metric.Completed)
	require.Empty(t, foreignReport.Competitors)
}
