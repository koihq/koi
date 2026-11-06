package database

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/auth"
	"github.com/koihq/koi/pkg/domain"
)

func identityCtx(t *testing.T, userID uuid.UUID) context.Context {
	t.Helper()

	return auth.WithIdentity(t.Context(), auth.Identity{UserID: userID, OrganizationID: userID})
}

func testIdentity(t *testing.T, store *Store, userID uuid.UUID) context.Context {
	t.Helper()
	identity, err := domain.ResolveIdentity(t.Context(), store, userID)
	if err == domain.ErrUnauthorized {
		testOrganizationForUser(t, store, userID)
		identity, err = domain.ResolveIdentity(t.Context(), store, userID)
	}
	require.NoError(t, err)
	return auth.WithIdentity(t.Context(), identity)
}
