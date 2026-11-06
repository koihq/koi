package database

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/auth"
	"github.com/koihq/koi/pkg/domain"
)

func appearanceCell(cell domain.VisibilityResult, name, link string) domain.VisibilityResult {
	now := time.Now().UTC()
	miss := domain.OutcomeMiss
	cell.Outcome, cell.FetchedAt = &miss, &now
	cell.RawAnswer = name + " is an option."
	cell.NamedEntities = []domain.NamedEntity{{Type: "software", Name: name}}
	if link != "" {
		cell.AnswerMarkdown = "1. [" + name + "](" + link + ")"
		cell.Sources = []string{link + "/docs", link + "/about"}
	}
	return cell
}

func TestObservedCompetitorsCountDistinctCellsAndNeverAutoConfirm(t *testing.T) {
	db, store, ctx, brand := liveVisibilityFixture(t, true)
	run := startVisibilityTracking(t, store, ctx, brand.ID)
	_, cells, err := store.BeginVisibilityAttempt(ctx, run.ID, *run.RiverJobID, 1)
	require.NoError(t, err)

	// Racing deliveries of the same cell still count once.
	var group sync.WaitGroup
	errors := make(chan error, 2)
	for range 2 {
		group.Go(func() {
			errors <- store.SaveVisibilityResult(ctx, run.ID, 1, appearanceCell(cells[0], "Acme", "https://acme.dev"))
		})
	}
	group.Wait()
	for range 2 {
		require.NoError(t, <-errors)
	}
	var state domain.CompetitorObservation
	require.NoError(t, db.First(&state, "brand_id = ?", brand.ID).Error)
	require.EqualValues(t, 1, state.Count)
	competitors, err := store.CompetitorRepo().List(ctx, domain.CompetitorFilter{
		BrandFilter: domain.BrandFilter{BrandID: brand.ID},
	}, domain.PageParams{
		Page:     1,
		PageSize: 10,
	})
	require.NoError(t, err)
	require.Empty(t, competitors.Items)

	// Concurrent independent cells create one suggestion under the brand lock.
	for _, cell := range cells[1:3] {
		group.Go(func() {
			errors <- store.SaveVisibilityResult(ctx, run.ID, 1, appearanceCell(cell, "Acme", "https://acme.dev"))
		})
	}
	group.Wait()
	for range 2 {
		require.NoError(t, <-errors)
	}
	require.NoError(t, db.First(&state, "brand_id = ?", brand.ID).Error)
	require.EqualValues(t, 3, state.Count)
	competitors, err = store.CompetitorRepo().List(ctx, domain.CompetitorFilter{
		BrandFilter: domain.BrandFilter{BrandID: brand.ID},
	}, domain.PageParams{
		Page:     1,
		PageSize: 10,
	})
	require.NoError(t, err)
	require.Len(t, competitors.Items, 1)
	suggestion := competitors.Items[0]
	require.Equal(t, domain.CompetitorSourceObserved, suggestion.Source)
	require.Equal(t, "acme.dev", suggestion.Domain)
	require.False(t, suggestion.Confirmed)
	require.NoError(t, store.VisibilityRepo().StartTracking(ctx, brand.ID))
	unchanged, err := store.CompetitorRepo().Get(ctx, suggestion.ID)
	require.NoError(t, err)
	require.False(t, unchanged.Confirmed)
	tracking, err := store.VisibilityRepo().Tracking(ctx, brand.ID)
	require.NoError(t, err)
	require.True(t, tracking.Confirmed)
	require.Nil(t, tracking.BlockedReason)

	read, err := store.VisibilityRepo().ListResults(ctx, run.ID, domain.PageParams{Page: 1, PageSize: 10})
	require.NoError(t, err)
	for _, cell := range read.Items {
		if cell.Outcome != nil {
			require.Equal(t, domain.OutcomeMiss, *cell.Outcome)
			require.Len(t, cell.Appearances, 1)
			require.Equal(t, suggestion.ID, *cell.Appearances[0].CompetitorID)
			require.False(t, cell.Appearances[0].Confirmed)
			require.Contains(t, cell.Appearances[0].Evidence, domain.AppearanceEvidence{Kind: "citation", Value: "https://acme.dev/docs"})
			require.Contains(t, cell.Appearances[0].Evidence, domain.AppearanceEvidence{Kind: "citation", Value: "https://acme.dev/about"})
		} else {
			require.NotNil(t, cell.Appearances)
		}
	}
}

