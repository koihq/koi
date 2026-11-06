package database

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/auth"
	"github.com/koihq/koi/pkg/domain"
)

type VisibilityJobArgs struct {
	RunID uuid.UUID `json:"run_id"`
}

func (VisibilityJobArgs) Kind() string { return "koi_visibility" }

type GormVisibilityRepository struct {
	*ListRepository[domain.VisibilityRun, domain.BrandFilter]
	client              *river.Client[*sql.Tx]
	enabled             bool
	cloudMode           bool
	firstLookDailyLimit int
}

func NewVisibilityRepository(db *gorm.DB, client *river.Client[*sql.Tx], enabled, cloudMode bool) *GormVisibilityRepository {
	base := newBrandChildRepository[domain.VisibilityRun](db)
	base.read = visibilityProgress

	return &GormVisibilityRepository{
		ListRepository:      newListRepository(base, filterByBrand, fixedOrder[domain.BrandFilter]("created_at DESC, id DESC")),
		client:              client,
		enabled:             enabled,
		cloudMode:           cloudMode,
		firstLookDailyLimit: domain.DefaultFirstLookDailyLimit,
	}
}

func visibilityProgress(db *gorm.DB) *gorm.DB {
	return db.Select(`visibility_runs.*, (SELECT count(*) FROM visibility_results
		WHERE visibility_results.run_id = visibility_runs.id AND outcome IS NOT NULL) AS completed_results`)
}

func (r *GormVisibilityRepository) Active(ctx context.Context, brandID uuid.UUID) (*domain.VisibilityRun, error) {
	return r.getBy(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("brand_id = ? AND state IN ?", brandID,
			[]domain.VisibilityState{domain.VisibilityQueued, domain.VisibilityRunning})
	})
}

// Inputs reads the entire curated set inside the brand-locked creation transaction.
func (r *GormVisibilityRepository) Inputs(ctx context.Context, brandID uuid.UUID) (domain.VisibilityInputs, error) {
	var inputs domain.VisibilityInputs
	prompts, err := NewPromptRepository(r.db).query(ctx)
	if err != nil {
		return inputs, err
	}

	if err := prompts.Where("brand_id = ? AND confirmed AND NOT inactive", brandID).
		Order("created_at ASC, id ASC").Find(&inputs.Prompts).Error; err != nil {
		return inputs, mapDBError(err)
	}

	return inputs, nil
}

func (r *GormVisibilityRepository) CreateAndEnqueue(ctx context.Context, run *domain.VisibilityRun, results []domain.VisibilityResult) error {
	if r.client == nil {
		return domain.ErrFeatureUnavailable
	}

	if _, err := NewBrandRepository(r.db).Get(ctx, run.BrandID); err != nil {
		return err
	}

	tx := r.db.WithContext(ctx)
	sqlTx, ok := tx.Statement.ConnPool.(*sql.Tx)
	if !ok {
		return errors.New("visibility enqueue requires a SQL transaction")
	}

	if err := run.Validate(); err != nil {
		return err
	}

	if int64(len(results)) != run.TotalResults {
		return errors.New("visibility matrix size mismatch")
	}

	if err := tx.Create(run).Error; err != nil {
		return mapDBError(err)
	}

	for i := range results {
		results[i].RunID = run.ID
		results[i].BrandID = run.BrandID
		results[i].RunCreatedAt = run.CreatedAt
		if err := results[i].Validate(); err != nil {
			return err
		}
	}

	if err := tx.CreateInBatches(&results, 100).Error; err != nil {
		return mapDBError(err)
	}

	inserted, err := r.client.InsertTx(ctx, sqlTx, VisibilityJobArgs{RunID: run.ID}, &river.InsertOpts{Queue: "visibility", MaxAttempts: 3})
	if err != nil {
		return err
	}

	run.RiverJobID = &inserted.Job.ID

	return mapDBError(tx.Model(run).Update("river_job_id", inserted.Job.ID).Error)
}

func (r *GormVisibilityRepository) ListResults(ctx context.Context, runID uuid.UUID, page domain.PageParams) (*domain.Page[domain.VisibilityResult], error) {
	run, err := r.Get(ctx, runID)
	if err != nil {
		return nil, err
	}
	brandID := run.BrandID
	results := newScopedRepository(r.db, tenantScope[domain.VisibilityResult]{
		read: func(identity auth.Identity, db *gorm.DB) *gorm.DB {
			return db.Where(`EXISTS (SELECT 1 FROM visibility_runs JOIN brands ON brands.id = visibility_runs.brand_id
				WHERE visibility_runs.id = visibility_results.run_id AND visibility_runs.brand_id = ? AND brands.organization_id = ?)`, brandID, identity.OrganizationID)
		},
	})

	query, err := results.query(ctx)
	if err != nil {
		return nil, err
	}

	response, err := paginate[domain.VisibilityResult](query.Where("run_id = ?", runID), page, "prompt_position ASC, engine ASC")
	if err != nil {
		return nil, err
	}
	if len(response.Items) == 0 {
		return response, nil
	}

	if err := r.enrichResults(ctx, brandID, response.Items, map[uuid.UUID]domain.VisibilityRun{runID: *run}); err != nil {
		return nil, err
	}

	return response, nil
}

func (r *GormVisibilityRepository) enrichResults(ctx context.Context, brandID uuid.UUID, cells []domain.VisibilityResult, runs map[uuid.UUID]domain.VisibilityRun) error {
	brand, err := NewBrandRepository(r.db).Get(ctx, brandID)
	if err != nil {
		return err
	}
	query, err := NewCompetitorRepository(r.db).query(ctx)
	if err != nil {
		return err
	}
	var competitors []domain.Competitor
	if err := query.Where("brand_id = ?", brandID).Order("created_at, id").Find(&competitors).Error; err != nil {
		return mapDBError(err)
	}

	matcher := domain.NewAppearanceMatcher(competitors)
	for i := range cells {
		cell := &cells[i]
		observations := cell.Observations[:0]
		for _, observation := range cell.Observations {
			if !domain.ObservationIsBrand(observation, brand.Snapshot()) {
				observations = append(observations, observation)
			}
		}
		cell.Observations = observations
		cell.Appearances = matcher.Match(*cell, runs[cell.RunID].BrandSnapshot)
	}

	return nil
}

var _ domain.VisibilityRepository = (*GormVisibilityRepository)(nil)
