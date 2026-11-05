package database

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/internal/testdb"
	"github.com/koihq/koi/pkg/domain"
)

func TestCompetitorUniqueDomainConcurrent(t *testing.T) {
	db, err := Open(testdb.URL(t))
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	require.NoError(t, Migrate(db))
	store := NewStore(db)
	user := &domain.User{Email: uuid.NewString() + "@example.com", PasswordHash: "hash"}
	require.NoError(t, store.UserRepo().Create(t.Context(), user))
	organization := &domain.Organization{Name: "Koi"}
	require.NoError(t, store.OrganizationRepo().Create(t.Context(), organization))
	require.NoError(t, store.MembershipRepo().Create(t.Context(), &domain.Membership{
		UserID: user.ID, OrganizationID: organization.ID, Role: domain.RoleOwner,
	}))
	ctx := testIdentity(t, store, user.ID)
	brand := &domain.Brand{
		OrganizationID: organization.ID,
		Name:           "Koi",
		Domain:         "koi.dev",
		Country:        "IT",
	}
	require.NoError(t, store.BrandRepo().Create(ctx, brand))
	require.Equal(t, organization.ID, brand.OrganizationID)

	errors := make(chan error, 2)

	for range 2 {
		go func() {
			_, err := domain.NewCompetitorCommander(store).Create(ctx, brand.ID, domain.CreateCompetitorParams{
				Name:   "Acme",
				Domain: "ACME.COM",
			})

			errors <- err
		}()
	}

	var succeeded, taken int

	for range 2 {
		err := <-errors
		if err == nil {
			succeeded++
			continue
		}

		var domainErr *domain.Error
		require.ErrorAs(t, err, &domainErr)
		require.Equal(t, domain.CodeValidation, domainErr.Code)
		require.Equal(t, []domain.FieldError{
			{
				Field: "domain",
				Code:  domain.FieldTaken,
			},
		}, domainErr.Fields)
		taken++
	}

	require.Equal(t, 1, succeeded)
	require.Equal(t, 1, taken)

	page, err := store.CompetitorRepo().List(ctx, domain.CompetitorFilter{
		BrandFilter: domain.BrandFilter{BrandID: brand.ID},
	}, domain.PageParams{
		Page:     1,
		PageSize: 10,
	})

	require.NoError(t, err)
	require.EqualValues(t, 1, page.TotalItems)

	for _, repo := range []string{"https://github.com/zsh-users/zsh-autosuggestions", "https://github.com/atuinsh/atuin"} {
		_, err := domain.NewCompetitorCommander(store).Create(ctx, brand.ID, domain.CreateCompetitorParams{Name: repo, GitHubRepo: repo})
		require.NoError(t, err)
	}

	_, err = domain.NewCompetitorCommander(store).Create(ctx, brand.ID, domain.CreateCompetitorParams{
		Name:       "Duplicate",
		GitHubRepo: "https://github.com/zsh-users/zsh-autosuggestions",
	})
	var repoErr *domain.Error
	require.ErrorAs(t, err, &repoErr)
	require.Equal(t, []domain.FieldError{
		{
			Field: "github_repo",
			Code:  domain.FieldTaken,
		},
	}, repoErr.Fields)

	otherUser := &domain.User{Email: uuid.NewString() + "@example.com", PasswordHash: "hash"}
	require.NoError(t, store.UserRepo().Create(t.Context(), otherUser))
	otherOrganizationID := testOrganizationForUser(t, store, otherUser.ID)
	otherCtx := testIdentity(t, store, otherUser.ID)
	otherBrand := &domain.Brand{OrganizationID: otherOrganizationID, Name: "Other", Domain: "other.dev", Country: "IT"}
	require.NoError(t, store.BrandRepo().Create(otherCtx, otherBrand))

	_, err = domain.NewCompetitorCommander(store).Create(otherCtx, otherBrand.ID, domain.CreateCompetitorParams{
		Name:   "Acme",
		Domain: "acme.com",
	})

	require.NoError(t, err)
	_, err = store.CompetitorRepo().Get(otherCtx, page.Items[0].ID)
	require.ErrorIs(t, err, domain.ErrNotFound)
	_, err = store.BrandRepo().Get(otherCtx, brand.ID)
	require.ErrorIs(t, err, domain.ErrNotFound)
	require.ErrorIs(t, store.BrandRepo().Update(otherCtx, brand), domain.ErrNotFound)
	competitorID := page.Items[0].ID
	require.ErrorIs(t, store.CompetitorRepo().Update(otherCtx, &page.Items[0]), domain.ErrNotFound)
	require.ErrorIs(t, store.CompetitorRepo().Delete(otherCtx, competitorID), domain.ErrNotFound)
	require.ErrorIs(t, store.CompetitorRepo().Create(otherCtx, &domain.Competitor{BrandID: brand.ID}), domain.ErrNotFound)

	prompt := &domain.Prompt{
		BrandID:   brand.ID,
		Text:      "Who is Koi?",
		Source:    domain.PromptSourceManual,
		Confirmed: true,
	}
	require.NoError(t, store.PromptRepo().Create(ctx, prompt))
	promptID := prompt.ID
	_, err = store.PromptRepo().Get(otherCtx, promptID)
	require.ErrorIs(t, err, domain.ErrNotFound)
	require.ErrorIs(t, store.PromptRepo().Update(otherCtx, prompt), domain.ErrNotFound)
	require.ErrorIs(t, store.PromptRepo().Delete(otherCtx, promptID), domain.ErrNotFound)
	require.ErrorIs(t, store.PromptRepo().Create(otherCtx, &domain.Prompt{BrandID: brand.ID}), domain.ErrNotFound)
}
