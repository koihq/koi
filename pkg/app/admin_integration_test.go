package app

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/alexedwards/scs/postgresstore"
	"github.com/alexedwards/scs/v2"
	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverdatabasesql"
	"github.com/riverqueue/river/rivermigrate"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/koihq/koi/internal/testdb"
	"github.com/koihq/koi/pkg/auth"
	"github.com/koihq/koi/pkg/database"
	"github.com/koihq/koi/pkg/domain"
)

type adminFixture struct {
	db    *gorm.DB
	store *database.Store
}

func newAdminFixture(t *testing.T) *adminFixture {
	t.Helper()

	dsn := testdb.URL(t)
	t.Setenv("KOI_DATABASE_URL", dsn)

	db, err := database.Open(dsn)
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

	require.NoError(t, database.Migrate(db))

	migrator, err := rivermigrate.New(riverdatabasesql.New(sqlDB), nil)
	require.NoError(t, err)

	_, err = migrator.Migrate(t.Context(), rivermigrate.DirectionUp, nil)
	require.NoError(t, err)

	return &adminFixture{db: db, store: database.NewStore(db)}
}

// account creates someone with a brand, a prompt, a session, an agent and a queued email.
func (f *adminFixture) account(t *testing.T) *domain.Me {
	t.Helper()

	commander := domain.NewAuthCommander(f.store, domain.AuthOptions{OpenSignup: true})

	me, err := commander.Signup(t.Context(), domain.SignupParams{
		Email:    uuid.NewString() + "@example.com",
		Password: "the-old-password",
	})
	require.NoError(t, err)

	ctx := auth.WithIdentity(t.Context(), auth.Identity{
		UserID:         me.User.ID,
		OrganizationID: me.Organization.ID,
		Role:           me.Role,
	})

	brand, err := domain.NewBrandCommander(f.store).Create(ctx, domain.CreateBrandParams{
		Name:    "Acme",
		Domain:  "acme.example",
		Country: "US",
	})
	require.NoError(t, err)

	_, err = domain.NewPromptCommander(f.store).Create(ctx, brand.ID, domain.CreatePromptParams{Text: "Best private tools?"})
	require.NoError(t, err)

	_, err = domain.NewCredentialService(f.store).IssueTokens(ctx, me.User.ID, "client-1", "Agent", "koi:read", "https://koi.example/mcp", uuid.Nil)
	require.NoError(t, err)

	sqlDB, err := f.db.DB()
	require.NoError(t, err)

	sess := scs.New()
	sess.Store = postgresstore.NewWithCleanupInterval(sqlDB, 0)

	sessionCtx, err := sess.Load(t.Context(), "")
	require.NoError(t, err)

	sess.Put(sessionCtx, "user_id", me.User.ID.String())

	_, _, err = sess.Commit(sessionCtx)
	require.NoError(t, err)

	client, err := river.NewClient(riverdatabasesql.New(sqlDB), &river.Config{})
	require.NoError(t, err)

	_, err = client.Insert(t.Context(), database.WelcomeJobArgs{UserID: me.User.ID}, &river.InsertOpts{Queue: "email"})
	require.NoError(t, err)

	_, err = client.Insert(t.Context(), database.PasswordResetJobArgs{Email: me.User.Email}, &river.InsertOpts{Queue: "email"})
	require.NoError(t, err)

	return me
}

func (f *adminFixture) count(t *testing.T, table string) int64 {
	t.Helper()

	var count int64
	require.NoError(t, f.db.Table(table).Count(&count).Error)

	return count
}

func TestAdminResetPassword(t *testing.T) {
	f := newAdminFixture(t)
	me := f.account(t)

	var stdout bytes.Buffer

	err := RunAdmin(t.Context(), []string{"reset-password", me.User.Email}, strings.NewReader("too-short\n"), &stdout)
	require.ErrorIs(t, err, domain.ErrValidation)

	err = RunAdmin(t.Context(), []string{"reset-password", me.User.Email}, strings.NewReader("the-new-password\n"), &stdout)
	require.NoError(t, err)

	commander := domain.NewAuthCommander(f.store, domain.AuthOptions{})

	_, err = commander.Login(t.Context(), me.User.Email, "the-new-password")
	require.NoError(t, err)

	_, err = commander.Login(t.Context(), me.User.Email, "the-old-password")
	require.ErrorIs(t, err, domain.ErrInvalidCredentials)

	require.Zero(t, f.count(t, "credentials"))
	require.Zero(t, f.count(t, "sessions"))
}

