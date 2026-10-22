package koidatabase

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/koidomain"
)

type GenerationJobArgs struct {
	RunID uuid.UUID `json:"run_id"`
}

func (GenerationJobArgs) Kind() string { return "koi_generation" }

type GormGenerationRepository struct {
	*Repository[koidomain.GenerationRun]
	client    *river.Client[*sql.Tx]
	cloudMode bool
}

func NewGormGenerationRepository(db *gorm.DB, client *river.Client[*sql.Tx], cloud ...bool) *GormGenerationRepository {
	return &GormGenerationRepository{
		Repository: newScopedRepository[koidomain.GenerationRun](db, childOwnerScope("generation_runs")),
		client:     client,
		cloudMode:  len(cloud) > 0 && cloud[0],
	}
}

func (r *GormGenerationRepository) Get(ctx context.Context, brandID, runID uuid.UUID) (*koidomain.GenerationRun, error) {
	return r.getBy(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("brand_id = ? AND id = ?", brandID, runID)
	})
}

func (r *GormGenerationRepository) List(ctx context.Context, brandID uuid.UUID, page koidomain.PageParams) (*koidomain.Page[koidomain.GenerationRun], error) {
	return r.listBy(ctx, func(db *gorm.DB) *gorm.DB { return db.Where("brand_id = ?", brandID) }, page, "created_at DESC, id DESC")
}

func (r *GormGenerationRepository) ByKey(ctx context.Context, brandID uuid.UUID, key string) (*koidomain.GenerationRun, error) {
	return r.getBy(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("brand_id = ? AND idempotency_key = ?", brandID, key)
	})
}

func (r *GormGenerationRepository) Active(ctx context.Context, brandID uuid.UUID) (*koidomain.GenerationRun, error) {
	return r.getBy(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("brand_id = ? AND state IN ?", brandID,
			[]koidomain.GenerationState{koidomain.GenerationQueued, koidomain.GenerationRunning})
	})
}

func (r *GormGenerationRepository) HasSucceededInitial(ctx context.Context, brandID uuid.UUID) (bool, error) {
	db, err := r.query(ctx)
	if err != nil {
		return false, err
	}

	var count int64

	err = db.Model(&koidomain.GenerationRun{}).Where("brand_id = ? AND mode = ? AND state = ?",
		brandID, koidomain.GenerationInitial, koidomain.GenerationSucceeded).Count(&count).Error

	return count > 0, mapDBError(err)
}

func (r *GormGenerationRepository) CreateAndEnqueue(ctx context.Context, run *koidomain.GenerationRun) error {
	if r.client == nil {
		return koidomain.ErrFeatureUnavailable
	}

	brand, err := NewGormBrandRepository(r.db).Get(ctx, run.BrandID)
	if err != nil {
		return err
	}
	if r.cloudMode {
		limit, err := cloudPromptLimit(ctx, r.db, brand.OrganizationID)
		if err != nil {
			return err
		}
		if limit == 0 {
			return koidomain.ErrBillingRequired
		}
		if run.Mode == koidomain.GenerationPrompts {
			count, err := storedPromptCount(ctx, r.db, brand.OrganizationID)
			if err != nil {
				return err
			}
			if count >= int64(limit) {
				return koidomain.ErrPromptLimitReached
			}
		}
	}

	tx := r.db.WithContext(ctx)
	sqlTx, ok := tx.Statement.ConnPool.(*sql.Tx)
	if !ok {
		return errors.New("generation enqueue requires a SQL transaction")
	}

	if err := tx.Create(run).Error; err != nil {
		return mapDBError(err)
	}

	inserted, err := r.client.InsertTx(ctx, sqlTx, GenerationJobArgs{RunID: run.ID}, &river.InsertOpts{Queue: "generation", MaxAttempts: 3})
	if err != nil {
		return err
	}

	run.RiverJobID = &inserted.Job.ID

	return mapDBError(tx.Model(run).Update("river_job_id", inserted.Job.ID).Error)
}

var _ koidomain.GenerationRepository = (*GormGenerationRepository)(nil)
