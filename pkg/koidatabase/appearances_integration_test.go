package koidatabase

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/koiauth"
	"github.com/koihq/koi/pkg/koidomain"
)

func appearanceCell(cell koidomain.VisibilityResult, name, link string) koidomain.VisibilityResult {
	now := time.Now().UTC()
	miss := koidomain.OutcomeMiss
	cell.Outcome, cell.FetchedAt = &miss, &now
	cell.RawAnswer = name + " is an option."
	cell.NamedEntities = []koidomain.NamedEntity{{Type: "software", Name: name}}
	if link != "" {
		cell.AnswerMarkdown = "1. [" + name + "](" + link + ")"
		cell.Sources = []string{link + "/docs", link + "/about"}
	}
	return cell
}

func TestObservedCompetitorsCountDistinctCellsAndNeverAutoConfirm(t *testing.T) {
	db, store, ctx, brand := liveVisibilityFixture(t, true)
	run := finishVisibilityReviews(t, store, ctx, brand.ID)
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
	var state koidomain.CompetitorObservation
	require.NoError(t, db.First(&state, "brand_id = ?", brand.ID).Error)
	require.EqualValues(t, 1, state.Count)
	competitors, err := store.CompetitorRepo().List(ctx, koidomain.CompetitorFilter{
		BrandID: brand.ID,
		Page: koidomain.PageParams{
			Page:     1,
			PageSize: 10,
		},
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
	competitors, err = store.CompetitorRepo().List(ctx, koidomain.CompetitorFilter{
		BrandID: brand.ID,
		Page: koidomain.PageParams{
			Page:     1,
			PageSize: 10,
		},
	})
	require.NoError(t, err)
	require.Len(t, competitors.Items, 1)
	suggestion := competitors.Items[0]
	require.Equal(t, koidomain.CompetitorSourceObserved, suggestion.Source)
	require.Equal(t, "acme.dev", suggestion.Domain)
	require.False(t, suggestion.Confirmed)
	require.NoError(t, store.VisibilityRepo().FinishReview(ctx, brand.ID, "competitors"))
	unchanged, err := store.CompetitorRepo().Get(ctx, suggestion.ID)
	require.NoError(t, err)
	require.False(t, unchanged.Confirmed)
	tracking, err := store.VisibilityRepo().Tracking(ctx, brand.ID)
	require.NoError(t, err)
	require.True(t, tracking.CompetitorsReviewed)
	require.Nil(t, tracking.BlockedReason)

	read, err := store.VisibilityRepo().ListResults(ctx, run.ID, koidomain.PageParams{Page: 1, PageSize: 10})
	require.NoError(t, err)
	for _, cell := range read.Items {
		if cell.Outcome != nil {
			require.Equal(t, koidomain.OutcomeMiss, *cell.Outcome)
			require.Len(t, cell.Appearances, 1)
			require.Equal(t, suggestion.ID, *cell.Appearances[0].CompetitorID)
			require.False(t, cell.Appearances[0].Confirmed)
			require.Contains(t, cell.Appearances[0].Evidence, koidomain.AppearanceEvidence{Kind: "citation", Value: "https://acme.dev/docs"})
			require.Contains(t, cell.Appearances[0].Evidence, koidomain.AppearanceEvidence{Kind: "citation", Value: "https://acme.dev/about"})
		} else {
			require.NotNil(t, cell.Appearances)
		}
	}
}

func TestNameOnlySuggestionNeedsIdentityAndHistoricalLinksFollowEdits(t *testing.T) {
	_, store, ctx, brand := liveVisibilityFixture(t, true)
	run := finishVisibilityReviews(t, store, ctx, brand.ID)
	_, cells, err := store.BeginVisibilityAttempt(ctx, run.ID, *run.RiverJobID, 1)
	require.NoError(t, err)
	for _, cell := range cells[:2] {
		require.NoError(t, store.SaveVisibilityResult(ctx, run.ID, 1, appearanceCell(cell, "Acme", "")))
	}
	competitors, err := store.CompetitorRepo().List(ctx, koidomain.CompetitorFilter{
		BrandID: brand.ID,
		Page: koidomain.PageParams{
			Page:     1,
			PageSize: 10,
		},
	})
	require.NoError(t, err)
	require.Len(t, competitors.Items, 1)
	suggestion := competitors.Items[0]
	commander := koidomain.NewCompetitorCommander(store)
	_, err = commander.Confirm(ctx, suggestion.ID)
	require.ErrorIs(t, err, koidomain.ErrValidation)

	// This plain text has no discovery hints; an alias added later attaches it on read.
	cell := appearanceCell(cells[2], "Alternate", "")
	cell.NamedEntities = nil
	require.NoError(t, store.SaveVisibilityResult(ctx, run.ID, 1, cell))
	name, domain := "Acme Analytics", "acme.dev"
	aliases := []string{"Acme", "Alternate"}
	updated, err := commander.Patch(ctx, suggestion.ID, koidomain.CompetitorUpdate{Name: &name, Domain: &domain, Aliases: &aliases})
	require.NoError(t, err)
	require.False(t, updated.Confirmed)
	_, err = commander.Confirm(ctx, suggestion.ID)
	require.NoError(t, err)
	read, err := store.VisibilityRepo().ListResults(ctx, run.ID, koidomain.PageParams{Page: 1, PageSize: 10})
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

	otherOwner := koiauth.WithIdentity(context.Background(), koiauth.Identity{UserID: uuid.New(), OrganizationID: uuid.New()})
	_, err = commander.Patch(otherOwner, suggestion.ID, koidomain.CompetitorUpdate{Name: &name})
	require.ErrorIs(t, err, koidomain.ErrNotFound)
	foreign, err := store.VisibilityRepo().ListResults(otherOwner, run.ID, koidomain.PageParams{Page: 1, PageSize: 10})
	require.ErrorIs(t, err, koidomain.ErrNotFound)
	require.Nil(t, foreign) // The scoped list contains no rows for another owner.
	_, err = commander.Create(ctx, brand.ID, koidomain.CompetitorParams{
		Name: "Collision", Domain: "other.dev", AdditionalURLs: []string{"https://acme.dev"},
	})
	require.ErrorIs(t, err, koidomain.ErrValidation)
}

func TestObservedDismissalSurvivesRediscoveryAndAllowsManualIdentity(t *testing.T) {
	_, store, ctx, brand := liveVisibilityFixture(t, true)
	run := finishVisibilityReviews(t, store, ctx, brand.ID)
	_, cells, err := store.BeginVisibilityAttempt(ctx, run.ID, *run.RiverJobID, 1)
	require.NoError(t, err)
	for _, cell := range cells[:2] {
		require.NoError(t, store.SaveVisibilityResult(ctx, run.ID, 1, appearanceCell(cell, "Acme", "https://acme.dev")))
	}
	competitors, err := store.CompetitorRepo().List(ctx, koidomain.CompetitorFilter{
		BrandID: brand.ID,
		Page: koidomain.PageParams{
			Page:     1,
			PageSize: 10,
		},
	})
	require.NoError(t, err)
	commander := koidomain.NewCompetitorCommander(store)
	require.NoError(t, commander.Delete(ctx, competitors.Items[0].ID))
	tracking, err := store.VisibilityRepo().Tracking(ctx, brand.ID)
	require.NoError(t, err)
	require.True(t, tracking.CompetitorsReviewed)
	require.Nil(t, tracking.BlockedReason)
	// A changed label on the same official URL remains dismissed.
	require.NoError(t, store.SaveVisibilityResult(ctx, run.ID, 1, appearanceCell(cells[2], "Acme Analytics", "https://acme.dev")))
	competitors, err = store.CompetitorRepo().List(ctx, koidomain.CompetitorFilter{
		BrandID: brand.ID,
		Page: koidomain.PageParams{
			Page:     1,
			PageSize: 10,
		},
	})
	require.NoError(t, err)
	require.Empty(t, competitors.Items)
	manual, err := commander.Create(ctx, brand.ID, koidomain.CompetitorParams{Name: "Curated Acme", Domain: "acme.dev"})
	require.NoError(t, err)
	require.NoError(t, store.SaveVisibilityResult(ctx, run.ID, 1, appearanceCell(cells[3], "Acme", "https://acme.dev")))
	read, err := store.VisibilityRepo().ListResults(ctx, run.ID, koidomain.PageParams{Page: 1, PageSize: 10})
	require.NoError(t, err)
	for _, cell := range read.Items {
		require.Equal(t, manual.ID, *cell.Appearances[0].CompetitorID)
		require.True(t, cell.Appearances[0].Confirmed)
	}
}

func TestDiscoveryIgnoresSelfStubAndFailedAnswers(t *testing.T) {
	db, store, ctx, brand := liveVisibilityFixture(t, true)
	run := finishVisibilityReviews(t, store, ctx, brand.ID)
	_, cells, err := store.BeginVisibilityAttempt(ctx, run.ID, *run.RiverJobID, 1)
	require.NoError(t, err)
	for _, cell := range cells[:2] {
		require.NoError(t, store.SaveVisibilityResult(ctx, run.ID, 1, appearanceCell(cell, "Koi", "https://koi.dev")))
	}
	stub := appearanceCell(cells[2], "Unknown", "https://unknown.dev")
	stub.Collector = koidomain.CollectorStub
	require.NoError(t, db.Model(&stub).Update("collector", stub.Collector).Error)
	require.NoError(t, store.SaveVisibilityResult(ctx, run.ID, 1, stub))
	failed := appearanceCell(cells[3], "Unknown", "https://unknown.dev")
	errorOutcome := koidomain.OutcomeError
	code := "cloro_task_failed"
	failed.Outcome, failed.ErrorCode = &errorOutcome, &code
	require.NoError(t, store.SaveVisibilityResult(ctx, run.ID, 1, failed))
	var count int64
	require.NoError(t, db.Model(&koidomain.CompetitorObservation{}).Count(&count).Error)
	require.Zero(t, count)
	competitors, err := store.CompetitorRepo().List(ctx, koidomain.CompetitorFilter{
		BrandID: brand.ID,
		Page: koidomain.PageParams{
			Page:     1,
			PageSize: 10,
		},
	})
	require.NoError(t, err)
	require.Empty(t, competitors.Items)
}
