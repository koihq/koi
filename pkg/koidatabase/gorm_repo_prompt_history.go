package koidatabase

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/koidomain"
)

func historyPage[T any](items []T, total int64, params koidomain.PageParams) *koidomain.Page[T] {
	page := &koidomain.Page[T]{
		Items:       items,
		TotalItems:  total,
		CurrentPage: params.Page,
		HasPrev:     params.Page > 1,
	}
	if total > 0 {
		page.TotalPages = (total-1)/int64(params.PageSize) + 1
	}
	page.HasNext = int64(params.Page) < page.TotalPages

	return page
}

func (r *GormVisibilityRepository) ListHistoricalPrompts(ctx context.Context, brandID uuid.UUID, params koidomain.PageParams) (*koidomain.Page[koidomain.HistoricalPrompt], error) {
	if _, err := NewGormBrandRepository(r.db).Get(ctx, brandID); err != nil {
		return nil, err
	}

	const from = `FROM visibility_results AS result
		JOIN visibility_runs AS run ON run.id = result.run_id
		LEFT JOIN prompts AS prompt ON prompt.id = result.prompt_id AND prompt.brand_id = run.brand_id
		WHERE run.brand_id = ? AND prompt.id IS NULL`

	var total int64
	if err := r.db.WithContext(ctx).Raw(`SELECT count(DISTINCT result.prompt_id) `+from, brandID).Scan(&total).Error; err != nil {
		return nil, mapDBError(err)
	}

	items := make([]koidomain.HistoricalPrompt, 0)
	if total > 0 {
		const snapshots = `SELECT DISTINCT ON (result.prompt_id)
			result.prompt_id, result.prompt_text, run.created_at AS last_seen_at ` + from + `
			ORDER BY result.prompt_id, run.created_at DESC, run.id DESC`
		query := `SELECT * FROM (` + snapshots + `) AS snapshot
			ORDER BY last_seen_at DESC, prompt_id DESC LIMIT ? OFFSET ?`
		if err := r.db.WithContext(ctx).Raw(query, brandID, params.PageSize, (params.Page-1)*params.PageSize).Scan(&items).Error; err != nil {
			return nil, mapDBError(err)
		}
	}

	return historyPage(items, total, params), nil
}

func promptRunsQuery(db *gorm.DB, brandID, promptID uuid.UUID) *gorm.DB {
	return db.Where(`visibility_runs.brand_id = ? AND EXISTS (
		SELECT 1 FROM visibility_results AS result
		WHERE result.run_id = visibility_runs.id AND result.prompt_id = ?)`, brandID, promptID)
}

func (r *GormVisibilityRepository) ListPromptRuns(ctx context.Context, brandID, promptID uuid.UUID, params koidomain.PageParams) (*koidomain.Page[koidomain.PromptHistoryRun], error) {
	if _, err := NewGormBrandRepository(r.db).Get(ctx, brandID); err != nil {
		return nil, err
	}

	var total int64
	if err := promptRunsQuery(r.db.WithContext(ctx).Model(&koidomain.VisibilityRun{}), brandID, promptID).Count(&total).Error; err != nil {
		return nil, mapDBError(err)
	}
	if total == 0 {
		if _, err := NewGormPromptRepository(r.db).Get(ctx, koidomain.PromptFilter{ID: promptID, BrandID: brandID}); err != nil {
			return nil, err
		}
	}

	items := make([]koidomain.PromptHistoryRun, 0)
	if total == 0 || int64(params.Page) > (total-1)/int64(params.PageSize)+1 {
		return historyPage(items, total, params), nil
	}

	var runs []koidomain.VisibilityRun
	query := promptRunsQuery(visibilityProgress(r.db.WithContext(ctx).Model(&koidomain.VisibilityRun{})), brandID, promptID)
	if err := query.Order("visibility_runs.created_at DESC, visibility_runs.id DESC").Limit(params.PageSize).Offset((params.Page - 1) * params.PageSize).Find(&runs).Error; err != nil {
		return nil, mapDBError(err)
	}

	ids := make([]uuid.UUID, 0, len(runs))
	runByID := make(map[uuid.UUID]koidomain.VisibilityRun, len(runs))
	indexByID := make(map[uuid.UUID]int, len(runs))
	for i, run := range runs {
		ids = append(ids, run.ID)
		runByID[run.ID] = run
		indexByID[run.ID] = i
		items = append(items, koidomain.PromptHistoryRun{Run: run, Results: []koidomain.VisibilityResult{}})
	}

	var cells []koidomain.VisibilityResult
	if err := r.db.WithContext(ctx).Where("run_id IN ? AND prompt_id = ?", ids, promptID).Order("run_id, engine").Find(&cells).Error; err != nil {
		return nil, mapDBError(err)
	}
	if err := r.enrichResults(ctx, brandID, cells, runByID); err != nil {
		return nil, err
	}
	for _, cell := range cells {
		items[indexByID[cell.RunID]].Results = append(items[indexByID[cell.RunID]].Results, cell)
	}

	return historyPage(items, total, params), nil
}
