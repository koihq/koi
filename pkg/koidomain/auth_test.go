package koidomain

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/koihq/koi/pkg/koiauth"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type authDeps struct {
	store   *MockStore
	users   *MockUserRepository
	orgs    *MockOrganizationRepository
	members *MockMembershipRepository
}

func newAuthDeps(t *testing.T) authDeps {
	t.Helper()

	d := authDeps{
		store:   NewMockStore(t),
		users:   NewMockUserRepository(t),
		orgs:    NewMockOrganizationRepository(t),
		members: NewMockMembershipRepository(t),
	}

	d.store.On("UserRepo").Return(d.users)
	d.store.On("OrganizationRepo").Return(d.orgs).Maybe()
	d.store.On("MembershipRepo").Return(d.members).Maybe()

	return d
}

func (d authDeps) mockAtomic() {
	d.store.EXPECT().Atomic(mock.Anything, mock.Anything).RunAndReturn(
		func(_ context.Context, fn func(Store) error) error { return fn(d.store) },
	)
}

func assignUserID(_ context.Context, u *User) error {
	u.ID = uuid.New()

	return nil
}

func assignOrgID(_ context.Context, o *Organization) error {
	o.ID = uuid.New()

	return nil
}

func TestSignup_CreatesUserOrgMembership(t *testing.T) {
	d := newAuthDeps(t)
	d.mockAtomic()
	d.users.EXPECT().Create(mock.Anything, mock.Anything).RunAndReturn(assignUserID)
	d.orgs.EXPECT().Create(mock.Anything, mock.Anything).RunAndReturn(assignOrgID)
	d.members.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)

	me, err := NewAuthCommander(d.store).Signup(t.Context(), SignupParams{
		Email:    "You@Example.com",
		Password: "password1",
	})

	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, me.User.ID)
	require.NotEqual(t, uuid.Nil, me.Organization.ID)
	require.Equal(t, "you@example.com", me.User.Email)
	require.Equal(t, "you", me.Organization.Name)
	require.Equal(t, RoleOwner, me.Role)
}

func TestSignup_DuplicateEmail(t *testing.T) {
	d := newAuthDeps(t)
	d.mockAtomic()
	d.users.EXPECT().Create(mock.Anything, mock.Anything).Return(ErrEmailTaken)

	_, err := NewAuthCommander(d.store).Signup(t.Context(), SignupParams{
		Email:    "you@example.com",
		Password: "password1",
	})

	require.ErrorIs(t, err, ErrEmailTaken)
}

func TestLogin_WrongPasswordAndUnknownEmail(t *testing.T) {
	hash, err := HashPassword("password1")
	require.NoError(t, err)

	user := &User{
		BaseEntity: BaseEntity{
			ID: uuid.New(),
		},
		Email:        "you@example.com",
		PasswordHash: hash,
	}

	wrong := newAuthDeps(t)
	wrong.users.EXPECT().GetByEmail(mock.Anything, "you@example.com").Return(user, nil)
	_, err = NewAuthCommander(wrong.store).Login(t.Context(), "you@example.com", "wrongpass")
	require.ErrorIs(t, err, ErrInvalidCredentials)

	unknown := newAuthDeps(t)
	unknown.users.EXPECT().GetByEmail(mock.Anything, "nobody@example.com").Return(nil, ErrNotFound)
	_, err = NewAuthCommander(unknown.store).Login(t.Context(), "nobody@example.com", "password1")
	require.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestSignup_Validate(t *testing.T) {
	cmd := NewAuthCommander(NewMockStore(t))
	_, err := cmd.Signup(t.Context(), SignupParams{Email: "not-an-email", Password: "password1"})
	require.ErrorIs(t, err, ErrValidation)
	var ve *Error
	require.ErrorAs(t, err, &ve)
	require.Equal(t, []FieldError{
		{
			Field: "email",
			Code:  FieldInvalid,
		},
	}, ve.Fields)

	_, err = cmd.Signup(t.Context(), SignupParams{Email: "you@example.com", Password: "short"})
	require.ErrorIs(t, err, ErrValidation)
	require.ErrorAs(t, err, &ve)
	require.Equal(t, []FieldError{
		{
			Field: "password",
			Code:  FieldMinLength,
		},
	}, ve.Fields)
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
	hash, err := HashPassword("password1")
	require.NoError(t, err)
	userID := uuid.New()
	orgID := uuid.New()
	user := &User{
		BaseEntity: BaseEntity{
			ID: userID,
		},
		Email:        "you@example.com",
		PasswordHash: hash,
	}
	org := &Organization{
		BaseEntity: BaseEntity{
			ID: orgID,
		},
		Name: "you",
	}
	member := &Membership{UserID: userID, OrganizationID: orgID, Role: RoleOwner}

	d := newAuthDeps(t)
	d.users.EXPECT().GetByEmail(mock.Anything, "you@example.com").Return(user, nil)
	d.members.EXPECT().GetByUserID(mock.Anything, userID).Return(member, nil)
	d.orgs.EXPECT().Get(mock.Anything, orgID).Return(org, nil)

	me, err := NewAuthCommander(d.store).Login(t.Context(), "you@example.com", "password1")
	require.NoError(t, err)
	require.Equal(t, "you@example.com", me.User.Email)
	require.Equal(t, "you", me.Organization.Name)
}

func TestAuthQuerier_MeNotFound(t *testing.T) {
	d := newAuthDeps(t)
	id := uuid.New()
	d.users.EXPECT().Get(mock.Anything, id).Return(nil, ErrNotFound)

	_, err := NewAuthQuerier(d.store).Me(koiauth.WithIdentity(t.Context(), koiauth.Identity{UserID: id, OrganizationID: uuid.New(), Role: RoleOwner}))
	require.ErrorIs(t, err, ErrNotFound)
}

func TestAuthQuerierRequiresTenantIdentity(t *testing.T) {
	querier := NewAuthQuerier(NewMockStore(t))
	for _, ctx := range []context.Context{
		t.Context(),
		koiauth.WithIdentity(t.Context(), koiauth.Identity{UserID: uuid.New()}),
	} {
		_, err := querier.Me(ctx)
		require.ErrorIs(t, err, ErrUnauthorized)
	}
}
