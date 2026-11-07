package database

import (
	"encoding/base64"
	"fmt"
	"math/rand/v2"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/argon2"

	"github.com/koihq/koi/pkg/auth"
	"github.com/koihq/koi/pkg/domain"
)

const testAudience = "https://koi.example/mcp"

func TestPasswordResetEndsEveryGrant(t *testing.T) {
	db, store, _, _ := liveVisibilityFixture(t, true)
	store.WithPasswordResetEmail(true)

	commander := domain.NewAuthCommander(store, domain.AuthOptions{OpenSignup: true})
	service := domain.NewCredentialService(store)

	me, err := commander.Signup(t.Context(), domain.SignupParams{Email: "reset@example.com", Password: "old-password"})
	require.NoError(t, err)

	tokens, err := service.IssueTokens(t.Context(), me.User.ID, "client-1", "Agent", "koi:read", testAudience, uuid.Nil)
	require.NoError(t, err)

	require.NoError(t, service.CreateCode(t.Context(), &domain.OAuthCode{
		Hash:        "code-hash",
		UserID:      me.User.ID,
		ClientID:    "client-1",
		ClientName:  "Agent",
		RedirectURI: "https://client.example/callback",
		Challenge:   "challenge",
		Scope:       "koi:read",
		ExpiresAt:   time.Now().Add(time.Minute),
	}))

	// A made-up link changes nothing, so it cannot be used to sign someone's agents out.
	_, err = commander.ResetPassword(t.Context(), "invented-token", "new-password-1")
	require.ErrorIs(t, err, domain.ErrPasswordResetInvalid)

	resolved, err := service.Resolve(t.Context(), tokens.AccessToken)
	require.NoError(t, err)
	require.NotNil(t, resolved)

	_, raw, err := commander.IssuePasswordReset(t.Context(), me.User.Email)
	require.NoError(t, err)

	_, err = commander.ResetPassword(t.Context(), raw, "new-password-1")
	require.NoError(t, err)

	resolved, err = service.Resolve(t.Context(), tokens.AccessToken)
	require.NoError(t, err)
	require.Nil(t, resolved)

	_, err = service.RefreshTokens(t.Context(), tokens.RefreshToken, "client-1", testAudience)
	require.ErrorIs(t, err, domain.ErrInvalidCredentials)

	var codes, notices int64
	require.NoError(t, db.Model(&domain.OAuthCode{}).Where("user_id = ?", me.User.ID).Count(&codes).Error)
	require.Zero(t, codes)
	require.NoError(t, db.Table("river_job").Where("kind = ?", PasswordChangedJobArgs{}.Kind()).Count(&notices).Error)
	require.Equal(t, int64(1), notices)
}

func TestAnOlderPasswordHashStillSignsIn(t *testing.T) {
	db, store, _, _ := liveVisibilityFixture(t, false)
	commander := domain.NewAuthCommander(store, domain.AuthOptions{OpenSignup: true})

	me, err := commander.Signup(t.Context(), domain.SignupParams{Email: "old@example.com", Password: "long-enough-password"})
	require.NoError(t, err)

	salt := []byte("0123456789abcdef")
	key := argon2.IDKey([]byte("short"), salt, 1, 32*1024, 1, 32)
	hash := fmt.Sprintf("argon2id$v=19$m=%d$t=%d$p=%d$%s$%s", 32*1024, 1, 1,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key))
	require.NoError(t, db.Model(&domain.User{}).Where("id = ?", me.User.ID).Update("password_hash", hash).Error)

	_, err = commander.Login(t.Context(), "old@example.com", "short")
	require.NoError(t, err)
}

