package database

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/auth"
	"github.com/koihq/koi/pkg/domain"
)

// Results have no foreign key to prompts: they stay readable if the row ever disappears.
func TestPromptHistoryKeepsSnapshotsWithoutLivePrompt(t *testing.T) {
	_, store, ctx, brand := liveVisibilityFixture(t, true)
	prompts, err := store.PromptRepo().List(ctx, domain.PromptFilter{
		BrandFilter: domain.BrandFilter{BrandID: brand.ID},
	}, domain.PageParams{Page: 1, PageSize: 10})
	require.NoError(t, err)
	prompt := prompts.Items[0]
	run := finishVisibilityReviews(t, store, ctx, brand.ID)
	querier := domain.NewVisibilityQuerier(store)
	page := domain.PageParams{Page: 1, PageSize: 10}

	live, err := querier.ListPromptRuns(ctx, brand.ID, prompt.ID, page)
	require.NoError(t, err)
	require.EqualValues(t, 1, live.TotalItems)
	require.Len(t, live.Items[0].Results, 4)
	require.Equal(t, run.ID, live.Items[0].Run.ID)

	require.NoError(t, store.PromptRepo().Delete(ctx, prompt.ID))

	saved, err := querier.ListPromptRuns(ctx, brand.ID, prompt.ID, page)
	require.NoError(t, err)
	require.EqualValues(t, 1, saved.TotalItems)
	require.Len(t, saved.Items[0].Results, 4)
	require.Equal(t, prompt.Text, saved.Items[0].Results[0].PromptText)

	empty, err := querier.ListPromptRuns(ctx, brand.ID, prompt.ID, domain.PageParams{
		Page:     2,
		PageSize: 10,
	})
	require.NoError(t, err)
	require.Empty(t, empty.Items)
	require.True(t, empty.HasPrev)

	stranger := auth.WithIdentity(t.Context(), auth.Identity{UserID: uuid.New(), OrganizationID: uuid.New()})
	_, err = querier.ListPromptRuns(stranger, brand.ID, prompt.ID, page)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestHistoricalPromptLookupRequiresItsExplicitBrand(t *testing.T) {
	_, store, ctx, brand := liveVisibilityFixture(t, false)
	other := &domain.Brand{OrganizationID: brand.OrganizationID, Name: "Other brand", Domain: "other.dev", Country: "IT"}
	require.NoError(t, store.BrandRepo().Create(ctx, other))
	prompt, err := domain.NewPromptCommander(store).Create(ctx, other.ID, domain.CreatePromptParams{Text: "Other brand question"})
	require.NoError(t, err)
	_, err = store.VisibilityRepo().ListPromptRuns(ctx, brand.ID, prompt.ID, domain.PageParams{Page: 1, PageSize: 10})
	require.ErrorIs(t, err, domain.ErrNotFound)
}
