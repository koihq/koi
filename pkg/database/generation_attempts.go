package database

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/koihq/koi/pkg/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// BeginGenerationAttempt returns nil when a previous attempt already committed its result.
func (s *Store) BeginGenerationAttempt(ctx context.Context, runID uuid.UUID, jobID int64, attempt int) (*domain.GenerationRun, domain.GenerationInput, error) {
	var run domain.GenerationRun
	var input domain.GenerationInput

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&run, "id = ? AND river_job_id = ?", runID, jobID).Error; err != nil {
			return err
		}

		if run.State == domain.GenerationSucceeded || run.State == domain.GenerationFailed {
			return nil
		}
		promptTarget := domain.InitialDraftPromptLimit
		if s.cloudMode {
			available, err := cloudGenerationBudget(ctx, tx, run.BrandID)
			if err != nil {
				return err
			}
			promptTarget = min(promptTarget, available)
		}

		now := time.Now().UTC()
		if err := tx.Model(&run).Updates(map[string]any{
			"state":         domain.GenerationRunning,
			"attempt":       attempt,
			"started_at":    now,
			"next_retry_at": nil,
			"error_code":    nil,
		}).Error; err != nil {
			return err
		}

		input = domain.GenerationInput{
			Brand:               run.BrandSnapshot,
			PromptTarget:        promptTarget,
			ExistingPrompts:     []string{},
			ExistingCompetitors: []domain.CompetitorIdentity{},
		}
		if err := tx.Model(&domain.Prompt{}).Where("brand_id = ?", run.BrandID).Order("created_at, id").Pluck("text", &input.ExistingPrompts).Error; err != nil {
			return err
		}

		var companies []domain.Competitor
		if err := tx.Select("name", "domain", "github_repo").Where("brand_id = ?", run.BrandID).Order("created_at, id").Find(&companies).Error; err != nil {
			return err
		}

		for _, company := range companies {
			input.ExistingCompetitors = append(input.ExistingCompetitors, domain.CompetitorIdentity{
				Name:       company.Name,
				Domain:     company.Domain,
				GitHubRepo: company.GitHubRepo,
			})
		}

		return nil
	})
	if err != nil {
		return nil, domain.GenerationInput{}, err
	}

	if run.State == domain.GenerationSucceeded || run.State == domain.GenerationFailed {
		return nil, domain.GenerationInput{}, nil
	}

	return &run, input, nil
}

func (s *Store) FinishGenerationAttempt(ctx context.Context, runID uuid.UUID, attempt int, result domain.GenerationResult) (int, int, error) {
	var promptCount, competitorCount int

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var run domain.GenerationRun
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&run, "id = ?", runID).Error; err != nil {
			return err
		}

		if run.State == domain.GenerationSucceeded {
			promptCount, competitorCount = run.PromptCount, run.CompetitorCount

			return nil
		}

		if run.State != domain.GenerationRunning || run.Attempt != attempt {
			return errors.New("generation attempt changed")
		}

		// This lock also serializes competing generation commits for the brand.
		var brand domain.Brand
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&brand, "id = ?", run.BrandID).Error; err != nil {
			return err
		}
		promptBudget := min(domain.InitialDraftPromptLimit, len(result.Prompts))
		if s.cloudMode {
			if err := lockOrganization(ctx, tx, brand.OrganizationID); err != nil {
				return err
			}
			available, err := cloudGenerationBudget(ctx, tx, run.BrandID)
			if err != nil {
				return err
			}
			promptBudget = min(promptBudget, available)
		}

		var err error

		promptCount, err = insertGeneratedPrompts(tx, run.BrandID, result.Prompts, promptBudget)
		if err != nil {
			return err
		}

		competitorCount, err = insertGeneratedCompetitors(tx, run, result.Competitors)
		if err != nil {
			return err
		}

		return tx.Model(&run).Updates(map[string]any{
			"state":            domain.GenerationSucceeded,
			"prompt_count":     promptCount,
			"competitor_count": competitorCount,
			"finished_at":      time.Now().UTC(),
			"next_retry_at":    nil,
			"error_code":       nil,
		}).Error
	})

	return promptCount, competitorCount, err
}

// SaveGenerationSummary stores what the research understood while the draft continues.
func (s *Store) SaveGenerationSummary(ctx context.Context, runID uuid.UUID, summary domain.GenerationSummary) error {
	return s.db.WithContext(ctx).Model(&domain.GenerationRun{}).Where("id = ?", runID).
		Select("summary").Updates(&domain.GenerationRun{Summary: &summary}).Error
}

func cloudGenerationBudget(ctx context.Context, db *gorm.DB, brandID uuid.UUID) (int, error) {
	var brand domain.Brand
	if err := db.WithContext(ctx).Select("organization_id").Where("id = ?", brandID).Take(&brand).Error; err != nil {
		return 0, err
	}

	limit, err := cloudPromptLimit(ctx, db, brand.OrganizationID)
	if err != nil {
		return 0, err
	}

	if limit == 0 {
		return 0, domain.ErrBillingRequired
	}

	count, err := storedPromptCount(ctx, db, brand.OrganizationID)
	if err != nil {
		return 0, err
	}

	return domain.GenerationBudget(limit, int(count)), nil
}