func TestAdminExportAccount(t *testing.T) {
	f := newAdminFixture(t)
	me := f.account(t)
	f.account(t)

	var stdout bytes.Buffer
	require.NoError(t, RunAdmin(t.Context(), []string{"export-account", me.User.Email}, strings.NewReader(""), &stdout))

	var export map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &export))

	require.Contains(t, string(export["user"]), me.User.Email)
	require.Contains(t, string(export["prompts"]), "Best private tools?")
	require.NotContains(t, export, "onboarding_events")
	require.Contains(t, string(export["connections"]), "Agent")
	require.Equal(t, 1, strings.Count(stdout.String(), `"Best private tools?"`), "another account's data stays out")

	var user domain.User
	require.NoError(t, f.db.Where("id = ?", me.User.ID).Take(&user).Error)
	require.NotContains(t, stdout.String(), user.PasswordHash)
	require.NotContains(t, stdout.String(), "token_hash")
	require.NotContains(t, stdout.String(), "password_hash")
}

func TestAdminDeleteAccount(t *testing.T) {
	f := newAdminFixture(t)
	me := f.account(t)

	stdin := strings.NewReader("")

	var stdout bytes.Buffer

	require.Error(t, RunAdmin(t.Context(), []string{"delete-account", me.User.Email}, stdin, &stdout), "needs --yes")
	require.Equal(t, int64(1), f.count(t, "users"))

	require.NoError(t, RunAdmin(t.Context(), []string{"delete-account", me.User.Email, "--yes"}, stdin, &stdout))

	for _, table := range []string{
		"users", "organizations", "memberships", "brands", "prompts", "credentials",
		"sessions", "river_job",
	} {
		require.Zero(t, f.count(t, table), table)
	}
}

func TestAdminDeleteAccountKeepsOtherAccounts(t *testing.T) {
	f := newAdminFixture(t)
	me := f.account(t)
	other := f.account(t)

	require.NoError(t, RunAdmin(t.Context(), []string{"delete-account", me.User.Email, "--yes"}, strings.NewReader(""), &bytes.Buffer{}))

	require.Equal(t, int64(1), f.count(t, "users"))
	require.Equal(t, int64(1), f.count(t, "sessions"))
	require.Equal(t, int64(2), f.count(t, "river_job"), "the other account's emails stay queued")

	var remaining domain.User
	require.NoError(t, f.db.Take(&remaining).Error)
	require.Equal(t, other.User.ID, remaining.ID)
}