func TestNameOnlySuggestionNeedsIdentityAndHistoricalLinksFollowEdits(t *testing.T) {
	_, store, ctx, brand := liveVisibilityFixture(t, true)
	run := startVisibilityTracking(t, store, ctx, brand.ID)
	_, cells, err := store.BeginVisibilityAttempt(ctx, run.ID, *run.RiverJobID, 1)
	require.NoError(t, err)
	for _, cell := range cells[:2] {
		require.NoError(t, store.SaveVisibilityResult(ctx, run.ID, 1, appearanceCell(cell, "Acme", "")))
	}
	competitors, err := store.CompetitorRepo().List(ctx, domain.CompetitorFilter{
		BrandFilter: domain.BrandFilter{BrandID: brand.ID},
	}, domain.PageParams{
		Page:     1,
		PageSize: 10,
	})
	require.NoError(t, err)
	require.Len(t, competitors.Items, 1)
	suggestion := competitors.Items[0]
	commander := domain.NewCompetitorCommander(store)
	_, err = commander.Confirm(ctx, suggestion.ID)
	require.ErrorIs(t, err, domain.ErrValidation)

	// This plain text has no discovery hints; an alias added later attaches it on read.
	cell := appearanceCell(cells[2], "Alternate", "")
	cell.NamedEntities = nil
	require.NoError(t, store.SaveVisibilityResult(ctx, run.ID, 1, cell))
	name, website := "Acme Analytics", "acme.dev"
	aliases := []string{"Acme", "Alternate"}
	updated, err := commander.Update(ctx, suggestion.ID, domain.UpdateCompetitorParams{Name: &name, Domain: &website, Aliases: &aliases})
	require.NoError(t, err)
	require.False(t, updated.Confirmed)
	_, err = commander.Confirm(ctx, suggestion.ID)
	require.NoError(t, err)
	read, err := store.VisibilityRepo().ListResults(ctx, run.ID, domain.PageParams{Page: 1, PageSize: 10})
	require.NoError(t, err)
	for _, result := range read.Items {
		if result.Outcome == nil {
			continue
		}
		require.Len(t, result.Appearances, 1)
		require.Equal(t, "Acme Analytics", result.Appearances[0].Name)
		require.True(t, result.Appearances[0].Confirmed)
		if result.ID == cell.ID {
			require.Equal(t, cell.RawAnswer, result.RawAnswer)
			require.WithinDuration(t, *cell.FetchedAt, *result.FetchedAt, time.Microsecond)
		}
	}

	otherOwner := auth.WithIdentity(context.Background(), auth.Identity{UserID: uuid.New(), OrganizationID: uuid.New()})
	_, err = commander.Update(otherOwner, suggestion.ID, domain.UpdateCompetitorParams{Name: &name})
	require.ErrorIs(t, err, domain.ErrNotFound)
	foreign, err := store.VisibilityRepo().ListResults(otherOwner, run.ID, domain.PageParams{Page: 1, PageSize: 10})
	require.ErrorIs(t, err, domain.ErrNotFound)
	require.Nil(t, foreign) // The scoped list contains no rows for another owner.
	_, err = commander.Create(ctx, brand.ID, domain.CreateCompetitorParams{
		Name: "Collision", Domain: "other.dev", AdditionalURLs: []string{"https://acme.dev"},
	})
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestObservedDismissalSurvivesRediscoveryAndAllowsManualIdentity(t *testing.T) {
	_, store, ctx, brand := liveVisibilityFixture(t, true)
	run := startVisibilityTracking(t, store, ctx, brand.ID)
	_, cells, err := store.BeginVisibilityAttempt(ctx, run.ID, *run.RiverJobID, 1)
	require.NoError(t, err)
	for _, cell := range cells[:2] {
		require.NoError(t, store.SaveVisibilityResult(ctx, run.ID, 1, appearanceCell(cell, "Acme", "https://acme.dev")))
	}
	competitors, err := store.CompetitorRepo().List(ctx, domain.CompetitorFilter{
		BrandFilter: domain.BrandFilter{BrandID: brand.ID},
	}, domain.PageParams{
		Page:     1,
		PageSize: 10,
	})
	require.NoError(t, err)
	commander := domain.NewCompetitorCommander(store)
	require.NoError(t, commander.Delete(ctx, competitors.Items[0].ID))
	tracking, err := store.VisibilityRepo().Tracking(ctx, brand.ID)
	require.NoError(t, err)
	require.True(t, tracking.Confirmed)
	require.Nil(t, tracking.BlockedReason)
	// A changed label on the same official URL remains dismissed.
	require.NoError(t, store.SaveVisibilityResult(ctx, run.ID, 1, appearanceCell(cells[2], "Acme Analytics", "https://acme.dev")))
	competitors, err = store.CompetitorRepo().List(ctx, domain.CompetitorFilter{
		BrandFilter: domain.BrandFilter{BrandID: brand.ID},
	}, domain.PageParams{
		Page:     1,
		PageSize: 10,
	})
	require.NoError(t, err)
	require.Empty(t, competitors.Items)
	manual, err := commander.Create(ctx, brand.ID, domain.CreateCompetitorParams{Name: "Curated Acme", Domain: "acme.dev"})
	require.NoError(t, err)
	require.NoError(t, store.SaveVisibilityResult(ctx, run.ID, 1, appearanceCell(cells[3], "Acme", "https://acme.dev")))
	read, err := store.VisibilityRepo().ListResults(ctx, run.ID, domain.PageParams{Page: 1, PageSize: 10})
	require.NoError(t, err)
	for _, cell := range read.Items {
		require.Equal(t, manual.ID, *cell.Appearances[0].CompetitorID)
		require.True(t, cell.Appearances[0].Confirmed)
	}
}

func TestDiscoveryIgnoresSelfStubAndFailedAnswers(t *testing.T) {
	db, store, ctx, brand := liveVisibilityFixture(t, true)
	run := startVisibilityTracking(t, store, ctx, brand.ID)
	_, cells, err := store.BeginVisibilityAttempt(ctx, run.ID, *run.RiverJobID, 1)
	require.NoError(t, err)
	for _, cell := range cells[:2] {
		require.NoError(t, store.SaveVisibilityResult(ctx, run.ID, 1, appearanceCell(cell, "Koi", "https://koi.dev")))
	}
	stub := appearanceCell(cells[2], "Unknown", "https://unknown.dev")
	stub.Collector = domain.CollectorStub
	require.NoError(t, db.Model(&stub).Update("collector", stub.Collector).Error)
	require.NoError(t, store.SaveVisibilityResult(ctx, run.ID, 1, stub))
	failed := appearanceCell(cells[3], "Unknown", "https://unknown.dev")
	errorOutcome := domain.OutcomeError
	code := "cloro_task_failed"
	failed.Outcome, failed.ErrorCode = &errorOutcome, &code
	require.NoError(t, store.SaveVisibilityResult(ctx, run.ID, 1, failed))
	var count int64
	require.NoError(t, db.Model(&domain.CompetitorObservation{}).Count(&count).Error)
	require.Zero(t, count)
	competitors, err := store.CompetitorRepo().List(ctx, domain.CompetitorFilter{
		BrandFilter: domain.BrandFilter{BrandID: brand.ID},
	}, domain.PageParams{
		Page:     1,
		PageSize: 10,
	})
	require.NoError(t, err)
	require.Empty(t, competitors.Items)
}

func TestMentionVerdictIsSavedAndConstrained(t *testing.T) {
	db, store, ctx, brand := liveVisibilityFixture(t, true)
	run := startVisibilityTracking(t, store, ctx, brand.ID)
	_, cells, err := store.BeginVisibilityAttempt(ctx, run.ID, *run.RiverJobID, 1)
	require.NoError(t, err)

	verdict := domain.MentionVerdictNoInfo
	cost := 0.0002
	cell := appearanceCell(cells[0], "Acme", "")
	cell.MentionVerdict = &verdict
	cell.MentionCostUSD = &cost
	require.NoError(t, store.SaveVisibilityResult(ctx, run.ID, 1, cell))

	// Unjudged answers keep a NULL verdict rather than an empty value.
	require.NoError(t, store.SaveVisibilityResult(ctx, run.ID, 1, appearanceCell(cells[1], "Acme", "")))

	read, err := store.VisibilityRepo().ListResults(ctx, run.ID, domain.PageParams{Page: 1, PageSize: 10})
	require.NoError(t, err)
	for _, result := range read.Items {
		switch result.ID {
		case cell.ID:
			require.Equal(t, &verdict, result.MentionVerdict)
			require.NotNil(t, result.MentionCostUSD)
			require.InDelta(t, cost, *result.MentionCostUSD, 1e-9)
		case cells[1].ID:
			require.Nil(t, result.MentionVerdict)
			require.Nil(t, result.MentionCostUSD)
		}
	}

	// A mismatch never reaches the database, and the database rejects unknown values itself.
	mismatch := appearanceCell(cells[2], "Acme", "")
	wrong := domain.MentionVerdictMention
	mismatch.MentionVerdict = &wrong
	require.ErrorIs(t, store.SaveVisibilityResult(ctx, run.ID, 1, mismatch), domain.ErrValidation)
	require.Error(t, db.Exec("UPDATE visibility_results SET mention_verdict = 'maybe' WHERE id = ?", cell.ID).Error)
}
