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

// BeginDraftAttempt returns nil when a previous attempt already committed its result.
func (s *Store) BeginDraftAttempt(ctx context.Context, runID uuid.UUID, jobID int64, attempt int) (*koidomain.GenerationRun, koidomain.DraftInput, error) {
	var run koidomain.GenerationRun
	var input koidomain.DraftInput
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&run, "id = ? AND river_job_id = ?", runID, jobID).Error; err != nil {
			return err
		}
		if run.State == koidomain.GenerationSucceeded || run.State == koidomain.GenerationFailed {
			return nil
		}
		now := time.Now().UTC()
		if err := tx.Model(&run).Updates(map[string]any{"state": koidomain.GenerationRunning, "attempt": attempt, "started_at": now, "next_retry_at": nil, "error_code": nil}).Error; err != nil {
			return err
		}
		input = koidomain.DraftInput{Brand: run.BrandSnapshot, Mode: run.Mode, ExistingPrompts: []string{}, ExistingCompetitors: []koidomain.DraftCompany{}}
		if run.Mode != koidomain.GenerationCompetitors {
			if err := tx.Model(&koidomain.Prompt{}).Where("brand_id = ?", run.BrandID).Order("created_at, id").Pluck("text", &input.ExistingPrompts).Error; err != nil {
				return err
			}
		}
		if run.Mode != koidomain.GenerationPrompts {
			var companies []koidomain.Competitor
			if err := tx.Select("name", "domain").Where("brand_id = ?", run.BrandID).Order("created_at, id").Find(&companies).Error; err != nil {
				return err
			}
			for _, company := range companies {
				input.ExistingCompetitors = append(input.ExistingCompetitors, koidomain.DraftCompany{Name: company.Name, Domain: company.Domain})
			}
		}
		return nil
	})
	if err != nil {
		return nil, koidomain.DraftInput{}, err
	}
	if run.State == koidomain.GenerationSucceeded || run.State == koidomain.GenerationFailed {
		return nil, koidomain.DraftInput{}, nil
	}
	return &run, input, nil
}

func (s *Store) FinishDraftAttempt(ctx context.Context, runID uuid.UUID, attempt int, draft koidomain.DraftOutput) (int, int, error) {
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
		var existingPrompts []string
		if err := tx.Model(&koidomain.Prompt{}).Where("brand_id = ?", run.BrandID).Pluck("text", &existingPrompts).Error; err != nil {
			return err
		}
		promptSeen := make(map[string]bool, len(existingPrompts)+len(draft.Prompts))
		for _, value := range existingPrompts {
			promptSeen[strings.ToLower(strings.Join(strings.Fields(value), " "))] = true
		}
		var existingDomains []string
		if err := tx.Model(&koidomain.Competitor{}).Where("brand_id = ?", run.BrandID).Pluck("domain", &existingDomains).Error; err != nil {
			return err
		}
		domainSeen := make(map[string]bool, len(existingDomains)+len(draft.Competitors)+1)
		for _, value := range existingDomains {
			domainSeen[strings.ToLower(value)] = true
		}
		domainSeen[strings.ToLower(run.BrandSnapshot.Domain)] = true
		if run.Mode != koidomain.GenerationCompetitors {
			for _, value := range draft.Prompts {
				prompt := koidomain.Prompt{BrandID: run.BrandID, Text: value, Source: koidomain.PromptSourceDrafted}
				prompt.Normalize()
				key := strings.ToLower(prompt.Text)
				if len(prompt.Text) > 2000 || promptSeen[key] || prompt.Validate() != nil {
					continue
				}
				promptSeen[key] = true
				result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&prompt)
				if result.Error != nil {
					return result.Error
				}
				promptCount += int(result.RowsAffected)
			}
		}
		if run.Mode != koidomain.GenerationPrompts {
			for _, value := range draft.Competitors {
				company := koidomain.Competitor{BrandID: run.BrandID, Name: value.Name, Domain: value.Domain, Source: koidomain.CompetitorSourceDrafted}
				company.Normalize()
				if len(company.Name) > 255 || domainSeen[company.Domain] || company.Validate() != nil {
					continue
				}
				domainSeen[company.Domain] = true
				result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&company)
				if result.Error != nil {
					return result.Error
				}
				competitorCount += int(result.RowsAffected)
			}
		}
		return tx.Model(&run).Updates(map[string]any{"state": koidomain.GenerationSucceeded, "prompt_count": promptCount, "competitor_count": competitorCount, "finished_at": time.Now().UTC(), "next_retry_at": nil, "error_code": nil}).Error
	})
	return promptCount, competitorCount, err
}

func (s *Store) RecordDraftFailure(ctx context.Context, runID uuid.UUID, attempt int, code string, retryAt *time.Time) error {
	values := map[string]any{"error_code": code, "next_retry_at": retryAt}
	if retryAt == nil {
		values["state"] = koidomain.GenerationFailed
		values["finished_at"] = time.Now().UTC()
	}
	return s.db.WithContext(ctx).Model(&koidomain.GenerationRun{}).
		Where("id = ? AND state = ? AND attempt = ?", runID, koidomain.GenerationRunning, attempt).Updates(values).Error
}

// ReconcileDraftRuns closes active runs whose River jobs can no longer execute.
func (s *Store) ReconcileDraftRuns(ctx context.Context) error {
	return s.db.WithContext(ctx).Exec(`UPDATE generation_runs AS g SET state = 'failed', error_code = 'job_lost', finished_at = now(), next_retry_at = NULL
		WHERE g.state IN ('queued', 'running') AND NOT EXISTS (
			SELECT 1 FROM river_job AS j WHERE j.id = g.river_job_id AND j.state NOT IN ('cancelled', 'discarded', 'completed')
		)`).Error
}
