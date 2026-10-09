package koidatabase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/koihq/koi/pkg/koidomain"
)

// Worker operations are internal and identify the run through its persisted River job.
func (s *Store) BeginVisibilityAttempt(ctx context.Context, runID uuid.UUID, jobID int64, attempt int) (*koidomain.VisibilityRun, []koidomain.VisibilityResult, error) {
	var run koidomain.VisibilityRun
	var results []koidomain.VisibilityResult

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&run, "id = ? AND river_job_id = ?", runID, jobID).Error; err != nil {
			return mapDBError(err)
		}

		if run.State == koidomain.VisibilitySucceeded || run.State == koidomain.VisibilityFailed {
			return nil
		}

		if attempt < run.Attempt {
			return errors.New("visibility attempt changed")
		}

		now := time.Now().UTC()
		if run.StartedAt == nil {
			run.StartedAt = &now
		}

		run.State, run.Attempt = koidomain.VisibilityRunning, attempt
		if err := tx.Model(&run).Updates(map[string]any{
			"state":      run.State,
			"attempt":    attempt,
			"started_at": run.StartedAt,
		}).Error; err != nil {
			return err
		}

		return tx.Where("run_id = ? AND outcome IS NULL", runID).Order("checked_at ASC NULLS FIRST, prompt_position ASC, engine ASC").Find(&results).Error
	})
	if err != nil {
		return nil, nil, err
	}

	if run.State == koidomain.VisibilitySucceeded || run.State == koidomain.VisibilityFailed {
		return nil, nil, nil
	}

	return &run, results, nil
}

// Contact is persisted before HTTP. A retry uses the same cell ID even if the reply was lost.
func (s *Store) ContactVisibilityTask(ctx context.Context, runID uuid.UUID, attempt int, result *koidomain.VisibilityResult, now time.Time) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockVisibilityAttempt(tx, runID, attempt); err != nil {
			return err
		}

		values := map[string]any{"checked_at": now}
		if result.CloroTaskID == nil && result.SubmittedAt == nil {
			values["submitted_at"] = now
			result.SubmittedAt = &now
		}

		return tx.Model(&koidomain.VisibilityResult{}).Where("id = ? AND run_id = ? AND outcome IS NULL", result.ID, runID).Updates(values).Error
	})
}

func (s *Store) SaveVisibilityTask(ctx context.Context, runID uuid.UUID, attempt int, cellID uuid.UUID, taskID string) error {
	if taskID == "" {
		return errors.New("missing Cloro task ID")
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockVisibilityAttempt(tx, runID, attempt); err != nil {
			return err
		}

		result := tx.Model(&koidomain.VisibilityResult{}).
			Where("id = ? AND run_id = ? AND outcome IS NULL AND (cloro_task_id IS NULL OR cloro_task_id = ?)", cellID, runID, taskID).
			Update("cloro_task_id", taskID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("Cloro task identity changed")
		}

		return nil
	})
}

func (s *Store) SaveVisibilityResult(ctx context.Context, runID uuid.UUID, attempt int, result koidomain.VisibilityResult) error {
	if result.RunID != runID || result.Outcome == nil {
		return errors.New("invalid visibility completion")
	}

	if err := result.Validate(); err != nil {
		return err
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockVisibilityAttempt(tx, runID, attempt); err != nil {
			return err
		}

		// A replay never overwrites an answer or its original fetched_at timestamp.
		return tx.Model(&koidomain.VisibilityResult{}).Where("id = ? AND run_id = ? AND outcome IS NULL", result.ID, runID).
			Select("outcome", "fetched_at", "raw_answer", "sources", "error_code").Updates(&result).Error
	})
}

func lockVisibilityAttempt(tx *gorm.DB, runID uuid.UUID, attempt int) error {
	var run koidomain.VisibilityRun
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&run, "id = ?", runID).Error; err != nil {
		return mapDBError(err)
	}

	if run.State != koidomain.VisibilityRunning || run.Attempt != attempt {
		return errors.New("visibility attempt changed")
	}

	return nil
}

func (s *Store) FinishVisibilityAttempt(ctx context.Context, runID uuid.UUID, attempt int) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockVisibilityAttempt(tx, runID, attempt); err != nil {
			return err
		}

		var pending int64
		if err := tx.Model(&koidomain.VisibilityResult{}).Where("run_id = ? AND outcome IS NULL", runID).Count(&pending).Error; err != nil {
			return err
		}

		if pending != 0 {
			return errors.New("visibility matrix is incomplete")
		}

		return tx.Model(&koidomain.VisibilityRun{}).Where("id = ?", runID).Updates(map[string]any{
			"state":       koidomain.VisibilitySucceeded,
			"finished_at": time.Now().UTC(),
			"error_code":  nil,
		}).Error
	})
}

func (s *Store) RecordVisibilityFailure(ctx context.Context, runID uuid.UUID, jobID int64, attempt int, code string, terminal bool) error {
	values := map[string]any{"error_code": code}
	if terminal {
		values["state"] = koidomain.VisibilityFailed
		values["finished_at"] = time.Now().UTC()
	}

	return s.db.WithContext(ctx).Model(&koidomain.VisibilityRun{}).
		Where("id = ? AND river_job_id = ? AND attempt <= ? AND state IN ?", runID, jobID, attempt,
			[]koidomain.VisibilityState{koidomain.VisibilityQueued, koidomain.VisibilityRunning}).Updates(values).Error
}

func (s *Store) ReconcileVisibilityRuns(ctx context.Context) error {
	// Historical stub cells stay readable, but queued sample jobs must never become paid tasks.
	if err := s.db.WithContext(ctx).Exec(`UPDATE visibility_runs AS v SET state = 'failed', error_code = 'sample_run_retired', finished_at = now()
		WHERE v.state IN ('queued', 'running') AND EXISTS (
			SELECT 1 FROM visibility_results AS r WHERE r.run_id = v.id AND r.collector = 'stub'
		)`).Error; err != nil {
		return err
	}

	return s.db.WithContext(ctx).Exec(`UPDATE visibility_runs AS v SET state = 'failed', error_code = 'job_lost', finished_at = now()
		WHERE v.state IN ('queued', 'running') AND NOT EXISTS (
			SELECT 1 FROM river_job AS j WHERE j.id = v.river_job_id AND j.state NOT IN ('cancelled', 'discarded', 'completed')
		)`).Error
}
