package database

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/koihq/koi/pkg/auth"
	"github.com/koihq/koi/pkg/domain"
)

func (r *GormVisibilityRepository) FinishReview(ctx context.Context, brandID uuid.UUID, kind string) error {
	if !validReviewKind(kind) {
		return domain.Validation(domain.FieldError{Field: "review", Code: domain.FieldInvalid})
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		brand, err := NewBrandRepository(tx).LockForUpdate(ctx, brandID)
		if err != nil {
			return err
		}
		if kind == "prompts" && r.cloudMode {
			if err := requirePromptCapacity(ctx, tx, brand.OrganizationID, 0); err != nil {
				return err
			}
		}

		if _, err := NewGenerationRepository(tx, r.client, r.cloudMode).Active(ctx, brandID); err == nil {
			return domain.Validation(domain.FieldError{Field: "generation", Code: "active"})
		} else if !errors.Is(err, domain.ErrNotFound) {
			return err
		}

		now := time.Now().UTC()
		if kind == "prompts" {
			var count int64
			if err := tx.Model(&domain.Prompt{}).Where("brand_id = ?", brandID).Count(&count).Error; err != nil {
				return err
			}

			if count == 0 {
				return domain.Validation(domain.FieldError{Field: "prompts", Code: "empty"})
			}

			if err := tx.Model(&domain.Prompt{}).Where("brand_id = ?", brandID).Update("confirmed", true).Error; err != nil {
				return err
			}
			brand.PromptsReviewedAt = &now
		} else {
			if err := tx.Model(&domain.Competitor{}).Where("brand_id = ? AND source <> ?", brandID, domain.CompetitorSourceObserved).Update("confirmed", true).Error; err != nil {
				return err
			}
			brand.CompetitorsReviewedAt = &now
		}

		if err := tx.Model(brand).Updates(map[string]any{
			"prompts_reviewed_at":     brand.PromptsReviewedAt,
			"competitors_reviewed_at": brand.CompetitorsReviewedAt,
		}).Error; err != nil {
			return err
		}

		// Finishing another review after tracking started never creates an extra run.
		if brand.VisibilityStartedAt != nil || !r.enabled || r.client == nil {
			return nil
		}

		return NewVisibilityRepository(tx, r.client, r.enabled, r.cloudMode).schedule(ctx, brand, now)
	})
}

func (r *GormVisibilityRepository) Tracking(ctx context.Context, brandID uuid.UUID) (*domain.TrackingStatus, error) {
	brand, err := NewBrandRepository(r.db).Get(ctx, brandID)
	if err != nil {
		return nil, err
	}

	status := &domain.TrackingStatus{
		PromptsReviewed:     brand.PromptsReviewedAt != nil,
		CompetitorsReviewed: brand.CompetitorsReviewedAt != nil,
		StartedAt:           brand.VisibilityStartedAt,
		NextRunAt:           brand.NextVisibilityAt,
	}

	reason, err := r.blockedReason(ctx, brand)
	if err != nil {
		return nil, err
	}
	if reason != "" {
		status.BlockedReason = &reason
	}

	return status, nil
}

func (r *GormVisibilityRepository) blockedReason(ctx context.Context, brand *domain.Brand) (string, error) {
	readiness := domain.TrackingReadiness{
		CollectionEnabled:   r.enabled && r.client != nil,
		PromptsReviewed:     brand.PromptsReviewedAt != nil,
		CompetitorsReviewed: brand.CompetitorsReviewedAt != nil,
	}
	if !readiness.CollectionEnabled || !readiness.PromptsReviewed || !readiness.CompetitorsReviewed {
		return domain.TrackingBlockedReason(readiness), nil
	}
	if r.cloudMode {
		allowed, err := cloudTrackingAllowed(ctx, r.db, brand.OrganizationID)
		if err != nil {
			return "", err
		}
		if !allowed {
			return "billing_inactive_or_over_limit", nil
		}
	}

	if err := r.db.WithContext(ctx).Raw("SELECT EXISTS (SELECT 1 FROM prompts WHERE brand_id = ? AND confirmed AND NOT inactive)", brand.ID).Scan(&readiness.HasPrompts).Error; err != nil {
		return "", err
	}
	if !readiness.HasPrompts {
		return domain.TrackingBlockedReason(readiness), nil
	}

	_, err := NewGenerationRepository(r.db, r.client, r.cloudMode).Active(ctx, brand.ID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return "", err
	}
	readiness.GenerationActive = err == nil

	return domain.TrackingBlockedReason(readiness), nil
}

