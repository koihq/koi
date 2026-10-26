package koidatabase

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/koiauth"
	"github.com/koihq/koi/pkg/koidomain"
)

type VisibilityJobArgs struct {
	RunID uuid.UUID `json:"run_id"`
}

func (VisibilityJobArgs) Kind() string { return "koi_visibility" }

type GormVisibilityRepository struct {
	*Repository[koidomain.VisibilityRun]
	client    *river.Client[*sql.Tx]
	enabled   bool
	cloudMode bool
}

func NewGormVisibilityRepository(db *gorm.DB, client *river.Client[*sql.Tx], enabled bool, cloud ...bool) *GormVisibilityRepository {
	cloudMode := len(cloud) > 0 && cloud[0]
	return &GormVisibilityRepository{
		Repository: newScopedRepository[koidomain.VisibilityRun](db, childOwnerScope("visibility_runs")),
		client:     client,
		enabled:    enabled,
		cloudMode:  cloudMode,
	}
}

func visibilityProgress(db *gorm.DB) *gorm.DB {
	return db.Select(`visibility_runs.*, (SELECT count(*) FROM visibility_results
		WHERE visibility_results.run_id = visibility_runs.id AND outcome IS NOT NULL) AS completed_results`)
}

func (r *GormVisibilityRepository) Get(ctx context.Context, runID uuid.UUID) (*koidomain.VisibilityRun, error) {
	return r.getBy(ctx, func(db *gorm.DB) *gorm.DB {
		return visibilityProgress(db).Where("id = ?", runID)
	})
}

func (r *GormVisibilityRepository) List(ctx context.Context, brandID uuid.UUID, page koidomain.PageParams) (*koidomain.Page[koidomain.VisibilityRun], error) {
	return r.listBy(ctx, func(db *gorm.DB) *gorm.DB {
		return visibilityProgress(db).Where("brand_id = ?", brandID)
	}, page, "created_at DESC, id DESC")
}

func (r *GormVisibilityRepository) Active(ctx context.Context, brandID uuid.UUID) (*koidomain.VisibilityRun, error) {
	return r.getBy(ctx, func(db *gorm.DB) *gorm.DB {
		return visibilityProgress(db).Where("brand_id = ? AND state IN ?", brandID,
			[]koidomain.VisibilityState{koidomain.VisibilityQueued, koidomain.VisibilityRunning})
	})
}

// Inputs reads the entire curated set inside the brand-locked creation transaction.
func (r *GormVisibilityRepository) Inputs(ctx context.Context, brandID uuid.UUID) (koidomain.VisibilityInputs, error) {
	var inputs koidomain.VisibilityInputs
	prompts, err := NewGormPromptRepository(r.db).query(ctx)
	if err != nil {
		return inputs, err
	}

	if err := prompts.Where("brand_id = ? AND confirmed AND NOT inactive", brandID).
		Order("created_at ASC, id ASC").Find(&inputs.Prompts).Error; err != nil {
		return inputs, mapDBError(err)
	}

	return inputs, nil
}

func (r *GormVisibilityRepository) CreateAndEnqueue(ctx context.Context, run *koidomain.VisibilityRun, results []koidomain.VisibilityResult) error {
	if r.client == nil {
		return koidomain.ErrFeatureUnavailable
	}

	if _, err := NewGormBrandRepository(r.db).Get(ctx, run.BrandID); err != nil {
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

func (r *GormVisibilityRepository) ListResults(ctx context.Context, runID uuid.UUID, page koidomain.PageParams) (*koidomain.Page[koidomain.VisibilityResult], error) {
	run, err := r.Get(ctx, runID)
	if err != nil {
		return nil, err
	}
	brandID := run.BrandID
	results := newScopedRepository[koidomain.VisibilityResult](r.db, func(identity koiauth.Identity, db *gorm.DB) *gorm.DB {
		return db.Where(`EXISTS (SELECT 1 FROM visibility_runs JOIN brands ON brands.id = visibility_runs.brand_id
			WHERE visibility_runs.id = visibility_results.run_id AND visibility_runs.brand_id = ? AND brands.organization_id = ?)`, brandID, identity.OrganizationID)
	})

	response, err := results.listBy(ctx, func(db *gorm.DB) *gorm.DB { return db.Where("run_id = ?", runID) }, page, "prompt_position ASC, engine ASC")
	if err != nil {
		return nil, err
	}
	if len(response.Items) == 0 {
		return response, nil
	}

	if err := r.enrichResults(ctx, brandID, response.Items, map[uuid.UUID]koidomain.VisibilityRun{runID: *run}); err != nil {
		return nil, err
	}

	return response, nil
}

func (r *GormVisibilityRepository) enrichResults(ctx context.Context, brandID uuid.UUID, cells []koidomain.VisibilityResult, runs map[uuid.UUID]koidomain.VisibilityRun) error {
	brand, err := NewGormBrandRepository(r.db).Get(ctx, brandID)
	if err != nil {
		return err
	}
	query, err := NewGormCompetitorRepository(r.db).query(ctx)
	if err != nil {
		return err
	}
	var competitors []koidomain.Competitor
	if err := query.Where("brand_id = ?", brandID).Order("created_at, id").Find(&competitors).Error; err != nil {
		return mapDBError(err)
	}

	for i := range cells {
		cell := &cells[i]
		observations := cell.Observations[:0]
		for _, observation := range cell.Observations {
			if !koidomain.ObservationIsBrand(observation, brand.Snapshot()) {
				observations = append(observations, observation)
			}
		}
		cell.Observations = observations
		cell.Appearances = koidomain.MatchAppearances(*cell, competitors, runs[cell.RunID].BrandSnapshot)
	}

	return nil
}

var _ koidomain.VisibilityRepository = (*GormVisibilityRepository)(nil)
