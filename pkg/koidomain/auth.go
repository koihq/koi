package koidomain

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

const minPasswordLen = 8

type SignupParams struct {
	Email    string
	Password string
}

func (p SignupParams) Validate() error {
	if !validEmail(p.Email) {
		return fmt.Errorf("%w: email", ErrValidation)
	}
	if len(p.Password) < minPasswordLen {
		return fmt.Errorf("%w: password", ErrValidation)
	}
	return nil
}

type Me struct {
	User         User
	Organization Organization
	Role         string
}

type AuthCommander struct {
	store Store
}

func NewAuthCommander(store Store) *AuthCommander {
	return &AuthCommander{store: store}
}

type AuthQuerier struct {
	store Store
}

func NewAuthQuerier(store Store) *AuthQuerier {
	return &AuthQuerier{store: store}
}

func (c *AuthCommander) Signup(ctx context.Context, p SignupParams) (*Me, error) {
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
		_, err := s.UserRepo().GetByEmail(ctx, p.Email)
		if err == nil {
			return ErrEmailTaken
		}
		if !errors.Is(err, ErrNotFound) {
			return err
		}
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
	return me, nil
}

func (c *AuthCommander) Login(ctx context.Context, email, password string) (*Me, error) {
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
	return NewAuthQuerier(c.store).Me(ctx, user.ID)
}

func (q *AuthQuerier) Me(ctx context.Context, userID uuid.UUID) (*Me, error) {
	user, err := q.store.UserRepo().Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	m, err := q.store.MembershipRepo().GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	org, err := q.store.OrganizationRepo().Get(ctx, m.OrganizationID)
	if err != nil {
		return nil, err
	}
	return &Me{User: *user, Organization: *org, Role: m.Role}, nil
}
