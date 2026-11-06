package database

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/domain"
)

type GormCompetitorRepository struct {
	*ListRepository[domain.Competitor, domain.CompetitorFilter]
	observations *Repository[domain.CompetitorObservation]
}

func NewCompetitorRepository(db *gorm.DB) *GormCompetitorRepository {
	base := newBrandChildRepository[domain.Competitor](db, "name", "domain", "github_repo", "aliases", "additional_urls", "confirmed")

	return &GormCompetitorRepository{
		ListRepository: newListRepository(base, applyCompetitorFilter, fixedOrder[domain.CompetitorFilter]("created_at ASC, id ASC")),
		observations:   newBrandChildRepository[domain.CompetitorObservation](db, "identity", "dismissed"),
	}
}

func applyCompetitorFilter(db *gorm.DB, filter domain.CompetitorFilter) *gorm.DB {
	db = filterByBrand(db, filter.BrandFilter)

	if filter.Confirmed != nil {
		db = db.Where("confirmed = ?", *filter.Confirmed)
	}

	if filter.Domain != "" {
		db = db.Where("domain = ?", filter.Domain)
	}

	if filter.Query != "" {
		pattern := likePattern(filter.Query)
		db = db.Where("(name ILIKE ? ESCAPE '!' OR domain ILIKE ? ESCAPE '!' OR github_repo ILIKE ? ESCAPE '!')", pattern, pattern, pattern)
	}

	return db
}

func (r *GormCompetitorRepository) ForBrand(ctx context.Context, brandID uuid.UUID) ([]domain.Competitor, error) {
	db, err := r.query(ctx)
	if err != nil {
		return nil, err
	}

	var competitors []domain.Competitor
	err = db.Where("brand_id = ?", brandID).Order("created_at, id").Find(&competitors).Error

	return competitors, mapDBError(err)
}

func (r *GormCompetitorRepository) Observations(ctx context.Context, brandID uuid.UUID) ([]domain.CompetitorObservation, error) {
	db, err := r.observations.query(ctx)
	if err != nil {
		return nil, err
	}

	var observations []domain.CompetitorObservation
	err = db.Where("brand_id = ?", brandID).Find(&observations).Error

	return observations, mapDBError(err)
}

// SaveObservations inserts new observations and updates the dismissal of existing ones.
func (r *GormCompetitorRepository) SaveObservations(ctx context.Context, observations []domain.CompetitorObservation) error {
	for i := range observations {
		observation := &observations[i]

		save := r.observations.Update
		if observation.ID == uuid.Nil {
			save = r.observations.Create
		}

		if err := save(ctx, observation); err != nil {
			return err
		}
	}

	return nil
}

// checkCompetitorURLs guards worker-side generated and observed inserts, which hold the brand lock.
func checkCompetitorURLs(tx *gorm.DB, competitor domain.Competitor) error {
	var existing []domain.Competitor
	if err := tx.Where("brand_id = ?", competitor.BrandID).Find(&existing).Error; err != nil {
		return err
	}

	return competitor.ConflictsWith(existing)
}

var _ domain.CompetitorRepository = (*GormCompetitorRepository)(nil)
