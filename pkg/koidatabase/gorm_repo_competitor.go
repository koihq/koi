package koidatabase

import (
	"context"
	"slices"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/koidomain"
)

type GormCompetitorRepository struct {
	*Repository[koidomain.Competitor]
}

func NewGormCompetitorRepository(db *gorm.DB) *GormCompetitorRepository {
	return &GormCompetitorRepository{
		Repository: newScopedRepository[koidomain.Competitor](db, childOwnerScope("competitors")),
	}
}

func (r *GormCompetitorRepository) Create(ctx context.Context, competitor *koidomain.Competitor) error {
	if _, err := r.Repository.query(ctx); err != nil {
		return err
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := NewGormBrandRepository(tx).LockForUpdate(ctx, competitor.BrandID); err != nil {
			return err
		}
		if err := checkCompetitorURLs(tx, *competitor); err != nil {
			return err
		}
		if err := NewGormCompetitorRepository(tx).Repository.Create(ctx, competitor); err != nil {
			return err
		}
		return invalidateReview(tx, competitor.BrandID, "competitors")
	})
}

func applyCompetitorFilter(db *gorm.DB, filter koidomain.CompetitorFilter) *gorm.DB {
	db = applyChildListFilter(db, filter.ID, filter.BrandID, filter.Confirmed)
	if filter.Domain != "" {
		db = db.Where("domain = ?", filter.Domain)
	}

	if filter.Query != "" {
		pattern := likePattern(filter.Query)
		db = db.Where("(name ILIKE ? ESCAPE '!' OR domain ILIKE ? ESCAPE '!' OR github_repo ILIKE ? ESCAPE '!')", pattern, pattern, pattern)
	}

	return db
}

func (r *GormCompetitorRepository) List(ctx context.Context, filter koidomain.CompetitorFilter) (*koidomain.Page[koidomain.Competitor], error) {
	return r.listBy(ctx, func(db *gorm.DB) *gorm.DB { return applyCompetitorFilter(db, filter) }, filter.Page, "created_at ASC, id ASC")
}

func (r *GormCompetitorRepository) Update(ctx context.Context, id uuid.UUID, update koidomain.CompetitorUpdate) error {
	currentEntity, err := r.Get(ctx, id)
	if err != nil {
		return err
	}
	filter := koidomain.CompetitorFilter{ID: id, BrandID: currentEntity.BrandID}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := NewGormBrandRepository(tx).LockForUpdate(ctx, filter.BrandID); err != nil {
			return err
		}

		repo := NewGormCompetitorRepository(tx)
		competitor, err := repo.Get(ctx, filter.ID)
		if err != nil {
			return err
		}

		columns := applyCompetitorUpdate(competitor, update)
		if len(columns) == 0 {
			return nil
		}
		competitor.Normalize()
		if err := competitor.Validate(); err != nil {
			return err
		}
		if err := checkCompetitorURLs(tx, *competitor); err != nil {
			return err
		}

		if err := repo.updateBy(ctx, func(db *gorm.DB) *gorm.DB { return applyCompetitorFilter(db, filter) }, competitor, columns, true); err != nil {
			return err
		}
		if update.Confirmed != nil && !*update.Confirmed && competitor.Source != koidomain.CompetitorSourceObserved {
			return invalidateReview(tx, filter.BrandID, "competitors")
		}
		return nil
	})
}

func applyCompetitorUpdate(competitor *koidomain.Competitor, update koidomain.CompetitorUpdate) []string {
	columns := []string{}
	if update.Name != nil {
		competitor.Name = *update.Name
		columns = append(columns, "name")
	}
	if update.Domain != nil {
		competitor.Domain = *update.Domain
		columns = append(columns, "domain")
	}
	if update.GitHubRepo != nil {
		competitor.GitHubRepo = *update.GitHubRepo
		columns = append(columns, "github_repo")
	}
	if update.Aliases != nil {
		competitor.Aliases = *update.Aliases
		columns = append(columns, "aliases")
	}
	if update.AdditionalURLs != nil {
		competitor.AdditionalURLs = *update.AdditionalURLs
		columns = append(columns, "additional_urls")
	}
	if update.Confirmed != nil {
		competitor.Confirmed = *update.Confirmed
		columns = append(columns, "confirmed")
	}
	return columns
}

// Callers hold the brand lock, including worker-side generated and observed inserts.
func checkCompetitorURLs(tx *gorm.DB, competitor koidomain.Competitor) error {
	var existing []koidomain.Competitor
	if err := tx.Where("brand_id = ? AND id <> ?", competitor.BrandID, competitor.ID).Find(&existing).Error; err != nil {
		return err
	}

	for _, other := range existing {
		if competitor.SharesIdentityURL(other) {
			field := "additional_urls"
			if competitor.Domain != "" && (koidomain.Competitor{Domain: competitor.Domain}).SharesIdentityURL(other) {
				field = "domain"
			} else if competitor.GitHubRepo != "" && (koidomain.Competitor{GitHubRepo: competitor.GitHubRepo}).SharesIdentityURL(other) {
				field = "github_repo"
			}
			return koidomain.Validation(koidomain.FieldError{Field: field, Code: koidomain.FieldTaken})
		}
	}
	return nil
}

func (r *GormCompetitorRepository) Delete(ctx context.Context, id uuid.UUID) error {
	currentEntity, err := r.Get(ctx, id)
	if err != nil {
		return err
	}
	filter := koidomain.CompetitorFilter{ID: id, BrandID: currentEntity.BrandID}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := NewGormBrandRepository(tx).LockForUpdate(ctx, filter.BrandID); err != nil {
			return err
		}

		repo := NewGormCompetitorRepository(tx)
		competitor, err := repo.Get(ctx, filter.ID)
		if err != nil {
			return err
		}
		if competitor.Source == koidomain.CompetitorSourceObserved {
			if err := dismissCompetitor(tx, *competitor); err != nil {
				return err
			}
		}
		if err := repo.deleteBy(ctx, func(db *gorm.DB) *gorm.DB { return applyCompetitorFilter(db, filter) }, true); err != nil {
			return err
		}
		if competitor.Confirmed || competitor.Source != koidomain.CompetitorSourceObserved {
			return invalidateReview(tx, filter.BrandID, "competitors")
		}
		return nil
	})
}

func dismissCompetitor(tx *gorm.DB, competitor koidomain.Competitor) error {
	var states []koidomain.CompetitorObservation
	if err := tx.Where("brand_id = ?", competitor.BrandID).Find(&states).Error; err != nil {
		return err
	}

	found := false
	for _, state := range states {
		observation := koidomain.AppearanceObservation{Name: state.Identity.Name, URLs: state.Identity.IdentityURLs()}
		index, _ := koidomain.ResolveObservation(observation, []koidomain.Competitor{competitor})
		if state.Identity.ID != competitor.ID && index < 0 {
			continue
		}

		found = true
		identity := competitor
		identity.Aliases = slices.Concat(identity.Aliases, []string{state.Identity.Name})
		identity.Normalize()
		state.Identity, state.Dismissed = identity, true
		if err := tx.Select("identity", "dismissed").Updates(&state).Error; err != nil {
			return err
		}
	}
	if !found {
		state := koidomain.CompetitorObservation{
			BrandID:   competitor.BrandID,
			Key:       koidomain.AppearanceNameKey(competitor.Name),
			Dismissed: true,
			Identity:  competitor,
		}
		return tx.Create(&state).Error
	}
	return nil
}
