package domain

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/koihq/koi/pkg/auth"
)

func identityCtx(t *testing.T, userID uuid.UUID) context.Context {
	t.Helper()

	return auth.WithIdentity(t.Context(), auth.Identity{UserID: userID})
}
