package database

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/koihq/koi/internal/testdb"
	"github.com/koihq/koi/pkg/auth"
	"github.com/koihq/koi/pkg/domain"
)

type repositoryFixture struct {
	db       *gorm.DB
	ctx      context.Context
	otherCtx context.Context
	brand    *domain.Brand
}

func newRepositoryFixture(t *testing.T) repositoryFixture {
	t.Helper()

	db, err := Open(testdb.URL(t))
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

	require.NoError(t, Migrate(db))

	store := NewStore(db)
	owner := &domain.User{Email: uuid.NewString() + "@example.com", PasswordHash: "hash"}
	require.NoError(t, store.UserRepo().Create(t.Context(), owner))

	stranger := &domain.User{Email: uuid.NewString() + "@example.com", PasswordHash: "hash"}
	require.NoError(t, store.UserRepo().Create(t.Context(), stranger))

	organizationID := testOrganizationForUser(t, store, owner.ID)
	ctx := testIdentity(t, store, owner.ID)
	brand := &domain.Brand{
		OrganizationID: organizationID,
		Name:           "Koi",
		Domain:         "koi.dev",
		Country:        "IT",
	}
	require.NoError(t, store.BrandRepo().Create(ctx, brand))

	return repositoryFixture{
		db:       db,
		ctx:      ctx,
		otherCtx: testIdentity(t, store, stranger.ID),
		brand:    brand,
	}
}

func newPromptFixture(brandID uuid.UUID, text string) *domain.Prompt {
	return &domain.Prompt{
		BrandID:   brandID,
		Text:      text,
		Source:    domain.PromptSourceDrafted,
		Confirmed: true,
	}
}

func TestRepositoryCreateIsScopedToTheBrandOrganization(t *testing.T) {
	f := newRepositoryFixture(t)
	repo := newBrandChildRepository[domain.Prompt](f.db)

	foreign := newPromptFixture(f.brand.ID, "Foreign")
	require.ErrorIs(t, repo.Create(f.otherCtx, foreign), domain.ErrNotFound)
	require.Equal(t, uuid.Nil, foreign.ID)

	anonymous := newPromptFixture(f.brand.ID, "Anonymous")
	require.ErrorIs(t, repo.Create(t.Context(), anonymous), domain.ErrUnauthorized)
	require.Equal(t, uuid.Nil, anonymous.ID)

	owned := newPromptFixture(f.brand.ID, "Owned")
	require.NoError(t, repo.Create(f.ctx, owned))
	require.NotEqual(t, uuid.Nil, owned.ID)
}

func TestScopedRepositoryWithoutCreateAuthorizerRejectsCreate(t *testing.T) {
	f := newRepositoryFixture(t)
	repo := newScopedRepository(f.db, tenantScope[domain.Prompt]{
		read: func(_ auth.Identity, db *gorm.DB) *gorm.DB { return db },
	})

	require.ErrorIs(t, repo.Create(f.ctx, newPromptFixture(f.brand.ID, "Rejected")), domain.ErrUnauthorized)
}

func TestRepositoryReadsAndMutationsAreScoped(t *testing.T) {
	f := newRepositoryFixture(t)
	repo := newBrandChildRepository[domain.Prompt](f.db, "text")

	prompt := newPromptFixture(f.brand.ID, "Original")
	require.NoError(t, repo.Create(f.ctx, prompt))

	_, err := repo.Get(f.otherCtx, prompt.ID)
	require.ErrorIs(t, err, domain.ErrNotFound)

	prompt.Text = "Hijacked"
	require.ErrorIs(t, repo.Update(f.otherCtx, prompt), domain.ErrNotFound)
	require.ErrorIs(t, repo.Delete(f.otherCtx, prompt.ID), domain.ErrNotFound)

	stored, err := repo.Get(f.ctx, prompt.ID)
	require.NoError(t, err)
	require.Equal(t, "Original", stored.Text)

	_, err = repo.Get(t.Context(), prompt.ID)
	require.ErrorIs(t, err, domain.ErrUnauthorized)
	require.ErrorIs(t, repo.Update(t.Context(), prompt), domain.ErrUnauthorized)
	require.ErrorIs(t, repo.Delete(t.Context(), prompt.ID), domain.ErrUnauthorized)
}

func TestRepositoryUpdateWritesOnlyDeclaredColumns(t *testing.T) {
	f := newRepositoryFixture(t)
	repo := newBrandChildRepository[domain.Prompt](f.db, "text", "confirmed")

	prompt := newPromptFixture(f.brand.ID, "Original")
	require.NoError(t, repo.Create(f.ctx, prompt))

	prompt.Text = "Changed"
	prompt.Confirmed = false
	prompt.Source = domain.PromptSourceManual
	prompt.Inactive = true
	require.NoError(t, repo.Update(f.ctx, prompt))

	stored, err := repo.Get(f.ctx, prompt.ID)
	require.NoError(t, err)
	require.Equal(t, "Changed", stored.Text)
	require.False(t, stored.Confirmed, "a zero value in a declared column must be written")
	require.Equal(t, domain.PromptSourceDrafted, stored.Source)
	require.False(t, stored.Inactive)
}

func TestRepositoryUpdateAndDeleteMissingRow(t *testing.T) {
	f := newRepositoryFixture(t)
	repo := newBrandChildRepository[domain.Prompt](f.db, "text")

	missing := newPromptFixture(f.brand.ID, "Missing")
	missing.ID = uuid.New()
	require.ErrorIs(t, repo.Update(f.ctx, missing), domain.ErrNotFound)
	require.ErrorIs(t, repo.Delete(f.ctx, missing.ID), domain.ErrNotFound)
}

func TestListRepositoryFiltersByTenantAndKeepsEmptyPageShape(t *testing.T) {
	f := newRepositoryFixture(t)
	repo := newListRepository(
		newBrandChildRepository[domain.Prompt](f.db),
		filterByBrand,
		fixedOrder[domain.BrandFilter]("created_at ASC, id ASC"),
	)
	page := domain.PageParams{Page: 1, PageSize: 10}
	filter := domain.BrandFilter{BrandID: f.brand.ID}

	empty, err := repo.List(f.ctx, filter, page)
	require.NoError(t, err)
	require.Equal(t, &domain.Page[domain.Prompt]{
		Items:       []domain.Prompt{},
		CurrentPage: 1,
	}, empty)

	require.NoError(t, repo.Create(f.ctx, newPromptFixture(f.brand.ID, "First")))

	owned, err := repo.List(f.ctx, filter, page)
	require.NoError(t, err)
	require.Len(t, owned.Items, 1)

	foreign, err := repo.List(f.otherCtx, filter, page)
	require.NoError(t, err)
	require.Empty(t, foreign.Items)
	require.Zero(t, foreign.TotalItems)

	_, err = repo.List(t.Context(), filter, page)
	require.ErrorIs(t, err, domain.ErrUnauthorized)
}
