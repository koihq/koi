package koidomain

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type memStore struct {
	users   map[uuid.UUID]*User
	orgs    map[uuid.UUID]*Organization
	members map[uuid.UUID]*Membership
}

func newMemStore() *memStore {
	return &memStore{
		users:   map[uuid.UUID]*User{},
		orgs:    map[uuid.UUID]*Organization{},
		members: map[uuid.UUID]*Membership{},
	}
}

func (s *memStore) Atomic(_ context.Context, fn func(Store) error) error { return fn(s) }
func (s *memStore) Ping(context.Context) error                           { return nil }
func (s *memStore) UserRepo() UserRepository                             { return memUsers{s} }
func (s *memStore) OrganizationRepo() OrganizationRepository             { return memOrgs{s} }
func (s *memStore) MembershipRepo() MembershipRepository                 { return memMembers{s} }

type memUsers struct{ s *memStore }
type memOrgs struct{ s *memStore }
type memMembers struct{ s *memStore }

func assignID(id *uuid.UUID) {
	if *id == uuid.Nil {
		*id = uuid.New()
	}
}

func (r memUsers) Create(_ context.Context, user *User) error {
	assignID(&user.ID)
	cp := *user
	r.s.users[user.ID] = &cp
	return nil
}

func (r memUsers) Get(_ context.Context, id uuid.UUID) (*User, error) {
	u, ok := r.s.users[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (r memUsers) GetByEmail(_ context.Context, email string) (*User, error) {
	for _, u := range r.s.users {
		if u.Email == email {
			cp := *u
			return &cp, nil
		}
	}
	return nil, ErrNotFound
}

func (r memOrgs) Create(_ context.Context, org *Organization) error {
	assignID(&org.ID)
	cp := *org
	r.s.orgs[org.ID] = &cp
	return nil
}

func (r memOrgs) Get(_ context.Context, id uuid.UUID) (*Organization, error) {
	o, ok := r.s.orgs[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *o
	return &cp, nil
}

func (r memMembers) Create(_ context.Context, m *Membership) error {
	assignID(&m.ID)
	cp := *m
	r.s.members[m.ID] = &cp
	return nil
}

func (r memMembers) Get(_ context.Context, id uuid.UUID) (*Membership, error) {
	m, ok := r.s.members[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *m
	return &cp, nil
}

func (r memMembers) GetByUserID(_ context.Context, userID uuid.UUID) (*Membership, error) {
	for _, m := range r.s.members {
		if m.UserID == userID {
			cp := *m
			return &cp, nil
		}
	}
	return nil, ErrNotFound
}

func TestSignup_CreatesUserOrgMembership(t *testing.T) {
	store := newMemStore()
	me, err := NewAuthCommander(store).Signup(t.Context(), SignupParams{
		Email: "You@Example.com", Password: "password1",
	})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, me.User.ID)
	require.NotEqual(t, uuid.Nil, me.Organization.ID)
	require.Equal(t, "you@example.com", me.User.Email)
	require.Equal(t, "you", me.Organization.Name)
	require.Equal(t, RoleOwner, me.Role)
	require.Len(t, store.users, 1)
	require.Len(t, store.orgs, 1)
	require.Len(t, store.members, 1)
}

func TestSignup_DuplicateEmail(t *testing.T) {
	cmd := NewAuthCommander(newMemStore())
	p := SignupParams{Email: "you@example.com", Password: "password1"}
	_, err := cmd.Signup(t.Context(), p)
	require.NoError(t, err)
	_, err = cmd.Signup(t.Context(), p)
	require.ErrorIs(t, err, ErrEmailTaken)
}

func TestLogin_WrongPasswordAndUnknownEmail(t *testing.T) {
	cmd := NewAuthCommander(newMemStore())
	_, err := cmd.Signup(t.Context(), SignupParams{Email: "you@example.com", Password: "password1"})
	require.NoError(t, err)

	_, err = cmd.Login(t.Context(), "you@example.com", "wrongpass")
	require.ErrorIs(t, err, ErrInvalidCredentials)

	_, err = cmd.Login(t.Context(), "nobody@example.com", "password1")
	require.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestSignup_Validate(t *testing.T) {
	cmd := NewAuthCommander(newMemStore())
	_, err := cmd.Signup(t.Context(), SignupParams{Email: "not-an-email", Password: "password1"})
	require.ErrorIs(t, err, ErrValidation)

	_, err = cmd.Signup(t.Context(), SignupParams{Email: "you@example.com", Password: "short"})
	require.ErrorIs(t, err, ErrValidation)
}

func TestUserValidate(t *testing.T) {
	require.ErrorIs(t, User{Email: "bad", PasswordHash: "x"}.Validate(), ErrValidation)
	require.ErrorIs(t, User{Email: "you@example.com"}.Validate(), ErrValidation)
	require.NoError(t, User{Email: "you@example.com", PasswordHash: "x"}.Validate())
}

func TestCheckPassword(t *testing.T) {
	hash, err := HashPassword("password1")
	require.NoError(t, err)
	require.True(t, CheckPassword(hash, "password1"))
	require.False(t, CheckPassword(hash, "password2"))
}

func TestLogin_Me(t *testing.T) {
	cmd := NewAuthCommander(newMemStore())
	_, err := cmd.Signup(t.Context(), SignupParams{Email: "you@example.com", Password: "password1"})
	require.NoError(t, err)
	me, err := cmd.Login(t.Context(), "you@example.com", "password1")
	require.NoError(t, err)
	require.Equal(t, "you@example.com", me.User.Email)
	require.Equal(t, "you", me.Organization.Name)
}
