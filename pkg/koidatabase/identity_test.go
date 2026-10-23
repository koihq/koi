package koidatabase

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/koiauth"
	"github.com/koihq/koi/pkg/koidomain"
)

func identityCtx(t *testing.T, userID uuid.UUID) context.Context {
	t.Helper()

	return koiauth.WithIdentity(t.Context(), koiauth.Identity{UserID: userID, OrganizationID: userID})
}

func testIdentity(t *testing.T, store *Store, userID uuid.UUID) context.Context {
	t.Helper()
	identity, err := koidomain.ResolveIdentity(t.Context(), store, userID)
	if err == koidomain.ErrUnauthorized {
		testOrganizationForUser(t, store, userID)
		identity, err = koidomain.ResolveIdentity(t.Context(), store, userID)
	}
	require.NoError(t, err)
	return koiauth.WithIdentity(t.Context(), identity)
}
