package koidomain

import (
	"context"

	"github.com/google/uuid"
	"github.com/koihq/koi/pkg/koiauth"
)

type ProductAccessChecker interface {
	ProductAccess(context.Context, uuid.UUID) (bool, error)
}

func RequireProductAccess(ctx context.Context, checker ProductAccessChecker) error {
	identity, ok := koiauth.FromContext(ctx)
	if !ok || identity.OrganizationID == uuid.Nil {
		return ErrUnauthorized
	}
	allowed, err := checker.ProductAccess(ctx, identity.OrganizationID)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrBillingRequired
	}
	return nil
}
