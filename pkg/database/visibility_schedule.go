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

// StartTracking confirms the brand's lists once and schedules its first run. Later calls change
// nothing, so a retry never creates an extra run or confirms a later suggestion.
func (r *GormVisibilityRepository) StartTracking(ctx context.Context, brandID uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		brand, err := NewBrandRepository(tx).LockForUpdate(ctx, brandID)
		if err != nil {
			return err
		}

		if brand.TrackingConfirmedAt != nil {
			return nil
		}

		if r.cloudMode {
			if err := requirePromptCapacity(ctx, tx, brand.OrganizationID, 0); err != nil {
				return err
			}
		}

		if _, err := NewGenerationRepository(tx, r.client, r.cloudMode).Active(ctx, brandID); err == nil {
			return domain.Validation(domain.FieldError{Field: "generation", Code: "active"})
		} else if !errors.Is(err, domain.ErrNotFound) {
			return err
		}

		var active int64
		if err := tx.Model(&domain.Prompt{}).Where("brand_id = ? AND NOT inactive", brandID).Count(&active).Error; err != nil {
			return err
		}

		if active == 0 {
			return domain.Validation(domain.FieldError{Field: "prompts", Code: "empty"})
		}

		if err := tx.Model(&domain.Prompt{}).Where("brand_id = ?", brandID).Update("confirmed", true).Error; err != nil {
			return err
		}

		if err := tx.Model(&domain.Competitor{}).
			Where("brand_id = ? AND source <> ?", brandID, domain.CompetitorSourceObserved).
			Update("confirmed", true).Error; err != nil {
			return err
		}

		now := time.Now().UTC()
		brand.TrackingConfirmedAt = &now

		if err := tx.Model(brand).Update("tracking_confirmed_at", now).Error; err != nil {
			return err
		}

		if !r.enabled || r.client == nil {
			return nil
		}

		repository := NewVisibilityRepository(tx, r.client, r.enabled, r.cloudMode)
		repository.firstLookDailyLimit = r.firstLookDailyLimit

		return repository.schedule(ctx, brand, now)
	})
}

func (r *GormVisibilityRepository) Tracking(ctx context.Context, brandID uuid.UUID) (*domain.TrackingStatus, error) {
	brand, err := NewBrandRepository(r.db).Get(ctx, brandID)
	if err != nil {
		return nil, err
	}

	status := &domain.TrackingStatus{
		Confirmed: brand.TrackingConfirmedAt != nil,
		StartedAt: brand.VisibilityStartedAt,
		NextRunAt: brand.NextVisibilityAt,
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
		CollectionEnabled: r.enabled && r.client != nil,
		Confirmed:         brand.TrackingConfirmedAt != nil,
	}
	if !readiness.CollectionEnabled || !readiness.Confirmed {
		return domain.TrackingBlockedReason(readiness), nil
	}
	if r.cloudMode {
		firstLook, err := cloudFirstLook(ctx, r.db, brand.OrganizationID)
		if err != nil {
			return "", err
		}

		if firstLook {
			reason, err := r.firstLookBlockedReason(ctx, brand)
			if err != nil || reason != "" {
				return reason, err
			}
		}

		allowed, err := cloudCollectionAllowed(ctx, r.db, brand.OrganizationID, firstLook)
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

	return domain.TrackingBlockedReason(readiness), nil
}

// firstLookBlockedReason limits unpaid collection: one run per organization, one organization per
// site, and a daily total that caps what abuse can spend, since sign-up is otherwise unmetered.
func (r *GormVisibilityRepository) firstLookBlockedReason(ctx context.Context, brand *domain.Brand) (string, error) {
	db := r.db.WithContext(ctx)

	var used bool
	if err := db.Raw(`SELECT EXISTS (SELECT 1 FROM visibility_runs
		JOIN brands ON brands.id = visibility_runs.brand_id
		WHERE brands.organization_id = ?)`, brand.OrganizationID).Scan(&used).Error; err != nil {
		return "", err
	}
	if used {
		return domain.TrackingBlockedFirstLookUsed, nil
	}

	var taken bool
	if err := db.Raw(`SELECT EXISTS (SELECT 1 FROM visibility_runs
		JOIN brands ON brands.id = visibility_runs.brand_id
		WHERE brands.organization_id <> ? AND (
			(? <> '' AND lower(brands.domain) = lower(?)) OR
			(? <> '' AND lower(brands.github_repo) = lower(?))
		))`, brand.OrganizationID, brand.Domain, brand.Domain, brand.GitHubRepo, brand.GitHubRepo).
		Scan(&taken).Error; err != nil {
		return "", err
	}
	if taken {
		return domain.TrackingBlockedFirstLookSiteUsed, nil
	}

	var today int64
	if err := db.Model(&domain.VisibilityRun{}).
		Where("fast_start AND created_at > ?", time.Now().UTC().Add(-24*time.Hour)).
		Count(&today).Error; err != nil {
		return "", err
	}
	if today >= int64(r.firstLookDailyLimit) {
		return domain.TrackingBlockedFirstLookPaused, nil
	}

	return "", nil
}

func (s *Store) ownerContext(ctx context.Context, organizationID uuid.UUID) (context.Context, error) {
	var owner domain.Membership
	if err := s.db.WithContext(ctx).
		Where("organization_id = ? AND role = ?", organizationID, domain.RoleOwner).
		Take(&owner).Error; err != nil {
		return nil, err
	}

	return auth.WithIdentity(ctx, auth.Identity{UserID: owner.UserID, OrganizationID: owner.OrganizationID, Role: owner.Role}), nil
}

// StartTrackingAsOwner starts collection from a worker, which has no request identity.
func (s *Store) StartTrackingAsOwner(ctx context.Context, brandID uuid.UUID) error {
	var brand domain.Brand
	if err := s.db.WithContext(ctx).Select("organization_id").Where("id = ?", brandID).Take(&brand).Error; err != nil {
		return mapDBError(err)
	}

	ownerCtx, err := s.ownerContext(ctx, brand.OrganizationID)
	if err != nil {
		return mapDBError(err)
	}

	return s.VisibilityRepo().StartTracking(ownerCtx, brandID)
}

// DispatchVisibility uses persisted UTC slots; a restart does not reset the cadence.
func (s *Store) DispatchVisibility(ctx context.Context, now time.Time) error {
	if !s.liveCollection || s.enqueueClient == nil {
		return nil
	}

	var brands []domain.Brand
	if err := s.db.WithContext(ctx).Where(`next_visibility_at <= ? OR
		(visibility_started_at IS NULL AND tracking_confirmed_at IS NOT NULL)`, now).
		Find(&brands).Error; err != nil {
		return err
	}

	for _, candidate := range brands {
		ownerCtx, err := s.ownerContext(ctx, candidate.OrganizationID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}
		if err != nil {
			return err
		}
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
