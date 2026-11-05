package database

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/domain"
)

func historyPage[T any](items []T, total int64, params domain.PageParams) *domain.Page[T] {
	page := &domain.Page[T]{
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

// PromptsWithResults reports which of a brand's prompts appear in at least one run.
func (r *GormVisibilityRepository) PromptsWithResults(ctx context.Context, brandID uuid.UUID, promptIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	if _, err := NewBrandRepository(r.db).Get(ctx, brandID); err != nil {
		return nil, err
	}

	var found []uuid.UUID
	err := r.db.WithContext(ctx).Model(&domain.VisibilityResult{}).
		Where("brand_id = ? AND prompt_id IN ?", brandID, promptIDs).
		Distinct().
		Pluck("prompt_id", &found).Error
	if err != nil {
		return nil, mapDBError(err)
	}

	withResults := make(map[uuid.UUID]bool, len(found))
	for _, id := range found {
		withResults[id] = true
	}

	return withResults, nil
}

func promptRunsQuery(db *gorm.DB, brandID, promptID uuid.UUID) *gorm.DB {
	return db.Where(`visibility_runs.brand_id = ? AND EXISTS (
		SELECT 1 FROM visibility_results AS result
		WHERE result.run_id = visibility_runs.id AND result.prompt_id = ?)`, brandID, promptID)
}

func (r *GormVisibilityRepository) ListPromptRuns(ctx context.Context, brandID, promptID uuid.UUID, params domain.PageParams) (*domain.Page[domain.PromptHistoryRun], error) {
	if _, err := NewBrandRepository(r.db).Get(ctx, brandID); err != nil {
		return nil, err
	}

	var total int64
	if err := promptRunsQuery(r.db.WithContext(ctx).Model(&domain.VisibilityRun{}), brandID, promptID).Count(&total).Error; err != nil {
		return nil, mapDBError(err)
	}
	if total == 0 {
		prompt, err := NewPromptRepository(r.db).Get(ctx, promptID)
		if err != nil {
			return nil, err
		}
		if prompt.BrandID != brandID {
			return nil, domain.ErrNotFound
		}
	}

	items := make([]domain.PromptHistoryRun, 0)
	if total == 0 || int64(params.Page) > (total-1)/int64(params.PageSize)+1 {
		return historyPage(items, total, params), nil
	}

	var runs []domain.VisibilityRun
	query := promptRunsQuery(visibilityProgress(r.db.WithContext(ctx).Model(&domain.VisibilityRun{})), brandID, promptID)
	if err := query.Order("visibility_runs.created_at DESC, visibility_runs.id DESC").Limit(params.PageSize).Offset((params.Page - 1) * params.PageSize).Find(&runs).Error; err != nil {
		return nil, mapDBError(err)
	}

	ids := make([]uuid.UUID, 0, len(runs))
	runByID := make(map[uuid.UUID]domain.VisibilityRun, len(runs))
	indexByID := make(map[uuid.UUID]int, len(runs))
	for i, run := range runs {
		ids = append(ids, run.ID)
		runByID[run.ID] = run
		indexByID[run.ID] = i
		items = append(items, domain.PromptHistoryRun{Run: run, Results: []domain.VisibilityResult{}})
	}

	var cells []domain.VisibilityResult
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
