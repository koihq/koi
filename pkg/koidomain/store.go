package koidomain

import "context"

type Store interface {
	Atomic(context.Context, func(Store) error) error
	Ping(context.Context) error
	UserRepo() UserRepository
	OrganizationRepo() OrganizationRepository
	MembershipRepo() MembershipRepository
}
