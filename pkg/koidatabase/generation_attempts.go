package koidatabase

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/koihq/koi/pkg/koidomain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// BeginGenerationAttempt returns nil when a previous attempt already committed its result.
func (s *Store) BeginGenerationAttempt(ctx context.Context, runID uuid.UUID, jobID int64, attempt int) (*koidomain.GenerationRun, koidomain.GenerationInput, error) {
	var run koidomain.GenerationRun
	var input koidomain.GenerationInput

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&run, "id = ? AND river_job_id = ?", runID, jobID).Error; err != nil {
			return err
		}

		if run.State == koidomain.GenerationSucceeded || run.State == koidomain.GenerationFailed {
			return nil
		}

		now := time.Now().UTC()
		if err := tx.Model(&run).Updates(map[string]any{
			"state":         koidomain.GenerationRunning,
			"attempt":       attempt,
			"started_at":    now,
			"next_retry_at": nil,
			"error_code":    nil,
		}).Error; err != nil {
			return err
		}

		input = koidomain.GenerationInput{
			Brand:               run.BrandSnapshot,
			Mode:                run.Mode,
			ExistingPrompts:     []string{},
			ExistingCompetitors: []koidomain.CompetitorIdentity{},
		}
		if run.Mode != koidomain.GenerationCompetitors {
			if err := tx.Model(&koidomain.Prompt{}).Where("brand_id = ?", run.BrandID).Order("created_at, id").Pluck("text", &input.ExistingPrompts).Error; err != nil {
				return err
			}
		}

		if run.Mode != koidomain.GenerationPrompts {
			var companies []koidomain.Competitor
			if err := tx.Select("name", "domain", "github_repo").Where("brand_id = ?", run.BrandID).Order("created_at, id").Find(&companies).Error; err != nil {
				return err
			}

			for _, company := range companies {
				input.ExistingCompetitors = append(input.ExistingCompetitors, koidomain.CompetitorIdentity{
					Name:       company.Name,
					Domain:     company.Domain,
					GitHubRepo: company.GitHubRepo,
				})
			}
		}

		return nil
	})
	if err != nil {
		return nil, koidomain.GenerationInput{}, err
	}

	if run.State == koidomain.GenerationSucceeded || run.State == koidomain.GenerationFailed {
		return nil, koidomain.GenerationInput{}, nil
	}

	return &run, input, nil
}

func (s *Store) FinishGenerationAttempt(ctx context.Context, runID uuid.UUID, attempt int, result koidomain.GenerationResult) (int, int, error) {
	var promptCount, competitorCount int

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var run koidomain.GenerationRun
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&run, "id = ?", runID).Error; err != nil {
			return err
		}

		if run.State == koidomain.GenerationSucceeded {
			promptCount, competitorCount = run.PromptCount, run.CompetitorCount

			return nil
		}

		if run.State != koidomain.GenerationRunning || run.Attempt != attempt {
			return errors.New("generation attempt changed")
		}

		// This lock also serializes competing generation commits for the brand.
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&koidomain.Brand{}, "id = ?", run.BrandID).Error; err != nil {
			return err
		}

		if run.Mode != koidomain.GenerationCompetitors {
			count, err := insertGeneratedPrompts(tx, run.BrandID, result.Prompts)
			if err != nil {
				return err
			}

			promptCount = count
		}

		if run.Mode != koidomain.GenerationPrompts {
			count, err := insertGeneratedCompetitors(tx, run, result.Competitors)
			if err != nil {
				return err
			}

			competitorCount = count
		}
		if promptCount > 0 {
			if err := invalidateReview(tx, run.BrandID, "prompts"); err != nil {
				return err
			}
		}
		if competitorCount > 0 {
			if err := invalidateReview(tx, run.BrandID, "competitors"); err != nil {
				return err
			}
		}
		return tx.Model(&run).Updates(map[string]any{
			"state":            koidomain.GenerationSucceeded,
			"prompt_count":     promptCount,
			"competitor_count": competitorCount,
			"finished_at":      time.Now().UTC(),
			"next_retry_at":    nil,
			"error_code":       nil,
		}).Error
	})

	return promptCount, competitorCount, err
}

func insertGeneratedPrompts(tx *gorm.DB, brandID uuid.UUID, suggestions []string) (int, error) {
	var existing []string
	if err := tx.Model(&koidomain.Prompt{}).Where("brand_id = ?", brandID).Pluck("text", &existing).Error; err != nil {
		return 0, err
	}

	seen := make(map[string]bool, len(existing)+len(suggestions))

	for _, value := range existing {
		seen[strings.ToLower(strings.Join(strings.Fields(value), " "))] = true
	}

	count := 0

	for _, value := range suggestions {
		prompt := koidomain.Prompt{BrandID: brandID, Text: value, Source: koidomain.PromptSourceDrafted}
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

func insertGeneratedCompetitors(tx *gorm.DB, run koidomain.GenerationRun, suggestions []koidomain.CompetitorIdentity) (int, error) {
	var domains []string
	if err := tx.Model(&koidomain.Competitor{}).Where("brand_id = ?", run.BrandID).Pluck("domain", &domains).Error; err != nil {
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
	if err := tx.Model(&koidomain.Competitor{}).Where("brand_id = ?", run.BrandID).Pluck("github_repo", &repos).Error; err != nil {
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
		company := koidomain.Competitor{
			BrandID:    run.BrandID,
			Name:       value.Name,
			Domain:     value.Domain,
			GitHubRepo: value.GitHubRepo,
			Source:     koidomain.CompetitorSourceDrafted,
		}

		company.Normalize()
		if len(company.Name) > 255 ||
			(company.Domain != "" && domainSeen[company.Domain]) ||
			(company.GitHubRepo != "" && repoSeen[company.GitHubRepo]) ||
			company.Validate() != nil {
			continue
		}

		if err := checkCompetitorURLs(tx, company); err != nil {
			if errors.Is(err, koidomain.ErrValidation) {
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
		values["state"] = koidomain.GenerationFailed
		values["finished_at"] = time.Now().UTC()
	}

	return s.db.WithContext(ctx).Model(&koidomain.GenerationRun{}).
		Where("id = ? AND state = ? AND attempt = ?", runID, koidomain.GenerationRunning, attempt).Updates(values).Error
}

// ReconcileGenerationRuns closes active runs whose River jobs can no longer execute.
func (s *Store) ReconcileGenerationRuns(ctx context.Context) error {
	return s.db.WithContext(ctx).Exec(`UPDATE generation_runs AS g SET state = 'failed', error_code = 'job_lost', finished_at = now(), next_retry_at = NULL
		WHERE g.state IN ('queued', 'running') AND NOT EXISTS (
			SELECT 1 FROM river_job AS j WHERE j.id = g.river_job_id AND j.state NOT IN ('cancelled', 'discarded', 'completed')
		)`).Error
}
