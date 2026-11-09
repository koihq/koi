package database

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/koihq/koi/pkg/domain"
)

type GormAccountDataRepository struct{ db *gorm.DB }

func NewAccountDataRepository(db *gorm.DB) *GormAccountDataRepository {
	return &GormAccountDataRepository{db: db}
}

func (r *GormAccountDataRepository) AccountFacts(ctx context.Context, userID uuid.UUID) (domain.AccountFacts, error) {
	db := r.db.WithContext(ctx)

	var membership domain.Membership
	if err := db.Where("user_id = ?", userID).Take(&membership).Error; err != nil {
		return domain.AccountFacts{}, mapDBError(err)
	}

	var facts domain.AccountFacts

	err := db.Raw(`SELECT
			coalesce(organizations.stripe_customer_id, '') AS stripe_customer_id,
			coalesce((SELECT status FROM billing_subscriptions
				WHERE organization_id = organizations.id), '') AS subscription_status,
			(SELECT count(*) FROM memberships WHERE organization_id = organizations.id) AS members,
			(SELECT count(*) FROM generation_runs JOIN brands ON brands.id = generation_runs.brand_id
				WHERE brands.organization_id = organizations.id AND generation_runs.state IN ('queued', 'running'))
			+ (SELECT count(*) FROM visibility_runs JOIN brands ON brands.id = visibility_runs.brand_id
				WHERE brands.organization_id = organizations.id AND visibility_runs.state IN ('queued', 'running'))
				AS active_work
		FROM organizations WHERE organizations.id = ?`, membership.OrganizationID).Scan(&facts).Error
	if err != nil {
		return domain.AccountFacts{}, err
	}

	// Scan resets the whole struct, so this has to come after it.
	facts.OrganizationID = membership.OrganizationID

	return facts, nil
}

func (r *GormAccountDataRepository) ExportAccount(ctx context.Context, userID, organizationID uuid.UUID) (map[string]json.RawMessage, error) {
	const brandsOfOrganization = `JOIN brands ON brands.id = t.brand_id WHERE brands.organization_id = ?`

	sections := []struct {
		name  string
		query string
		arg   uuid.UUID
	}{
		{"user", `SELECT email, created_at FROM users WHERE id = ?`, userID},
		{"connections", `SELECT name, kind, created_at, expires_at FROM credentials
			WHERE user_id = ? AND kind = 'oauth_refresh' AND used_at IS NULL`, userID},
		{"organization", `SELECT name, created_at, trial_used_at FROM organizations WHERE id = ?`, organizationID},
		{"subscription", `SELECT status, quantity, current_period_end, trial_end, cancel_at_period_end
			FROM billing_subscriptions WHERE organization_id = ?`, organizationID},
		{"brands", `SELECT name, domain, github_repo, country, description, created_at
			FROM brands WHERE organization_id = ?`, organizationID},
		{"prompts", `SELECT brands.name AS brand, t.text, t.source, t.confirmed, t.inactive, t.created_at
			FROM prompts AS t ` + brandsOfOrganization, organizationID},
		{"competitors", `SELECT brands.name AS brand, t.name, t.domain, t.github_repo, t.source, t.confirmed
			FROM competitors AS t ` + brandsOfOrganization, organizationID},
		{"drafts", `SELECT brands.name AS brand, t.state, t.summary, t.created_at
			FROM generation_runs AS t ` + brandsOfOrganization, organizationID},
		{"runs", `SELECT brands.name AS brand, t.state, t.scheduled_for, t.created_at
			FROM visibility_runs AS t ` + brandsOfOrganization, organizationID},
		{"results", `SELECT brands.name AS brand, t.prompt_text, t.engine, t.outcome, t.mention_verdict,
				t.answer_markdown, t.sources, t.fetched_at
			FROM visibility_results AS t ` + brandsOfOrganization, organizationID},
	}

	export := make(map[string]json.RawMessage, len(sections))

	for _, section := range sections {
		var rows string

		err := r.db.WithContext(ctx).
			Raw(`SELECT coalesce(jsonb_agg(to_jsonb(section_row)), '[]')::text FROM (`+section.query+`) AS section_row`, section.arg).
			Scan(&rows).Error
		if err != nil {
			return nil, err
		}

		export[section.name] = json.RawMessage(rows)
	}

	return export, nil
}

// DeleteAccount deletes children before parents where the foreign key restricts it. Credentials
// have no foreign key, and River's jobs only name their subject inside their arguments.
func (r *GormAccountDataRepository) DeleteAccount(ctx context.Context, user *domain.User, organizationID uuid.UUID) error {
	db := r.db.WithContext(ctx)

	err := db.Exec(`DELETE FROM river_job WHERE
		(kind IN (?, ?) AND args->>'user_id' = ?)
		OR (kind = ? AND args->>'email' = ?)
		OR (kind IN (?, ?) AND args->>'run_id' IN (
			SELECT runs.id::text FROM generation_runs AS runs JOIN brands ON brands.id = runs.brand_id
			WHERE brands.organization_id = ?))
		OR (kind IN (?, ?) AND args->>'run_id' IN (
			SELECT runs.id::text FROM visibility_runs AS runs JOIN brands ON brands.id = runs.brand_id
			WHERE brands.organization_id = ?))`,
		WelcomeJobArgs{}.Kind(), PasswordChangedJobArgs{}.Kind(), user.ID.String(),
		PasswordResetJobArgs{}.Kind(), user.Email,
		GenerationJobArgs{}.Kind(), GenerationJobArgs{}.Kind(), organizationID,
		VisibilityJobArgs{}.Kind(), FirstCheckEmailJobArgs{}.Kind(), organizationID,
	).Error
	if err != nil {
		return err
	}

	steps := []struct {
		model any
		where string
		arg   uuid.UUID
	}{
		{&domain.Credential{}, "user_id = ?", user.ID},
		{&domain.OAuthCode{}, "user_id = ?", user.ID},
		{&domain.BillingChangeRequest{}, "organization_id = ?", organizationID},
		{&domain.BillingCheckoutSession{}, "organization_id = ?", organizationID},
		{&domain.BillingSubscription{}, "organization_id = ?", organizationID},
		{&domain.Membership{}, "organization_id = ?", organizationID},
		{&domain.Brand{}, "organization_id = ?", organizationID},
		{&domain.Organization{}, "id = ?", organizationID},
		{&domain.User{}, "id = ?", user.ID},
	}

	for _, step := range steps {
		if err := db.Where(step.where, step.arg).Delete(step.model).Error; err != nil {
			return err
		}
	}

	return nil
}

var _ domain.AccountDataRepository = (*GormAccountDataRepository)(nil)
