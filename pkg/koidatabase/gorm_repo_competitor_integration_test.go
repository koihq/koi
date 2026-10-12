package koidatabase

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/internal/testdb"
	"github.com/koihq/koi/pkg/koiauth"
	"github.com/koihq/koi/pkg/koidomain"
)

func TestCompetitorUniqueDomainConcurrent(t *testing.T) {
	db, err := Open(testdb.URL(t))
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	require.NoError(t, Migrate(db))
	store := NewStore(db)
	user := &koidomain.User{Email: uuid.NewString() + "@example.com", PasswordHash: "hash"}
	require.NoError(t, store.UserRepo().Create(t.Context(), user))
	ctx := koiauth.WithIdentity(t.Context(), koiauth.Identity{UserID: user.ID})
	brand := &koidomain.Brand{
		UserID:  uuid.New(),
		Name:    "Koi",
		Domain:  "koi.dev",
		Country: "IT",
	}
	require.NoError(t, store.BrandRepo().Create(ctx, brand))
	require.Equal(t, user.ID, brand.UserID)

	errors := make(chan error, 2)

	for range 2 {
		go func() {
			_, err := koidomain.NewCompetitorCommander(store).Create(ctx, brand.ID, koidomain.CompetitorParams{
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

		var domainErr *koidomain.Error
		require.ErrorAs(t, err, &domainErr)
		require.Equal(t, koidomain.CodeValidation, domainErr.Code)
		require.Equal(t, []koidomain.FieldError{
			{
				Field: "domain",
				Code:  koidomain.FieldTaken,
			},
		}, domainErr.Fields)
		taken++
	}

	require.Equal(t, 1, succeeded)
	require.Equal(t, 1, taken)

	page, err := store.CompetitorRepo().List(ctx, koidomain.CompetitorFilter{
		BrandID: brand.ID,
		Page: koidomain.PageParams{
			Page:     1,
			PageSize: 10,
		},
	})

	require.NoError(t, err)
	require.EqualValues(t, 1, page.TotalItems)

	for _, repo := range []string{"https://github.com/zsh-users/zsh-autosuggestions", "https://github.com/atuinsh/atuin"} {
		_, err := koidomain.NewCompetitorCommander(store).Create(ctx, brand.ID, koidomain.CompetitorParams{Name: repo, GitHubRepo: repo})
		require.NoError(t, err)
	}

	_, err = koidomain.NewCompetitorCommander(store).Create(ctx, brand.ID, koidomain.CompetitorParams{
		Name:       "Duplicate",
		GitHubRepo: "https://github.com/zsh-users/zsh-autosuggestions",
	})
	var repoErr *koidomain.Error
	require.ErrorAs(t, err, &repoErr)
	require.Equal(t, []koidomain.FieldError{
		{
			Field: "github_repo",
			Code:  koidomain.FieldTaken,
		},
	}, repoErr.Fields)

	otherUser := &koidomain.User{Email: uuid.NewString() + "@example.com", PasswordHash: "hash"}
	require.NoError(t, store.UserRepo().Create(t.Context(), otherUser))
	otherCtx := koiauth.WithIdentity(t.Context(), koiauth.Identity{UserID: otherUser.ID})
	otherBrand := &koidomain.Brand{Name: "Other", Domain: "other.dev", Country: "IT"}
	require.NoError(t, store.BrandRepo().Create(otherCtx, otherBrand))

	_, err = koidomain.NewCompetitorCommander(store).Create(otherCtx, otherBrand.ID, koidomain.CompetitorParams{
		Name:   "Acme",
		Domain: "acme.com",
	})

	require.NoError(t, err)
	_, err = store.CompetitorRepo().Get(otherCtx, koidomain.CompetitorFilter{ID: page.Items[0].ID})
	require.ErrorIs(t, err, koidomain.ErrNotFound)
	_, err = store.BrandRepo().Get(otherCtx, brand.ID)
	require.ErrorIs(t, err, koidomain.ErrNotFound)
	require.ErrorIs(t, store.BrandRepo().Update(otherCtx, brand), koidomain.ErrNotFound)
	confirmed := false
	filter := koidomain.CompetitorFilter{ID: page.Items[0].ID, BrandID: brand.ID}
	require.ErrorIs(t, store.CompetitorRepo().Update(otherCtx, filter, koidomain.CompetitorUpdate{Confirmed: &confirmed}), koidomain.ErrNotFound)
	require.ErrorIs(t, store.CompetitorRepo().Delete(otherCtx, filter), koidomain.ErrNotFound)
	require.ErrorIs(t, store.CompetitorRepo().Create(otherCtx, &koidomain.Competitor{BrandID: brand.ID}), koidomain.ErrNotFound)

	prompt := &koidomain.Prompt{
		BrandID:   brand.ID,
		Text:      "Who is Koi?",
		Source:    koidomain.PromptSourceManual,
		Confirmed: true,
	}
	require.NoError(t, store.PromptRepo().Create(ctx, prompt))
	promptFilter := koidomain.PromptFilter{ID: prompt.ID, BrandID: brand.ID}
	_, err = store.PromptRepo().Get(otherCtx, promptFilter)
	require.ErrorIs(t, err, koidomain.ErrNotFound)
	require.ErrorIs(t, store.PromptRepo().Update(otherCtx, promptFilter, koidomain.PromptUpdate{Confirmed: &confirmed}), koidomain.ErrNotFound)
	require.ErrorIs(t, store.PromptRepo().Delete(otherCtx, promptFilter), koidomain.ErrNotFound)
	require.ErrorIs(t, store.PromptRepo().Create(otherCtx, &koidomain.Prompt{BrandID: brand.ID}), koidomain.ErrNotFound)
}
