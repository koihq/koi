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
	client *river.Client[*sql.Tx]
}

func NewGormVisibilityRepository(db *gorm.DB, client *river.Client[*sql.Tx]) *GormVisibilityRepository {
	return &GormVisibilityRepository{
		Repository: newScopedRepository[koidomain.VisibilityRun](db, childOwnerScope("visibility_runs")),
		client:     client,
	}
}

func visibilityProgress(db *gorm.DB) *gorm.DB {
	return db.Select(`visibility_runs.*, (SELECT count(*) FROM visibility_results
		WHERE visibility_results.run_id = visibility_runs.id AND outcome IS NOT NULL) AS completed_results`)
}

func (r *GormVisibilityRepository) Get(ctx context.Context, brandID, runID uuid.UUID) (*koidomain.VisibilityRun, error) {
	return r.getBy(ctx, func(db *gorm.DB) *gorm.DB {
		return visibilityProgress(db).Where("brand_id = ? AND id = ?", brandID, runID)
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
	if err := prompts.Where("brand_id = ?", brandID).Order("created_at ASC, id ASC").Find(&inputs.Prompts).Error; err != nil {
		return inputs, mapDBError(err)
	}
	competitors, err := NewGormCompetitorRepository(r.db).query(ctx)
	if err != nil {
		return inputs, err
	}
	err = competitors.Model(&koidomain.Competitor{}).Where("brand_id = ? AND confirmed = false", brandID).Count(&inputs.UnconfirmedCompetitors).Error
	return inputs, mapDBError(err)
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

func (r *GormVisibilityRepository) ListResults(ctx context.Context, brandID, runID uuid.UUID, page koidomain.PageParams) (*koidomain.Page[koidomain.VisibilityResult], error) {
	results := newScopedRepository[koidomain.VisibilityResult](r.db, func(identity koiauth.Identity, db *gorm.DB) *gorm.DB {
		return db.Where(`EXISTS (SELECT 1 FROM visibility_runs JOIN brands ON brands.id = visibility_runs.brand_id
			WHERE visibility_runs.id = visibility_results.run_id AND visibility_runs.brand_id = ? AND brands.user_id = ?)`, brandID, identity.UserID)
	})
	return results.listBy(ctx, func(db *gorm.DB) *gorm.DB { return db.Where("run_id = ?", runID) }, page, "prompt_position ASC, engine ASC")
}

var _ koidomain.VisibilityRepository = (*GormVisibilityRepository)(nil)
