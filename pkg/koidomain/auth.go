package koidomain

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"github.com/koihq/koi/pkg/koiauth"
)

const minPasswordLen = 8

type SignupParams struct {
	Email    string
	Password string
}

func (p SignupParams) Validate() error {
	if !validEmail(p.Email) {
		return Validation(FieldError{Field: "email", Code: FieldInvalid})
	}

	if len(p.Password) < minPasswordLen {
		return Validation(FieldError{Field: "password", Code: FieldMinLength})
	}

	return nil
}

type Me struct {
	User         User
	Organization Organization
	Role         string
	ActiveBrand  *Brand
}

type AuthCommander interface {
	Signup(ctx context.Context, p SignupParams) (*Me, error)
	Login(ctx context.Context, email, password string) (*Me, error)
}

type AuthQuerier interface {
	Me(ctx context.Context, preferredBrandID uuid.UUID) (*Me, error)
}

type authCommander struct {
	store Store
}

func NewAuthCommander(store Store) AuthCommander {
	return &authCommander{store: store}
}

type authQuerier struct {
	store Store
}

func NewAuthQuerier(store Store) AuthQuerier {
	return &authQuerier{store: store}
}

func (c *authCommander) Signup(ctx context.Context, p SignupParams) (*Me, error) {
	p.Email = strings.ToLower(strings.TrimSpace(p.Email))
	if err := p.Validate(); err != nil {
		return nil, err
	}

	hash, err := HashPassword(p.Password)
	if err != nil {
		return nil, err
	}

	var me *Me

	err = c.store.Atomic(ctx, func(s Store) error {
		local, _, _ := strings.Cut(p.Email, "@")
		user := &User{Email: p.Email, PasswordHash: hash}
		if err := user.Validate(); err != nil {
			return err
		}

		if err := s.UserRepo().Create(ctx, user); err != nil {
			return err
		}

		org := &Organization{Name: local}
		if err := org.Validate(); err != nil {
			return err
		}

		if err := s.OrganizationRepo().Create(ctx, org); err != nil {
			return err
		}

		m := &Membership{UserID: user.ID, OrganizationID: org.ID, Role: RoleOwner}
		if err := m.Validate(); err != nil {
			return err
		}

		if err := s.MembershipRepo().Create(ctx, m); err != nil {
			return err
		}

		me = &Me{User: *user, Organization: *org, Role: m.Role}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if err := c.store.EnqueueWelcomeEmail(ctx, me.User.ID); err != nil {
		slog.Warn("welcome email enqueue failed", "user_id", me.User.ID, "err", err)
	}

	return me, nil
}

func (c *authCommander) Login(ctx context.Context, email, password string) (*Me, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	user, err := c.store.UserRepo().GetByEmail(ctx, email)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	hash := dummyPasswordHash
	if user != nil {
		hash = user.PasswordHash
	}

	if !CheckPassword(hash, password) || user == nil {
		return nil, ErrInvalidCredentials
	}

	return loadMe(ctx, c.store, user)
}

func (q *authQuerier) Me(ctx context.Context, preferredBrandID uuid.UUID) (*Me, error) {
	identity, ok := koiauth.FromContext(ctx)
	if !ok || identity.OrganizationID == uuid.Nil {
		return nil, ErrUnauthorized
	}

	user, err := q.store.UserRepo().Get(ctx, identity.UserID)
	if err != nil {
		return nil, err
	}

	organization, err := q.store.OrganizationRepo().Get(ctx, identity.OrganizationID)
	if err != nil {
		return nil, err
	}

	var brand *Brand
	if preferredBrandID != uuid.Nil {
		brand, err = q.store.BrandRepo().Get(ctx, preferredBrandID)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return nil, err
		}
	}
	if brand == nil {
		brand, err = q.store.BrandRepo().Latest(ctx)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return nil, err
		}
	}

	return &Me{
		User:         *user,
		Organization: *organization,
		Role:         identity.Role,
		ActiveBrand:  brand,
	}, nil
}

func loadMe(ctx context.Context, store Store, user *User) (*Me, error) {
	m, err := store.MembershipRepo().GetByUserID(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	org, err := store.OrganizationRepo().Get(ctx, m.OrganizationID)
	if err != nil {
		return nil, err
	}

	return &Me{User: *user, Organization: *org, Role: m.Role}, nil
}

// ResolveIdentity verifies the current membership for each authenticated request.
func ResolveIdentity(ctx context.Context, store Store, userID uuid.UUID) (koiauth.Identity, error) {
	membership, err := store.MembershipRepo().GetByUserID(ctx, userID)
	if errors.Is(err, ErrNotFound) {
		return koiauth.Identity{}, ErrUnauthorized
	}
	if err != nil {
		return koiauth.Identity{}, err
	}
	return koiauth.Identity{UserID: userID, OrganizationID: membership.OrganizationID, Role: membership.Role}, nil
}