func TestRefreshTokenReuseEndsTheConnectionAfterTheGracePeriod(t *testing.T) {
	db, store, ctx, _ := liveVisibilityFixture(t, false)
	service := domain.NewCredentialService(store)
	identity, _ := auth.FromContext(ctx)

	tokens, err := service.IssueTokens(ctx, identity.UserID, "client-1", "Agent", "koi:read", testAudience, uuid.Nil)
	require.NoError(t, err)

	rotated, err := service.RefreshTokens(ctx, tokens.RefreshToken, "client-1", testAudience)
	require.NoError(t, err)

	// A client that lost the response retries at once: it fails, but keeps its connection.
	_, err = service.RefreshTokens(ctx, tokens.RefreshToken, "client-1", testAudience)
	require.ErrorIs(t, err, domain.ErrInvalidCredentials)

	resolved, err := service.Resolve(ctx, rotated.AccessToken)
	require.NoError(t, err)
	require.NotNil(t, resolved)

	require.NoError(t, db.Model(&domain.Credential{}).
		Where("token_hash = ?", domain.HashCredential(tokens.RefreshToken)).
		Update("used_at", time.Now().Add(-time.Minute)).Error)

	_, err = service.RefreshTokens(ctx, tokens.RefreshToken, "client-1", testAudience)
	require.ErrorIs(t, err, domain.ErrInvalidCredentials)

	resolved, err = service.Resolve(ctx, rotated.AccessToken)
	require.NoError(t, err)
	require.Nil(t, resolved, "a replayed token means someone else may hold it")

	_, err = service.RefreshTokens(ctx, rotated.RefreshToken, "client-1", testAudience)
	require.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestRevokingOneTokenEndsItsConnection(t *testing.T) {
	_, store, ctx, _ := liveVisibilityFixture(t, false)
	service := domain.NewCredentialService(store)
	identity, _ := auth.FromContext(ctx)

	first, err := service.IssueTokens(ctx, identity.UserID, "client-1", "Agent", "koi:read", testAudience, uuid.Nil)
	require.NoError(t, err)

	other, err := service.IssueTokens(ctx, identity.UserID, "client-2", "Other", "koi:read", testAudience, uuid.Nil)
	require.NoError(t, err)

	require.NoError(t, service.RevokeToken(ctx, first.RefreshToken))
	require.NoError(t, service.RevokeToken(ctx, "unknown-token"))

	resolved, err := service.Resolve(ctx, first.AccessToken)
	require.NoError(t, err)
	require.Nil(t, resolved)

	resolved, err = service.Resolve(ctx, other.AccessToken)
	require.NoError(t, err)
	require.NotNil(t, resolved)
}

func TestCleanupRemovesExpiredResetsAndUnusedClients(t *testing.T) {
	db, store, ctx, _ := liveVisibilityFixture(t, false)
	identity, _ := auth.FromContext(ctx)
	repo := store.CredentialRepo()
	old := time.Now().Add(-31 * 24 * time.Hour)

	require.NoError(t, db.Create(&domain.PasswordResetToken{
		UserID:    identity.UserID,
		TokenHash: "expired",
		ExpiresAt: time.Now().Add(-time.Minute),
	}).Error)
	require.NoError(t, db.Create(&domain.PasswordResetToken{
		UserID:    identity.UserID,
		TokenHash: "live",
		ExpiresAt: time.Now().Add(time.Hour),
	}).Error)

	for id, createdAt := range map[string]time.Time{"unused": old, "connected": old, "recent": time.Now()} {
		require.NoError(t, db.Create(&domain.OAuthClient{
			ID:           id,
			RedirectURIs: []string{"https://client.example/callback"},
			Name:         id,
			CreatedAt:    createdAt,
		}).Error)
	}

	_, err := domain.NewCredentialService(store).IssueTokens(ctx, identity.UserID, "connected", "Agent", "koi:read", testAudience, uuid.Nil)
	require.NoError(t, err)

	require.NoError(t, repo.CleanupExpired(t.Context(), time.Now()))

	var resets []string
	require.NoError(t, db.Model(&domain.PasswordResetToken{}).Pluck("token_hash", &resets).Error)
	require.Equal(t, []string{"live"}, resets)

	var clients []string
	require.NoError(t, db.Model(&domain.OAuthClient{}).Order("id").Pluck("id", &clients).Error)
	require.Equal(t, []string{"connected", "recent"}, clients)
}

func TestClosedSignupAdmitsOnlyTheFirstUser(t *testing.T) {
	_, store, _, _ := liveVisibilityFixture(t, false)
	require.NoError(t, store.db.Exec("DELETE FROM users").Error)
	require.NoError(t, store.db.Exec("DELETE FROM memberships").Error)

	commander := domain.NewAuthCommander(store, domain.AuthOptions{OpenSignup: false})

	var wg sync.WaitGroup
	results := make([]error, 4)

	for i := range results {
		wg.Go(func() {
			_, results[i] = commander.Signup(t.Context(), domain.SignupParams{
				Email:    uuid.NewString() + "@example.com",
				Password: "long-enough-password",
			})
		})
	}
	wg.Wait()

	admitted := 0

	for _, err := range results {
		if err == nil {
			admitted++
		} else {
			require.ErrorIs(t, err, domain.ErrSignupClosed)
		}
	}

	require.Equal(t, 1, admitted)

	var users int64
	require.NoError(t, store.db.Model(&domain.User{}).Count(&users).Error)
	require.Equal(t, int64(1), users)
}

func TestLongPromptsFitTheUniqueIndex(t *testing.T) {
	_, store, ctx, brand := liveVisibilityFixture(t, false)
	commander := domain.NewPromptCommander(store)

	// Random multi-byte text does not compress, which is the worst case for an index entry.
	runes := []rune("àèìòùßçñœæあいうえお漢字仮名")
	var builder strings.Builder

	for range 4000 {
		builder.WriteRune(runes[rand.IntN(len(runes))])
	}

	long := builder.String()

	_, err := commander.Create(ctx, brand.ID, domain.CreatePromptParams{Text: long})
	require.NoError(t, err)

	// The index still treats a prompt written in another case as the same prompt.
	_, err = commander.Create(ctx, brand.ID, domain.CreatePromptParams{Text: strings.ToUpper(long)})
	require.Error(t, err)
}

func TestFirstLookAllowsOneBrand(t *testing.T) {
	db, store, ctx, brand := liveVisibilityFixture(t, false)
	params := domain.CreateBrandParams{Name: "Second", Domain: "second.example", Country: "US"}

	// Self-hosted instances have no First Look.
	_, err := domain.NewBrandCommander(store).Create(ctx, params)
	require.NoError(t, err)
	require.NoError(t, db.Where("name = ?", "Second").Delete(&domain.Brand{}).Error)

	store.WithCloudMode(true)
	_, err = domain.NewBrandCommander(store).Create(ctx, params)
	require.ErrorIs(t, err, domain.ErrBrandLimitReached)

	// A trial or subscription lifts the limit.
	activateFirstLookOrganization(t, db, brand.OrganizationID)
	_, err = domain.NewBrandCommander(store).Create(ctx, params)
	require.NoError(t, err)
}

func TestFirstLookConcurrentBrandsCreateOnlyOne(t *testing.T) {
	db, store, _, _ := liveVisibilityFixture(t, false)
	store.WithCloudMode(true)

	user := &domain.User{Email: uuid.NewString() + "@example.com", PasswordHash: "hash"}
	require.NoError(t, store.UserRepo().Create(t.Context(), user))

	organizationID := testOrganizationForUser(t, store, user.ID)
	ctx := testIdentity(t, store, user.ID)

	var wg sync.WaitGroup
	for i := range 4 {
		wg.Go(func() {
			_, _ = domain.NewBrandCommander(store).Create(ctx, domain.CreateBrandParams{
				Name:    fmt.Sprintf("Brand %d", i),
				Domain:  fmt.Sprintf("brand-%d.example", i),
				Country: "US",
			})
		})
	}
	wg.Wait()

	var brands int64
	require.NoError(t, db.Model(&domain.Brand{}).Where("organization_id = ?", organizationID).Count(&brands).Error)
	require.Equal(t, int64(1), brands)
}

func failDrafts(t *testing.T, store *Store, brandID uuid.UUID) {
	t.Helper()

	require.NoError(t, store.db.Model(&domain.GenerationRun{}).
		Where("brand_id = ?", brandID).Update("state", domain.GenerationFailed).Error)
}

func TestFirstLookDraftAttemptsAreCapped(t *testing.T) {
	_, store, ctx, brand := liveVisibilityFixture(t, true)
	store.WithCloudMode(true)

	commander := domain.NewGenerationCommander(store, true)
	key := "first-attempt"

	for i := range 3 {
		var runKey *string
		if i == 0 {
			runKey = &key
		}

		_, _, err := commander.Create(ctx, brand.ID, runKey, true)
		require.NoError(t, err)
		failDrafts(t, store, brand.ID)
	}

	_, _, err := commander.Create(ctx, brand.ID, nil, true)
	require.ErrorIs(t, err, domain.ErrFirstLookUnavailable)

	// A retry with the same key replays the earlier run instead of spending another attempt.
	run, replay, err := commander.Create(ctx, brand.ID, &key, true)
	require.NoError(t, err)
	require.True(t, replay)
	require.NotNil(t, run)
}

func TestFirstLookDraftsHaveADailyLimit(t *testing.T) {
	_, store, ctx, brand := liveVisibilityFixture(t, true)
	store.WithCloudMode(true).WithFirstLookDailyLimit(1)

	_, _, err := domain.NewGenerationCommander(store, true).Create(ctx, brand.ID, nil, false)
	require.NoError(t, err)

	otherCtx, other := firstLookBrand(t, store, "other.example")

	_, _, err = domain.NewGenerationCommander(store, true).Create(otherCtx, other.ID, nil, false)
	require.ErrorIs(t, err, domain.ErrFirstLookUnavailable)
}

func TestPaidOrganizationsDoNotConsumeFirstLookLimits(t *testing.T) {
	db, store, ctx, brand := liveVisibilityFixture(t, true)
	store.WithCloudMode(true).WithFirstLookDailyLimit(1)

	startVisibilityTracking(t, store, ctx, brand.ID)

	otherCtx, other := firstLookBrand(t, store, "other.example")
	require.NoError(t, store.VisibilityRepo().StartTracking(otherCtx, other.ID))
	require.Equal(t, domain.TrackingBlockedFirstLookPaused, blockedReason(t, store, otherCtx, other.ID))

	activateFirstLookOrganization(t, db, brand.OrganizationID)
	require.Empty(t, blockedReason(t, store, otherCtx, other.ID))
}

func TestFirstLookSiteIgnoresWWW(t *testing.T) {
	_, store, _, _ := liveVisibilityFixture(t, true)
	store.WithCloudMode(true)

	user := &domain.User{Email: uuid.NewString() + "@example.com", PasswordHash: "hash"}
	require.NoError(t, store.UserRepo().Create(t.Context(), user))
	otherCtx := testIdentity(t, store, user.ID)
	testOrganizationForUser(t, store, user.ID)

	created, err := domain.NewBrandCommander(store).Create(otherCtx, domain.CreateBrandParams{
		Name:    "Acme",
		Domain:  "www.acme.example",
		Country: "US",
	})
	require.NoError(t, err)
	require.Equal(t, "acme.example", created.Domain)

	_, err = domain.NewPromptCommander(store).Create(otherCtx, created.ID, domain.CreatePromptParams{Text: "Best tools?"})
	require.NoError(t, err)
	startVisibilityTracking(t, store, otherCtx, created.ID)

	for range domain.FirstLookSiteLimit - 1 {
		colleagueCtx, colleague := firstLookBrand(t, store, "acme.example")
		startVisibilityTracking(t, store, colleagueCtx, colleague.ID)
	}

	thirdCtx, third := firstLookBrand(t, store, "second.example")
	updated := "www.ACME.example"
	_, err = domain.NewBrandCommander(store).Update(thirdCtx, third.ID, domain.UpdateBrandParams{Domain: &updated})
	require.NoError(t, err)
	require.NoError(t, store.VisibilityRepo().StartTracking(thirdCtx, third.ID))
	require.Equal(t, domain.TrackingBlockedFirstLookSiteUsed, blockedReason(t, store, thirdCtx, third.ID))
}