func TestAdminDeleteAccountRefuses(t *testing.T) {
	t.Run("a live subscription", func(t *testing.T) {
		f := newAdminFixture(t)
		me := f.account(t)
		end := time.Now().Add(24 * time.Hour)
		customer := "cus_test"

		require.NoError(t, f.db.Model(&domain.Organization{}).Where("id = ?", me.Organization.ID).
			Update("stripe_customer_id", customer).Error)
		require.NoError(t, f.db.Create(&domain.BillingSubscription{
			OrganizationID:       me.Organization.ID,
			StripeSubscriptionID: "sub_test",
			StripePriceID:        "price_test",
			Quantity:             25,
			Status:               "active",
			CurrentPeriodEnd:     &end,
		}).Error)

		err := RunAdmin(t.Context(), []string{"delete-account", me.User.Email, "--yes"}, strings.NewReader(""), &bytes.Buffer{})
		require.ErrorIs(t, err, domain.ErrAccountSubscriptionLive)
		require.Equal(t, int64(1), f.count(t, "users"))

		// A canceled one no longer blocks, and the operator is told what is left in Stripe.
		require.NoError(t, f.db.Model(&domain.BillingSubscription{}).Where("organization_id = ?", me.Organization.ID).
			Update("status", "canceled").Error)

		var stdout bytes.Buffer
		require.NoError(t, RunAdmin(t.Context(), []string{"delete-account", me.User.Email, "--yes"}, strings.NewReader(""), &stdout))
		require.Contains(t, stdout.String(), customer)
		require.Zero(t, f.count(t, "billing_subscriptions"))
	})

	t.Run("other members", func(t *testing.T) {
		f := newAdminFixture(t)
		me := f.account(t)

		teammate := &domain.User{Email: uuid.NewString() + "@example.com", PasswordHash: "hash"}
		require.NoError(t, f.db.Create(teammate).Error)
		require.NoError(t, f.db.Create(&domain.Membership{
			UserID:         teammate.ID,
			OrganizationID: me.Organization.ID,
			Role:           "member",
		}).Error)

		err := RunAdmin(t.Context(), []string{"delete-account", me.User.Email, "--yes"}, strings.NewReader(""), &bytes.Buffer{})
		require.ErrorIs(t, err, domain.ErrAccountHasOtherMembers)
	})

	t.Run("a draft still running", func(t *testing.T) {
		f := newAdminFixture(t)
		me := f.account(t)

		var brand domain.Brand
		require.NoError(t, f.db.Where("organization_id = ?", me.Organization.ID).Take(&brand).Error)
		require.NoError(t, f.db.Create(&domain.GenerationRun{
			BrandID:       brand.ID,
			State:         domain.GenerationRunning,
			ModelSlug:     domain.GenerationModelSlug,
			BrandSnapshot: brand.Snapshot(),
		}).Error)

		err := RunAdmin(t.Context(), []string{"delete-account", me.User.Email, "--yes"}, strings.NewReader(""), &bytes.Buffer{})
		require.ErrorIs(t, err, domain.ErrAccountWorkActive)
	})
}

func TestAdminSeedDemoRefusesLiveCloudBeforeOpeningDatabase(t *testing.T) {
	t.Setenv("KOI_DATABASE_URL", "invalid")
	t.Setenv("KOI_DEPLOYMENT_MODE", "cloud")

	for _, live := range []string{"", "true"} {
		t.Setenv("KOI_LIVE_COLLECTION", live)

		err := RunAdmin(t.Context(), []string{"seed-demo"}, strings.NewReader(""), &bytes.Buffer{})
		require.ErrorContains(t, err, "KOI_LIVE_COLLECTION=false")
	}
}

func TestAdminSeedDemoInCloudStagingGrantsAPlan(t *testing.T) {
	f := newAdminFixture(t)
	t.Setenv("KOI_DEPLOYMENT_MODE", "cloud")
	t.Setenv("KOI_LIVE_COLLECTION", "false")

	for range 2 {
		var out bytes.Buffer
		require.NoError(t, RunAdmin(t.Context(), []string{"seed-demo"}, strings.NewReader(""), &out))
		require.Contains(t, out.String(), "demo@koi.local")
	}

	var subscriptions []domain.BillingSubscription
	require.NoError(t, f.db.Find(&subscriptions).Error)
	require.Len(t, subscriptions, 1)
	require.Equal(t, "active", subscriptions[0].Status)
	require.Greater(t, subscriptions[0].Quantity, 0)
	require.True(t, subscriptions[0].CurrentPeriodEnd.After(time.Now().AddDate(0, 0, 29)))

	var brands int64
	require.NoError(t, f.db.Model(&domain.Brand{}).Count(&brands).Error)
	require.Equal(t, int64(3), brands)
}

func TestAdminSeedDemoSelfHostedHasNoSubscription(t *testing.T) {
	f := newAdminFixture(t)
	t.Setenv("KOI_DEPLOYMENT_MODE", "selfhosted")

	require.NoError(t, RunAdmin(t.Context(), []string{"seed-demo", "--reset"}, strings.NewReader(""), &bytes.Buffer{}))

	var subscriptions int64
	require.NoError(t, f.db.Model(&domain.BillingSubscription{}).Count(&subscriptions).Error)
	require.Zero(t, subscriptions)

	err := RunAdmin(t.Context(), []string{"seed-demo", "--wipe"}, strings.NewReader(""), &bytes.Buffer{})
	require.ErrorContains(t, err, "usage")
}
