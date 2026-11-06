package domain

import (
	"context"

	"github.com/google/uuid"
)

//go:generate go tool mockery

type Store interface {
	Atomic(context.Context, func(Store) error) error
	Ping(context.Context) error
	EnqueueWelcomeEmail(context.Context, uuid.UUID) error
	EnqueuePasswordResetEmail(ctx context.Context, email string) error
	UserRepo() UserRepository
	OrganizationRepo() OrganizationRepository
	MembershipRepo() MembershipRepository
	BillingRepo() BillingRepository
	CredentialRepo() CredentialRepository
	PasswordResetRepo() PasswordResetRepository
	BrandRepo() BrandRepository
	PromptRepo() PromptRepository
	CompetitorRepo() CompetitorRepository
	GenerationRepo() GenerationRepository
	VisibilityRepo() VisibilityRepository
}
