package koidatabase

import (
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/koiauth"
	"github.com/koihq/koi/pkg/koidomain"
)

func TestCompetitorUniqueDomainConcurrent(t *testing.T) {
	dsn := os.Getenv("KOI_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set KOI_TEST_DATABASE_URL to run the Postgres integration test")
	}
	admin, err := Open(dsn)
	require.NoError(t, err)
	schemaName := "koi_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	require.NoError(t, admin.Exec("CREATE SCHEMA "+schemaName).Error)
	t.Cleanup(func() { require.NoError(t, admin.Exec("DROP SCHEMA "+schemaName+" CASCADE").Error) })

	scopedURL, err := url.Parse(dsn)
	require.NoError(t, err)
	params := scopedURL.Query()
	params.Set("search_path", schemaName)
	scopedURL.RawQuery = params.Encode()
	db, err := Open(scopedURL.String())
	require.NoError(t, err)
	require.NoError(t, Migrate(db))
	store := NewStore(db)
	user := &koidomain.User{Email: uuid.NewString() + "@example.com", PasswordHash: "hash"}
	require.NoError(t, store.UserRepo().Create(t.Context(), user))
	ctx := koiauth.WithIdentity(t.Context(), koiauth.Identity{UserID: user.ID})
	brand := &koidomain.Brand{UserID: uuid.New(), Name: "Koi", Domain: "koi.dev", Country: "IT"}
	require.NoError(t, store.BrandRepo().Create(ctx, brand))
	require.Equal(t, user.ID, brand.UserID)

	errors := make(chan error, 2)
	for range 2 {
		go func() {
			_, err := koidomain.NewCompetitorCommander(store).Create(ctx, brand.ID, koidomain.CompetitorParams{
				Name: "Acme", Domain: "ACME.COM",
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
		require.Equal(t, []koidomain.FieldError{{Field: "domain", Code: koidomain.FieldTaken}}, domainErr.Fields)
		taken++
	}
	require.Equal(t, 1, succeeded)
	require.Equal(t, 1, taken)

	page, err := store.CompetitorRepo().List(ctx, koidomain.CompetitorFilter{
		BrandID: brand.ID, Page: koidomain.PageParams{Page: 1, PageSize: 10},
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, page.TotalItems)

	otherUser := &koidomain.User{Email: uuid.NewString() + "@example.com", PasswordHash: "hash"}
	require.NoError(t, store.UserRepo().Create(t.Context(), otherUser))
	otherCtx := koiauth.WithIdentity(t.Context(), koiauth.Identity{UserID: otherUser.ID})
	otherBrand := &koidomain.Brand{Name: "Other", Domain: "other.dev", Country: "IT"}
	require.NoError(t, store.BrandRepo().Create(otherCtx, otherBrand))
	_, err = koidomain.NewCompetitorCommander(store).Create(otherCtx, otherBrand.ID, koidomain.CompetitorParams{
		Name: "Acme", Domain: "acme.com",
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

	prompt := &koidomain.Prompt{BrandID: brand.ID, Text: "Who is Koi?", Source: koidomain.PromptSourceManual, Confirmed: true}
	require.NoError(t, store.PromptRepo().Create(ctx, prompt))
	promptFilter := koidomain.PromptFilter{ID: prompt.ID, BrandID: brand.ID}
	_, err = store.PromptRepo().Get(otherCtx, promptFilter)
	require.ErrorIs(t, err, koidomain.ErrNotFound)
	require.ErrorIs(t, store.PromptRepo().Update(otherCtx, promptFilter, koidomain.PromptUpdate{Confirmed: &confirmed}), koidomain.ErrNotFound)
	require.ErrorIs(t, store.PromptRepo().Delete(otherCtx, promptFilter), koidomain.ErrNotFound)
	require.ErrorIs(t, store.PromptRepo().Create(otherCtx, &koidomain.Prompt{BrandID: brand.ID}), koidomain.ErrNotFound)
}