func insertGeneratedPrompts(tx *gorm.DB, brandID uuid.UUID, suggestions []string, budget int) (int, error) {
	var existing []string
	if err := tx.Model(&domain.Prompt{}).Where("brand_id = ?", brandID).Pluck("text", &existing).Error; err != nil {
		return 0, err
	}

	seen := make(map[string]bool, len(existing)+len(suggestions))

	for _, value := range existing {
		seen[strings.ToLower(strings.Join(strings.Fields(value), " "))] = true
	}

	count := 0

	for _, value := range suggestions {
		if count >= budget {
			break
		}
		prompt := domain.Prompt{BrandID: brandID, Text: value, Source: domain.PromptSourceDrafted}
		prompt.Normalize()
		key := strings.ToLower(prompt.Text)
		if len(prompt.Text) > 2000 || seen[key] || prompt.Validate() != nil {
			continue
		}

		seen[key] = true
		inserted := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&prompt)
		if inserted.Error != nil {
			return 0, inserted.Error
		}

		count += int(inserted.RowsAffected)
	}

	return count, nil
}

func insertGeneratedCompetitors(tx *gorm.DB, run domain.GenerationRun, suggestions []domain.CompetitorIdentity) (int, error) {
	var domains []string
	if err := tx.Model(&domain.Competitor{}).Where("brand_id = ?", run.BrandID).Pluck("domain", &domains).Error; err != nil {
		return 0, err
	}

	domainSeen := make(map[string]bool, len(domains)+len(suggestions)+1)

	for _, value := range domains {
		if value != "" {
			domainSeen[strings.ToLower(value)] = true
		}
	}

	if run.BrandSnapshot.Domain != "" {
		domainSeen[strings.ToLower(run.BrandSnapshot.Domain)] = true
	}

	var repos []string
	if err := tx.Model(&domain.Competitor{}).Where("brand_id = ?", run.BrandID).Pluck("github_repo", &repos).Error; err != nil {
		return 0, err
	}

	repoSeen := make(map[string]bool, len(repos)+len(suggestions)+1)

	for _, value := range repos {
		if value != "" {
			repoSeen[strings.ToLower(strings.TrimSuffix(value, "/"))] = true
		}
	}

	if run.BrandSnapshot.GitHubRepo != "" {
		repoSeen[strings.ToLower(strings.TrimSuffix(run.BrandSnapshot.GitHubRepo, "/"))] = true
	}

	count := 0

	for _, value := range suggestions {
		company := domain.Competitor{
			BrandID:    run.BrandID,
			Name:       value.Name,
			Domain:     value.Domain,
			GitHubRepo: value.GitHubRepo,
			Source:     domain.CompetitorSourceDrafted,
		}

		company.Normalize()
		if len(company.Name) > 255 ||
			(company.Domain != "" && domainSeen[company.Domain]) ||
			(company.GitHubRepo != "" && repoSeen[company.GitHubRepo]) ||
			company.Validate() != nil {
			continue
		}

		if err := checkCompetitorURLs(tx, company); err != nil {
			if errors.Is(err, domain.ErrValidation) {
				continue
			}
			return 0, err
		}

		if company.Domain != "" {
			domainSeen[company.Domain] = true
		}

		if company.GitHubRepo != "" {
			repoSeen[company.GitHubRepo] = true
		}

		inserted := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&company)
		if inserted.Error != nil {
			return 0, inserted.Error
		}

		count += int(inserted.RowsAffected)
	}

	return count, nil
}

func (s *Store) RecordGenerationFailure(ctx context.Context, runID uuid.UUID, attempt int, code string, retryAt *time.Time) error {
	values := map[string]any{"error_code": code, "next_retry_at": retryAt}
	if retryAt == nil {
		values["state"] = domain.GenerationFailed
		values["finished_at"] = time.Now().UTC()
	}

	return s.db.WithContext(ctx).Model(&domain.GenerationRun{}).
		Where("id = ? AND state = ? AND attempt = ?", runID, domain.GenerationRunning, attempt).Updates(values).Error
}

// ReconcileGenerationRuns closes active runs whose River jobs can no longer execute.
func (s *Store) ReconcileGenerationRuns(ctx context.Context) error {
	return s.db.WithContext(ctx).Exec(`UPDATE generation_runs AS g SET state = 'failed', error_code = 'job_lost', finished_at = now(), next_retry_at = NULL
		WHERE g.state IN ('queued', 'running') AND NOT EXISTS (
			SELECT 1 FROM river_job AS j WHERE j.id = g.river_job_id AND j.state NOT IN ('cancelled', 'discarded', 'completed')
		)`).Error
}
