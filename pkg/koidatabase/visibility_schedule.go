package koidatabase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/koiauth"
	"github.com/koihq/koi/pkg/koidomain"
)

func (r *GormVisibilityRepository) FinishReview(ctx context.Context, brandID uuid.UUID, kind string) error {
	if kind != "prompts" && kind != "competitors" {
		return koidomain.Validation(koidomain.FieldError{Field: "review", Code: koidomain.FieldInvalid})
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		brand, err := NewGormBrandRepository(tx).LockForUpdate(ctx, brandID)
		if err != nil {
			return err
		}

		if _, err := NewGormGenerationRepository(tx, r.client).Active(ctx, brandID); err == nil {
			return koidomain.Validation(koidomain.FieldError{Field: "generation", Code: "active"})
		} else if !errors.Is(err, koidomain.ErrNotFound) {
			return err
		}

		now := time.Now().UTC()
		if kind == "prompts" {
			var count int64
			if err := tx.Model(&koidomain.Prompt{}).Where("brand_id = ?", brandID).Count(&count).Error; err != nil {
				return err
			}

			if count == 0 {
				return koidomain.Validation(koidomain.FieldError{Field: "prompts", Code: "empty"})
			}

			if err := tx.Model(&koidomain.Prompt{}).Where("brand_id = ?", brandID).Update("confirmed", true).Error; err != nil {
				return err
			}
			brand.PromptsReviewedAt = &now
		} else {
			if err := tx.Model(&koidomain.Competitor{}).Where("brand_id = ? AND source <> ?", brandID, koidomain.CompetitorSourceObserved).Update("confirmed", true).Error; err != nil {
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

		return NewGormVisibilityRepository(tx, r.client, r.enabled).schedule(ctx, brand, now)
	})
}

func (r *GormVisibilityRepository) Tracking(ctx context.Context, brandID uuid.UUID) (*koidomain.TrackingStatus, error) {
	brand, err := NewGormBrandRepository(r.db).Get(ctx, brandID)
	if err != nil {
		return nil, err
	}

	status := &koidomain.TrackingStatus{
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

func (r *GormVisibilityRepository) blockedReason(ctx context.Context, brand *koidomain.Brand) (string, error) {
	readiness := koidomain.TrackingReadiness{
		CollectionEnabled:   r.enabled && r.client != nil,
		PromptsReviewed:     brand.PromptsReviewedAt != nil,
		CompetitorsReviewed: brand.CompetitorsReviewedAt != nil,
	}
	if !readiness.CollectionEnabled || !readiness.PromptsReviewed || !readiness.CompetitorsReviewed {
		return koidomain.TrackingBlockedReason(readiness), nil
	}

	if err := r.db.WithContext(ctx).Raw("SELECT EXISTS (SELECT 1 FROM prompts WHERE brand_id = ?)", brand.ID).Scan(&readiness.HasPrompts).Error; err != nil {
		return "", err
	}
	if !readiness.HasPrompts {
		return koidomain.TrackingBlockedReason(readiness), nil
	}

	_, err := NewGormGenerationRepository(r.db, r.client).Active(ctx, brand.ID)
	if err != nil && !errors.Is(err, koidomain.ErrNotFound) {
		return "", err
	}
	readiness.GenerationActive = err == nil

	return koidomain.TrackingBlockedReason(readiness), nil
}

// DispatchVisibility uses persisted UTC slots; a restart does not reset the cadence.
func (s *Store) DispatchVisibility(ctx context.Context, now time.Time) error {
	if !s.liveCollection || s.enqueueClient == nil {
		return nil
	}

	var brands []koidomain.Brand
	if err := s.db.WithContext(ctx).Where(`next_visibility_at <= ? OR
		(visibility_started_at IS NULL AND prompts_reviewed_at IS NOT NULL AND competitors_reviewed_at IS NOT NULL)`, now).
		Find(&brands).Error; err != nil {
		return err
	}

	for _, candidate := range brands {
		ownerCtx := koiauth.WithIdentity(ctx, koiauth.Identity{UserID: candidate.UserID})
		if err := s.Atomic(ownerCtx, func(tx koidomain.Store) error {
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
func (r *GormVisibilityRepository) schedule(ctx context.Context, brand *koidomain.Brand, now time.Time) error {
	if brand.NextVisibilityAt != nil && brand.NextVisibilityAt.After(now) {
		return nil
	}

	anchor := now
	if brand.VisibilityStartedAt != nil {
		anchor = *brand.VisibilityStartedAt
	}
	slot, next := koidomain.NextVisibilitySchedule(anchor, now)
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
	} else if !errors.Is(err, koidomain.ErrNotFound) {
		return err
	}

	inputs, err := r.Inputs(ctx, brand.ID)
	if err != nil {
		return err
	}
	run := &koidomain.VisibilityRun{
		BrandID:       brand.ID,
		ScheduledFor:  &slot,
		BrandSnapshot: brand.Snapshot(),
		Engines:       koidomain.VisibilityEngines(),
		State:         koidomain.VisibilityQueued,
		TotalResults:  int64(len(inputs.Prompts) * len(koidomain.VisibilityEngines())),
	}
	results := make([]koidomain.VisibilityResult, 0, run.TotalResults)
	for position, prompt := range inputs.Prompts {
		for _, engine := range run.Engines {
			results = append(results, koidomain.VisibilityResult{
				PromptID:       prompt.ID,
				PromptText:     prompt.Text,
				PromptPosition: position,
				Engine:         engine,
				Collector:      koidomain.CollectorCloro,
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

func invalidateReview(tx *gorm.DB, brandID uuid.UUID, kind string) error {
	return tx.Model(&koidomain.Brand{}).Where("id = ?", brandID).Update(kind+"_reviewed_at", nil).Error
}
