// Package demoseed creates local UI examples, never live visibility measurements.
package demoseed

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/domain"
)

const Email = "demo@koi.local"
const Password = "KoiDemo2026!"

var scenarios = []struct {
	key  string
	name string
	days int
}{
	{"history", "Koi · Storico", 180},
	{"empty", "Koi · Vuoto", 0},
	{"first", "Koi · Prima raccolta", 1},
}

func demoID(key string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(repository+"/demo/"+key))
}

func entity(key string, at time.Time) domain.BaseEntity {
	return domain.BaseEntity{
		ID:        demoID(key),
		CreatedAt: at,
		UpdatedAt: at,
	}
}

// Run preserves existing scenarios unless reset is requested. The caller owns
// schema initialization. All writes, including reset, commit together.
func Run(ctx context.Context, db *gorm.DB, now time.Time, reset bool) error {
	now = now.UTC()

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Serialize repeated invocations, including creation before a user row exists.
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", int64(724150629)).Error; err != nil {
			return err
		}

		if err := ensureAccount(tx, now); err != nil {
			return err
		}

		for _, scenario := range scenarios {
			brandID := demoID(scenario.key)
			var existing domain.Brand
			err := tx.First(&existing, "id = ?", brandID).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if err == nil {
				if existing.OrganizationID != demoID("organization") {
					return fmt.Errorf("demo brand %s belongs to another organization", brandID)
				}
				if !reset {
					continue
				}

				// Existing foreign-key cascades remove only this brand's children.
				if err := tx.Delete(&existing).Error; err != nil {
					return err
				}
			}

			if err := seedBrand(tx, scenario.key, scenario.name, scenario.days, now); err != nil {
				return fmt.Errorf("seed %s: %w", scenario.key, err)
			}
		}

		return nil
	})
}

func ensureAccount(tx *gorm.DB, now time.Time) error {
	var user domain.User
	err := tx.Where("email = ? OR id = ?", Email, demoID("user")).First(&user).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if err == nil {
		if user.ID != demoID("user") || user.Email != Email {
			return fmt.Errorf("demo email or user ID is already occupied by another account")
		}

		var membership domain.Membership
		if err := tx.Where("user_id = ?", user.ID).First(&membership).Error; err != nil {
			return fmt.Errorf("demo membership: %w", err)
		}
		if membership.OrganizationID != demoID("organization") || membership.Role != domain.RoleOwner {
			return fmt.Errorf("demo account has an unexpected organization or role")
		}

		return nil
	}

	hash, err := domain.HashPassword(Password)
	if err != nil {
		return err
	}

	user = domain.User{
		BaseEntity:   entity("user", now.AddDate(0, 0, -180)),
		Email:        Email,
		PasswordHash: hash,
	}
	organization := domain.Organization{
		BaseEntity: entity("organization", user.CreatedAt),
		Name:       "Koi Demo",
	}
	membership := domain.Membership{
		BaseEntity:     entity("membership", user.CreatedAt),
		UserID:         user.ID,
		OrganizationID: organization.ID,
		Role:           domain.RoleOwner,
	}

	for _, record := range []interface{ Validate() error }{&user, &organization, &membership} {
		if err := record.Validate(); err != nil {
			return err
		}
		if err := tx.Create(record).Error; err != nil {
			return err
		}
	}

	return nil
}

