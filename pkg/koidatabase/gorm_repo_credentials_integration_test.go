package koidatabase

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/internal/testdb"
	"github.com/koihq/koi/pkg/koiauth"
	"github.com/koihq/koi/pkg/koidomain"
)

func TestOAuthCredentialsRotateRevokeAndIsolate(t *testing.T) {
	db, err := Open(testdb.URL(t))
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	require.NoError(t, Migrate(db))
	store := NewStore(db)
	service := koidomain.NewCredentialService(store)
	first, err := koidomain.NewAuthCommander(store).Signup(t.Context(), koidomain.SignupParams{Email: uuid.NewString() + "@example.com", Password: "password123"})
	require.NoError(t, err)
	second, err := koidomain.NewAuthCommander(store).Signup(t.Context(), koidomain.SignupParams{Email: uuid.NewString() + "@example.com", Password: "password123"})
	require.NoError(t, err)
	identity := koiauth.Identity{UserID: first.User.ID, OrganizationID: first.Organization.ID, Role: first.Role}
	ctx := koiauth.WithIdentity(t.Context(), identity)
	firstTokens, err := service.IssueTokens(ctx, first.User.ID, "client-1", "Test client", "koi:read", "https://koi.example/mcp", uuid.Nil)
	require.NoError(t, err)
	refreshed, err := service.RefreshTokens(ctx, firstTokens.RefreshToken, "client-1", "https://koi.example/mcp")
	require.NoError(t, err)
	_, err = service.RefreshTokens(ctx, firstTokens.RefreshToken, "client-1", "https://koi.example/mcp")
	require.ErrorIs(t, err, koidomain.ErrInvalidCredentials)
	page, err := service.List(ctx, koidomain.PageParams{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	foreignCtx := koiauth.WithIdentity(t.Context(), koiauth.Identity{UserID: second.User.ID, OrganizationID: second.Organization.ID, Role: second.Role})
	require.ErrorIs(t, service.Revoke(foreignCtx, page.Items[0].ID), koidomain.ErrNotFound)
	require.NoError(t, service.Revoke(ctx, page.Items[0].ID))
	for _, token := range []string{firstTokens.AccessToken, refreshed.AccessToken, refreshed.RefreshToken} {
		credential, err := service.Resolve(ctx, token)
		require.NoError(t, err)
		require.Nil(t, credential)
	}
}
