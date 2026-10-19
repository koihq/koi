package koidatabase

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/koiauth"
	"github.com/koihq/koi/pkg/koidomain"
)

func TestPromptHistoryKeepsDeletedPromptSnapshots(t *testing.T) {
	_, store, ctx, brand := liveVisibilityFixture(t, true)
	prompts, err := store.PromptRepo().List(ctx, koidomain.PromptFilter{
		BrandID: brand.ID,
		Page:    koidomain.PageParams{Page: 1, PageSize: 10},
	})
	require.NoError(t, err)
	prompt := prompts.Items[0]
	run := finishVisibilityReviews(t, store, ctx, brand.ID)
	querier := koidomain.NewVisibilityQuerier(store)
	page := koidomain.PageParams{Page: 1, PageSize: 10}

	live, err := querier.ListPromptRuns(ctx, brand.ID, prompt.ID, page)
	require.NoError(t, err)
	require.EqualValues(t, 1, live.TotalItems)
	require.Len(t, live.Items[0].Results, 4)
	require.Equal(t, run.ID, live.Items[0].Run.ID)

	require.NoError(t, koidomain.NewPromptCommander(store).Delete(ctx, brand.ID, prompt.ID))
	historical, err := querier.ListHistoricalPrompts(ctx, brand.ID, koidomain.PromptListParams{Page: page})
	require.NoError(t, err)
	require.EqualValues(t, 1, historical.TotalItems)
	require.Equal(t, prompt.ID, historical.Items[0].PromptID)
	require.Equal(t, prompt.Text, historical.Items[0].PromptText)

	saved, err := querier.ListPromptRuns(ctx, brand.ID, prompt.ID, page)
	require.NoError(t, err)
	require.EqualValues(t, 1, saved.TotalItems)
	require.Len(t, saved.Items[0].Results, 4)
	require.Equal(t, prompt.Text, saved.Items[0].Results[0].PromptText)

	empty, err := querier.ListPromptRuns(ctx, brand.ID, prompt.ID, koidomain.PageParams{
		Page:     2,
		PageSize: 10,
	})
	require.NoError(t, err)
	require.Empty(t, empty.Items)
	require.True(t, empty.HasPrev)

	stranger := koiauth.WithIdentity(t.Context(), koiauth.Identity{UserID: uuid.New()})
	_, err = querier.ListHistoricalPrompts(stranger, brand.ID, koidomain.PromptListParams{Page: page})
	require.ErrorIs(t, err, koidomain.ErrNotFound)
	_, err = querier.ListPromptRuns(stranger, brand.ID, prompt.ID, page)
	require.ErrorIs(t, err, koidomain.ErrNotFound)
}