func seedBrand(tx *gorm.DB, key, name string, days int, now time.Time) error {
	createdAt := now.AddDate(0, 0, -days)
	brand := domain.Brand{
		BaseEntity:     entity(key, createdAt),
		OrganizationID: demoID("organization"),
		Name:           name,
		GitHubRepo:     repository,
		Country:        "US",
		Description:    "Demo dataset: illustrative answers and simulated metrics. Koi is a developer- and agent-first AEO tool for tracking brand mentions, citations, and competing recommendations.",
	}
	if days > 0 {
		startedAt := now.AddDate(0, 0, 1-days)
		brand.TrackingConfirmedAt = &createdAt
		brand.VisibilityStartedAt = &startedAt
		// No NextVisibilityAt: snapshots stay offline even with Cloro configured.
	}

	brand.Normalize()
	if err := brand.Validate(); err != nil {
		return err
	}
	if err := tx.Create(&brand).Error; err != nil {
		return err
	}
	if days == 0 {
		return nil
	}

	promptCount, competitorCount := 32, len(competitors)
	if days == 1 {
		promptCount, competitorCount = 5, 4
	}
	for i := range competitorCount {
		spec := competitors[i]
		competitor := domain.Competitor{
			BaseEntity: entity(fmt.Sprintf("%s/competitor/%d", key, i), createdAt),
			BrandID:    brand.ID,
			Name:       spec.name,
			Domain:     spec.domain,
			Source:     domain.CompetitorSourceManual,
			Confirmed:  true,
		}
		competitor.Normalize()
		if err := competitor.Validate(); err != nil {
			return err
		}
		if err := tx.Create(&competitor).Error; err != nil {
			return err
		}
	}

	for i := range promptCount {
		prompt := domain.Prompt{
			BaseEntity: entity(fmt.Sprintf("%s/prompt/%d", key, i), createdAt.Add(time.Duration(i)*time.Millisecond)),
			BrandID:    brand.ID,
			Text:       prompts[i].text,
			Source:     domain.PromptSourceManual,
			Confirmed:  true,
			Inactive:   i >= 26 && i < 29,
		}
		if i >= 29 {
			prompt.Source = domain.PromptSourceDrafted
			prompt.Confirmed = false
		}
		prompt.Normalize()
		if err := prompt.Validate(); err != nil {
			return err
		}
		if err := tx.Create(&prompt).Error; err != nil {
			return err
		}
	}

	snapshot := brand.Snapshot()
	snapshot.Name = "Koi"
	for day := range days {
		at := now.AddDate(0, 0, day+1-days)
		count := 26
		if day < 90 {
			// Three now-stopped prompts were tracked earlier.
			count = 29
		}
		if days == 1 {
			count = 5
		}
		if err := seedRun(tx, key, snapshot, day, count, competitorCount, at); err != nil {
			return err
		}
	}

	return nil
}

func seedRun(tx *gorm.DB, key string, snapshot domain.BrandSnapshot, day, count, competitorCount int, at time.Time) error {
	startedAt := at.Add(-2 * time.Minute)
	if midnight := at.Truncate(24 * time.Hour); startedAt.Before(midnight) {
		startedAt = midnight
	}
	run := domain.VisibilityRun{
		BaseEntity:    entity(fmt.Sprintf("%s/run/%d", key, day), startedAt),
		BrandID:       demoID(key),
		BrandSnapshot: snapshot,
		Engines:       domain.VisibilityEngines(),
		State:         domain.VisibilityQueued,
		TotalResults:  int64(count * len(domain.VisibilityEngines())),
		StartedAt:     &startedAt,
		FinishedAt:    &at,
		Attempt:       1,
		FastStart:     day == 0,
	}
	if err := run.Validate(); err != nil {
		return err
	}
	run.State = domain.VisibilitySucceeded
	results := make([]domain.VisibilityResult, 0, run.TotalResults)

	for index := range count {
		for engineIndex, engine := range run.Engines {
			result := domain.VisibilityResult{
				BaseEntity:     entity(fmt.Sprintf("%s/run/%d/%d/%s", key, day, index, engine), at),
				BrandID:        run.BrandID,
				RunCreatedAt:   run.CreatedAt,
				RunID:          run.ID,
				PromptID:       demoID(fmt.Sprintf("%s/prompt/%d", key, index)),
				PromptText:     prompts[index].text,
				PromptPosition: index,
				Engine:         engine,
				Collector:      domain.CollectorStub,
				FetchedAt:      &at,
				Sources:        []string{},
				NamedEntities:  []domain.NamedEntity{},
				Observations:   []domain.AppearanceObservation{},
			}
			fillAnswer(&result, snapshot, index, engineIndex, day, competitorCount)
			if err := result.Validate(); err != nil {
				return err
			}
			results = append(results, result)
		}
	}

	if err := tx.Create(&run).Error; err != nil {
		return err
	}

	return tx.CreateInBatches(&results, 100).Error
}

