package database

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/domain"
)

func TestPromptWithResultsCanOnlyBeStoppedAndKeepsItsHistory(t *testing.T) {
	_, store, ctx, brand := liveVisibilityFixture(t, true)
	querier := domain.NewPromptQuerier(store)
	commander := domain.NewPromptCommander(store)
	filter := domain.PromptFilter{BrandFilter: domain.BrandFilter{BrandID: brand.ID}}
	page := domain.PageParams{Page: 1, PageSize: 10}

	unseen, err := commander.Create(ctx, brand.ID, domain.CreatePromptParams{Text: "Never collected?"})
	require.NoError(t, err)
	require.False(t, unseen.HasResults)
	require.NoError(t, commander.Delete(ctx, unseen.ID))

	listed, err := querier.List(ctx, filter, page)
	require.NoError(t, err)
	tracked := listed.Items[0]
	require.False(t, tracked.HasResults, "a prompt no run has recorded can still be deleted")

	startVisibilityTracking(t, store, ctx, brand.ID)

	current, err := querier.Get(ctx, tracked.ID)
	require.NoError(t, err)
	require.True(t, current.HasResults, "results exist from the moment the run is created")
	require.ErrorIs(t, commander.Delete(ctx, tracked.ID), domain.ErrResourceConflict)

	_, err = store.PromptRepo().Get(ctx, tracked.ID)
	require.NoError(t, err, "a refused delete leaves the prompt in place")

	stopped := true
	updated, err := commander.Update(ctx, tracked.ID, domain.UpdatePromptParams{Inactive: &stopped})
	require.NoError(t, err)
	require.True(t, updated.Inactive)
	require.True(t, updated.HasResults)

	status, err := store.VisibilityRepo().Tracking(ctx, brand.ID)
	require.NoError(t, err)
	require.True(t, status.Confirmed, "stopping a prompt does not ask for a new confirmation")

	inputs, err := store.VisibilityRepo().Inputs(ctx, brand.ID)
	require.NoError(t, err)
	require.Empty(t, inputs.Prompts, "a stopped prompt leaves future runs")

	runs, err := domain.NewVisibilityQuerier(store).ListPromptRuns(ctx, brand.ID, tracked.ID, page)
	require.NoError(t, err)
	require.EqualValues(t, 1, runs.TotalItems, "a stopped prompt keeps its saved runs")

	resumed := false
	updated, err = commander.Update(ctx, tracked.ID, domain.UpdatePromptParams{Inactive: &resumed})
	require.NoError(t, err)
	require.False(t, updated.Inactive)

	status, err = store.VisibilityRepo().Tracking(ctx, brand.ID)
	require.NoError(t, err)
	require.True(t, status.Confirmed, "resuming a prompt does not ask for a new confirmation")

	inputs, err = store.VisibilityRepo().Inputs(ctx, brand.ID)
	require.NoError(t, err)
	require.Len(t, inputs.Prompts, 1)
	require.Equal(t, tracked.ID, inputs.Prompts[0].ID, "resuming continues the same prompt")
}

func TestPromptFiltersSeparateTrackingDraftsAndStopped(t *testing.T) {
	_, store, ctx, brand := liveVisibilityFixture(t, false)
	querier := domain.NewPromptQuerier(store)
	commander := domain.NewPromptCommander(store)
	page := domain.PageParams{Page: 1, PageSize: 10}

	draft := &domain.Prompt{BrandID: brand.ID, Text: "Drafted?", Source: domain.PromptSourceDrafted}
	require.NoError(t, store.PromptRepo().Create(ctx, draft))

	stoppedPrompt, err := commander.Create(ctx, brand.ID, domain.CreatePromptParams{Text: "Stopped?"})
	require.NoError(t, err)

	inactive := true
	_, err = commander.Update(ctx, stoppedPrompt.ID, domain.UpdatePromptParams{Inactive: &inactive})
	require.NoError(t, err)

	_, err = commander.Update(ctx, draft.ID, domain.UpdatePromptParams{Inactive: &inactive})
	require.ErrorIs(t, err, domain.ErrValidation)

	yes, no := true, false
	for name, test := range map[string]struct {
		confirmed *bool
		inactive  *bool
		want      string
	}{
		"tracking": {confirmed: &yes, inactive: &no, want: "Best analytics tools?"},
		"drafts":   {confirmed: &no, want: "Drafted?"},
		"stopped":  {inactive: &yes, want: "Stopped?"},
	} {
		t.Run(name, func(t *testing.T) {
			found, err := querier.List(ctx, domain.PromptFilter{
				BrandFilter: domain.BrandFilter{BrandID: brand.ID},
				Confirmed:   test.confirmed,
				Inactive:    test.inactive,
			}, page)
			require.NoError(t, err)
			require.EqualValues(t, 1, found.TotalItems)
			require.Equal(t, test.want, found.Items[0].Text)
		})
	}
}

func TestPromptsWithResultsIsScopedToTheBrandOwner(t *testing.T) {
	_, store, ctx, brand := liveVisibilityFixture(t, true)
	prompts, err := store.PromptRepo().List(ctx, domain.PromptFilter{
		BrandFilter: domain.BrandFilter{BrandID: brand.ID},
	}, domain.PageParams{Page: 1, PageSize: 10})
	require.NoError(t, err)
	prompt := prompts.Items[0]

	startVisibilityTracking(t, store, ctx, brand.ID)

	found, err := store.VisibilityRepo().PromptsWithResults(ctx, brand.ID, []uuid.UUID{prompt.ID})
	require.NoError(t, err)
	require.True(t, found[prompt.ID])

	_, otherStore, otherCtx, otherBrand := liveVisibilityFixture(t, false)
	found, err = otherStore.VisibilityRepo().PromptsWithResults(otherCtx, otherBrand.ID, []uuid.UUID{prompt.ID})
	require.NoError(t, err)
	require.False(t, found[prompt.ID], "another brand's results do not count")
}
