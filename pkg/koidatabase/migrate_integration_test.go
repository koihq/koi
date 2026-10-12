package koidatabase

import (
	"testing"

	"github.com/koihq/koi/pkg/koidomain"
	"github.com/stretchr/testify/require"
)

func TestEntityIndexesSurviveRepeatedMigration(t *testing.T) {
	db, _, _, brand := liveVisibilityFixture(t, true)
	type index struct {
		ID         int64
		Definition string
	}
	read := func() map[string]index {
		t.Helper()
		var rows []struct {
			Name       string
			ID         int64
			Definition string
		}
		require.NoError(t, db.Raw(`SELECT indexname AS name,
			(format('%I.%I', schemaname, indexname)::regclass)::oid::bigint AS id,
			indexdef AS definition FROM pg_indexes WHERE schemaname = current_schema()`).Scan(&rows).Error)
		indexes := make(map[string]index, len(rows))
		for _, row := range rows {
			indexes[row.Name] = index{ID: row.ID, Definition: row.Definition}
		}
		return indexes
	}

	before := read()
	for name, columns := range map[string]string{
		"idx_visibility_runs_active":        "UNIQUE INDEX",
		"idx_visibility_runs_slot":          "(brand_id, scheduled_for)",
		"idx_visibility_runs_recent":        "(brand_id, created_at DESC, id DESC)",
		"idx_visibility_results_cell":       "(run_id, prompt_id, engine)",
		"idx_competitors_brand_domain":      "(brand_id, domain)",
		"idx_competitors_brand_github_repo": "(brand_id, github_repo)",
		"idx_prompts_brand_text":            "(brand_id, lower(text))",
		"idx_generation_runs_recent":        "(brand_id, created_at DESC, id DESC)",
		"idx_generation_runs_key":           "(brand_id, idempotency_key)",
		"idx_generation_runs_active":        "UNIQUE INDEX",
		"idx_visibility_results_order":      "(run_id, prompt_position, engine)",
	} {
		require.Contains(t, before, name)
		require.Contains(t, before[name].Definition, columns)
	}
	require.Contains(t, before["idx_competitors_brand_domain"].Definition, "WHERE")
	require.Contains(t, before["idx_competitors_brand_github_repo"].Definition, "WHERE")
	require.Contains(t, before["idx_visibility_runs_slot"].Definition, "scheduled_for IS NOT NULL")
	require.Contains(t, before["idx_generation_runs_key"].Definition, "idempotency_key IS NOT NULL")
	require.NoError(t, Migrate(db))
	require.Equal(t, before, read())

	duplicate := &koidomain.Prompt{
		BrandID:   brand.ID,
		Text:      "BEST ANALYTICS TOOLS?",
		Source:    koidomain.PromptSourceManual,
		Confirmed: true,
	}
	requireConstraint(t, db.Create(duplicate).Error, "idx_prompts_brand_text")
}