func fillAnswer(result *domain.VisibilityResult, snapshot domain.BrandSnapshot, prompt, engine, day, competitorCount int) {
	// Alternate improvement and decline by prompt, with weekly variation and an
	// engine offset. Some prompts intentionally remain at 0% or 100% visibility.
	phase := day / 30
	if prompt%2 == 1 {
		phase = 5 - phase
	}
	value := (prompt*7 + engine*3 + day%7) % 20
	visible := value < 4+phase*2+engine
	if prompt == 0 {
		visible = true
	}
	if prompt == 1 {
		visible = false
	}
	if day > 0 && (day+prompt*3+engine)%97 == 0 {
		result.Outcome = new(domain.OutcomeError)
		result.ErrorCode = new("stub_error")
		return
	}

	first := competitors[(prompt+engine+day/14)%competitorCount]
	second := competitors[(prompt+engine+day/14+1)%competitorCount]
	notes := topicNotes[prompts[prompt].topic]
	plain := fmt.Sprintf("For the question %q:\n\n%s\n\n%s\n%s", result.PromptText, notes, first.summary, second.summary)
	markdown := fmt.Sprintf("### Evaluating AI visibility tools\n\n**Question:** %s\n\n%s\n\n- [%s](https://%s/): %s\n- [%s](https://%s/): %s", result.PromptText, notes, first.name, first.domain, first.summary, second.name, second.domain, second.summary)
	if (prompt+engine)%5 == 0 {
		plain = fmt.Sprintf("%s and %s are options for AI search visibility monitoring. Compare saved answers and cited sources using a consistent US prompt set.", first.name, second.name)
		markdown = fmt.Sprintf("[%s](https://%s/) and [%s](https://%s/) are options for AI search visibility monitoring. Compare saved answers and cited sources using a consistent US prompt set.", first.name, first.domain, second.name, second.domain)
	}
	result.Sources = append(result.Sources, "https://"+first.domain+"/")
	if (prompt+engine)%2 == 0 {
		result.Sources = append(result.Sources, "https://"+second.domain+"/")
	}
	if visible {
		koi := "Koi tracks brand mentions and citations across four answer engines, preserves prompt-level evidence, and supports Markdown briefs for a founder's existing agent. It can be self-hosted; live collection uses the operator's own Cloro account."
		plain += "\n\n" + koi
		markdown += "\n\n**Developer workflow:** " + koi
		if (prompt+engine+day)%3 != 0 {
			result.Sources = append(result.Sources, repository)
			markdown += "\n\n[Koi source repository](" + repository + ")"
		}
	}
	if (prompt+engine)%4 == 0 {
		markdown += "\n\n#### A practical review\n\n1. Keep a fixed prompt set and US market.\n2. Read the answer and inspect each source.\n3. Compare competing recommendations.\n4. Assign one evidence-backed content change and measure again.\n\n" + topicNotes[(prompts[prompt].topic+1)%len(topicNotes)]
		plain += "\n\nPractical review: keep a fixed prompt set and US market, inspect sources, compare competing recommendations, and measure again after one content change."
	}

	// Both answer representations carry the disclosure; no disclaimer contains
	// the brand name, so it cannot turn a measured miss into a mention.
	result.RawAnswer = "Demo example — simulated answer; no engine was contacted.\n\n" + plain
	result.AnswerMarkdown = "> Demo example — simulated answer; no engine was contacted.\n\n" + markdown
	result.NamedEntities = []domain.NamedEntity{
		{Type: "organization", Name: first.name},
		{Type: "organization", Name: second.name},
	}
	if visible {
		result.NamedEntities = append(result.NamedEntities, domain.NamedEntity{Type: "organization", Name: "Koi"})
	}

	// Branded questions may themselves name Koi; score the actual persisted answer.
	result.Outcome = new(domain.ScoreVisibility(snapshot, result.RawAnswer, result.Sources))
	result.Observations = domain.ExtractObservations(*result)
}
