package koidatabase

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/koihq/koi/pkg/koiauth"
)

func identityCtx(t *testing.T, userID uuid.UUID) context.Context {
	t.Helper()
	return koiauth.WithIdentity(t.Context(), koiauth.Identity{UserID: userID})
}
