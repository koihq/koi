package koidatabase

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/koidomain"
)

func TestPromptSortPaginatesAfterRanking(t *testing.T) {
	db, store, ctx, brand := liveVisibilityFixture(t, true)
	commander := koidomain.NewPromptCommander(store)
	prompts, err := store.PromptRepo().List(ctx, koidomain.PromptFilter{
		BrandID: brand.ID,
		Page:    koidomain.PageParams{Page: 1, PageSize: 10},
	})
	require.NoError(t, err)
	ids := []uuid.UUID{prompts.Items[0].ID}
	for _, name := range []string{"B", "C", "D", "E"} {
		prompt, err := commander.Create(ctx, brand.ID, koidomain.PromptParams{Text: name})
		require.NoError(t, err)
		ids = append(ids, prompt.ID)
	}

	run := finishVisibilityReviews(t, store, ctx, brand.ID)
	var cells []koidomain.VisibilityResult
	require.NoError(t, db.Where("run_id = ?", run.ID).Find(&cells).Error)
	positiveByPrompt := map[uuid.UUID]int{
		ids[0]: 0,
		ids[1]: 1,
		ids[2]: 4,
		ids[3]: 2,
		ids[4]: 3,
	}
	seen := make(map[uuid.UUID]int)
	now := time.Now().UTC()
	for _, cell := range cells {
		outcome := koidomain.OutcomeMiss
		if seen[cell.PromptID] < positiveByPrompt[cell.PromptID] {
			outcome = koidomain.OutcomeCited
		}
		seen[cell.PromptID]++
		require.NoError(t, db.Model(&cell).Updates(map[string]any{
			"outcome":    outcome,
			"fetched_at": now,
		}).Error)
	}

	page := koidomain.PageParams{Page: 1, PageSize: 2}
	filter := koidomain.PromptFilter{
		BrandID:    brand.ID,
		Page:       page,
		Sort:       koidomain.PromptSortRate,
		Descending: true,
		Days:       30,
	}
	first, err := store.PromptRepo().List(ctx, filter)
	require.NoError(t, err)
	require.EqualValues(t, 5, first.TotalItems)
	require.Equal(t, []uuid.UUID{ids[2], ids[4]}, []uuid.UUID{first.Items[0].ID, first.Items[1].ID})
	require.True(t, first.HasNext)

	filter.Page.Page = 2
	second, err := store.PromptRepo().List(ctx, filter)
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{ids[3], ids[1]}, []uuid.UUID{second.Items[0].ID, second.Items[1].ID})

	for _, id := range ids {
		require.NoError(t, commander.Delete(ctx, brand.ID, id))
	}
	historyParams := koidomain.PromptListParams{
		Page:       page,
		Sort:       koidomain.PromptSortRate,
		Descending: true,
		Days:       30,
	}
	history, err := store.VisibilityRepo().ListHistoricalPrompts(ctx, brand.ID, historyParams)
	require.NoError(t, err)
	require.EqualValues(t, 5, history.TotalItems)
	require.Equal(t, []uuid.UUID{ids[2], ids[4]}, []uuid.UUID{history.Items[0].PromptID, history.Items[1].PromptID})
}
