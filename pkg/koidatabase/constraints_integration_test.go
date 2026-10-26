package koidatabase

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/koidomain"
)

func TestForeignKeysAndMembershipCardinality(t *testing.T) {
	db, store, ctx, brand := liveVisibilityFixture(t, false)
	type constraint struct {
		Name   string
		Action string
	}
	var rows []constraint
	require.NoError(t, db.Raw(`SELECT conname AS name, confdeltype::text AS action
 FROM pg_constraint WHERE contype = 'f' AND connamespace = current_schema()::regnamespace`).Scan(&rows).Error)
	actions := map[string]string{}
	for _, row := range rows {
		actions[row.Name] = row.Action
	}
	for name, action := range map[string]string{
		"fk_memberships_user":                       "c",
		"fk_memberships_organization":               "c",
		"fk_brands_organization":                    "r",
		"fk_billing_subscriptions_organization":     "r",
		"fk_billing_checkout_sessions_organization": "r",
		"fk_billing_change_requests_organization":   "r",
		"fk_prompts_brand":                          "c",
		"fk_visibility_results_run":                 "c",
	} {
		require.Equal(t, action, actions[name], name)
	}
	require.NotContains(t, actions, "fk_visibility_results_prompt")
	require.NoError(t, Migrate(db))

	orphan := &koidomain.Brand{OrganizationID: uuid.New(), Name: "Orphan", Domain: "orphan.dev", Country: "IT"}
	require.ErrorIs(t, mapDBError(db.Create(orphan).Error), koidomain.ErrNotFound)
	require.ErrorIs(t, mapDBError(db.Delete(&koidomain.Organization{}, "id = ?", brand.OrganizationID).Error), koidomain.ErrResourceConflict)

	var membership koidomain.Membership
	require.NoError(t, db.Where("organization_id = ?", brand.OrganizationID).First(&membership).Error)
	other := &koidomain.Organization{Name: "Other"}
	require.NoError(t, db.Create(other).Error)
	require.Error(t, db.Create(&koidomain.Membership{UserID: membership.UserID, OrganizationID: other.ID, Role: koidomain.RoleOwner}).Error)

	// A second member belongs to the same tenant and sees its resources.
	member := &koidomain.User{Email: uuid.NewString() + "@example.com", PasswordHash: "hash"}
	require.NoError(t, db.Create(member).Error)
	require.NoError(t, db.Create(&koidomain.Membership{UserID: member.ID, OrganizationID: brand.OrganizationID, Role: koidomain.RoleOwner}).Error)
	_, err := store.BrandRepo().Get(testIdentity(t, store, member.ID), brand.ID)
	require.NoError(t, err)
	require.NoError(t, db.Delete(member).Error)
	var remaining int64
	require.NoError(t, db.Model(&koidomain.Membership{}).Where("user_id = ?", member.ID).Count(&remaining).Error)
	require.Zero(t, remaining)
	_, err = store.BrandRepo().Get(ctx, brand.ID)
	require.NoError(t, err)
}
