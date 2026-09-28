package koiauth

import (
	"context"

	"github.com/google/uuid"
)

type Identity struct {
	UserID uuid.UUID
}

type identityKey struct{}

func WithIdentity(ctx context.Context, identity Identity) context.Context {
	return context.WithValue(ctx, identityKey{}, identity)
}

func FromContext(ctx context.Context) (Identity, bool) {
	identity, ok := ctx.Value(identityKey{}).(Identity)
	return identity, ok && identity.UserID != uuid.Nil
}
