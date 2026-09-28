package koidomain

import "context"

//go:generate go tool mockery

type Store interface {
	Atomic(context.Context, func(Store) error) error
	Ping(context.Context) error
	UserRepo() UserRepository
	OrganizationRepo() OrganizationRepository
	MembershipRepo() MembershipRepository
	BrandRepo() BrandRepository
	PromptRepo() PromptRepository
	CompetitorRepo() CompetitorRepository
}
