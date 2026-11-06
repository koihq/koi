package database

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/domain"
)

type GenerationJobArgs struct {
	RunID uuid.UUID `json:"run_id"`
}

func (GenerationJobArgs) Kind() string { return "koi_generation" }

type GormGenerationRepository struct {
	*ListRepository[domain.GenerationRun, domain.BrandFilter]
	client    *river.Client[*sql.Tx]
	cloudMode bool
}

func NewGenerationRepository(db *gorm.DB, client *river.Client[*sql.Tx], cloudMode bool) *GormGenerationRepository {
	base := newBrandChildRepository[domain.GenerationRun](db)

	return &GormGenerationRepository{
		ListRepository: newListRepository(base, filterByBrand, fixedOrder[domain.BrandFilter]("created_at DESC, id DESC")),
		client:         client,
		cloudMode:      cloudMode,
	}
}

func (r *GormGenerationRepository) ByKey(ctx context.Context, brandID uuid.UUID, key string) (*domain.GenerationRun, error) {
	return r.getBy(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("brand_id = ? AND idempotency_key = ?", brandID, key)
	})
}

func (r *GormGenerationRepository) Active(ctx context.Context, brandID uuid.UUID) (*domain.GenerationRun, error) {
	return r.getBy(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("brand_id = ? AND state IN ?", brandID,
			[]domain.GenerationState{domain.GenerationQueued, domain.GenerationRunning})
	})
}

func (r *GormGenerationRepository) HasSucceeded(ctx context.Context, brandID uuid.UUID) (bool, error) {
	db, err := r.query(ctx)
	if err != nil {
		return false, err
	}

	var count int64

	err = db.Model(&domain.GenerationRun{}).Where("brand_id = ? AND state = ?",
		brandID, domain.GenerationSucceeded).Count(&count).Error

	return count > 0, mapDBError(err)
}

func (r *GormGenerationRepository) CreateAndEnqueue(ctx context.Context, run *domain.GenerationRun) error {
	if r.client == nil {
		return domain.ErrFeatureUnavailable
	}

	brand, err := NewBrandRepository(r.db).Get(ctx, run.BrandID)
	if err != nil {
		return err
	}
	if r.cloudMode {
		limit, err := cloudPromptLimit(ctx, r.db, brand.OrganizationID)
		if err != nil {
			return err
		}
		if limit == 0 {
			return domain.ErrBillingRequired
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

var _ domain.GenerationRepository = (*GormGenerationRepository)(nil)