// DispatchVisibility uses persisted UTC slots; a restart does not reset the cadence.
func (s *Store) DispatchVisibility(ctx context.Context, now time.Time) error {
	if !s.liveCollection || s.enqueueClient == nil {
		return nil
	}

	var brands []domain.Brand
	if err := s.db.WithContext(ctx).Where(`next_visibility_at <= ? OR
		(visibility_started_at IS NULL AND prompts_reviewed_at IS NOT NULL AND competitors_reviewed_at IS NOT NULL)`, now).
		Find(&brands).Error; err != nil {
		return err
	}

	for _, candidate := range brands {
		var owner domain.Membership
		if err := s.db.WithContext(ctx).
			Where("organization_id = ? AND role = ?", candidate.OrganizationID, domain.RoleOwner).
			Take(&owner).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return err
		}
		ownerCtx := auth.WithIdentity(ctx, auth.Identity{UserID: owner.UserID, OrganizationID: owner.OrganizationID, Role: owner.Role})
		if err := s.Atomic(ownerCtx, func(tx domain.Store) error {
			brand, err := tx.BrandRepo().LockForUpdate(ownerCtx, candidate.ID)
			if err != nil {
				return err
			}

			return tx.VisibilityRepo().(*GormVisibilityRepository).schedule(ownerCtx, brand, now)
		}); err != nil {
			return err
		}
	}

	return nil
}

// schedule runs only inside the brand-locked transaction. Missed slots are skipped.
func (r *GormVisibilityRepository) schedule(ctx context.Context, brand *domain.Brand, now time.Time) error {
	if brand.NextVisibilityAt != nil && brand.NextVisibilityAt.After(now) {
		return nil
	}

	anchor := now
	if brand.VisibilityStartedAt != nil {
		anchor = *brand.VisibilityStartedAt
	}
	slot, next := domain.NextVisibilitySchedule(anchor, now)
	if brand.VisibilityStartedAt != nil {
		if err := r.db.WithContext(ctx).Model(brand).Update("next_visibility_at", next).Error; err != nil {
			return err
		}
	}

	reason, err := r.blockedReason(ctx, brand)
	if err != nil || reason != "" {
		return err
	}
	if _, err := r.Active(ctx, brand.ID); err == nil {
		return nil
	} else if !errors.Is(err, domain.ErrNotFound) {
		return err
	}

	inputs, err := r.Inputs(ctx, brand.ID)
	if err != nil {
		return err
	}
	// Serialize the first-run decision across brands in the same organization.
	var organization domain.Organization
	if err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&organization, "id = ?", brand.OrganizationID).Error; err != nil {
		return mapDBError(err)
	}
	var existing int64
	if err := r.db.WithContext(ctx).Model(&domain.VisibilityRun{}).
		Joins("JOIN brands ON brands.id = visibility_runs.brand_id").
		Where("brands.organization_id = ?", brand.OrganizationID).Count(&existing).Error; err != nil {
		return err
	}
	run := &domain.VisibilityRun{
		BrandID:       brand.ID,
		FastStart:     existing == 0,
		ScheduledFor:  &slot,
		BrandSnapshot: brand.Snapshot(),
		Engines:       domain.VisibilityEngines(),
		State:         domain.VisibilityQueued,
		TotalResults:  int64(len(inputs.Prompts) * len(domain.VisibilityEngines())),
	}
	results := make([]domain.VisibilityResult, 0, run.TotalResults)
	for position, prompt := range inputs.Prompts {
		for _, engine := range run.Engines {
			results = append(results, domain.VisibilityResult{
				PromptID:       prompt.ID,
				PromptText:     prompt.Text,
				PromptPosition: position,
				Engine:         engine,
				Collector:      domain.CollectorCloro,
				Sources:        []string{},
			})
		}
	}
	if err := r.CreateAndEnqueue(ctx, run, results); err != nil {
		return err
	}

	if brand.VisibilityStartedAt == nil {
		return r.db.WithContext(ctx).Model(brand).Updates(map[string]any{
			"visibility_started_at": slot,
			"next_visibility_at":    next,
		}).Error
	}

	return nil
}

// InvalidateReview asks the owner to review a set again after it changed.
func (r *GormVisibilityRepository) InvalidateReview(ctx context.Context, brandID uuid.UUID, kind string) error {
	if !validReviewKind(kind) {
		return domain.Validation(domain.FieldError{Field: "review", Code: domain.FieldInvalid})
	}

	brands, err := NewBrandRepository(r.db).query(ctx)
	if err != nil {
		return err
	}

	return requireRow(invalidateReviewRows(brands, brandID, kind))
}

func validReviewKind(kind string) bool {
	return kind == domain.ReviewPrompts || kind == domain.ReviewCompetitors
}

func invalidateReview(tx *gorm.DB, brandID uuid.UUID, kind string) error {
	return invalidateReviewRows(tx, brandID, kind).Error
}

func invalidateReviewRows(db *gorm.DB, brandID uuid.UUID, kind string) *gorm.DB {
	return db.Model(&domain.Brand{}).Where("id = ?", brandID).Update(kind+"_reviewed_at", nil)
}
