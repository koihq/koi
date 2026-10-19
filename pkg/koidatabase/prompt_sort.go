package koidatabase

import (
	"time"

	"github.com/google/uuid"

	"github.com/koihq/koi/pkg/koidomain"
)

func visibilitySortJoin(promptID string, brandID uuid.UUID, days int, engines []koidomain.EngineID) (string, []any) {
	if days == 0 {
		days = 30
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)
	start := today.AddDate(0, 0, 1-days)
	previousStart := start.AddDate(0, 0, -days)
	end := today.AddDate(0, 0, 1)

	query := `LEFT JOIN LATERAL (
		SELECT
			100.0 * count(*) FILTER (WHERE score_run.created_at >= ? AND score_result.outcome IN ('cited', 'mentioned')) /
				NULLIF(count(*) FILTER (WHERE score_run.created_at >= ?), 0) AS current_rate,
			100.0 * count(*) FILTER (WHERE score_run.created_at < ? AND score_result.outcome IN ('cited', 'mentioned')) /
				NULLIF(count(*) FILTER (WHERE score_run.created_at < ?), 0) AS previous_rate
		FROM visibility_results AS score_result
		JOIN visibility_runs AS score_run ON score_run.id = score_result.run_id
		WHERE score_result.prompt_id = ` + promptID + `
			AND score_run.brand_id = ?
			AND score_run.created_at >= ? AND score_run.created_at < ?
			AND score_result.outcome IN ('cited', 'mentioned', 'miss')`
	args := []any{start, start, start, start, brandID, previousStart, end}
	if len(engines) > 0 {
		query += ` AND score_result.engine IN ?`
		args = append(args, engines)
	}

	return query + `) AS visibility_sort ON true`, args
}

func promptSortOrder(sort koidomain.PromptSort, descending bool, text, original string) string {
	direction := "ASC"
	if descending {
		direction = "DESC"
	}

	switch sort {
	case koidomain.PromptSortText:
		return "lower(" + text + ") " + direction + ", " + original
	case koidomain.PromptSortRate:
		return "visibility_sort.current_rate " + direction + " NULLS LAST, " + original
	case koidomain.PromptSortChange:
		return "(visibility_sort.current_rate - visibility_sort.previous_rate) " + direction + " NULLS LAST, " + original
	default:
		return original
	}
}
