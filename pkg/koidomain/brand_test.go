package koidomain

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

func TestBrandValidate(t *testing.T) {
	valid := Brand{UserID: uuid.New(), Name: "Koi", Domain: "koi.dev", Country: "IT"}
	require.NoError(t, valid.Validate())

	repoOnly := valid
	repoOnly.Domain = ""
	repoOnly.GitHubRepo = "https://github.com/koihq/koi"
	require.NoError(t, repoOnly.Validate())

	both := valid
	both.GitHubRepo = repoOnly.GitHubRepo
	require.NoError(t, both.Validate())

	tests := []struct {
		name  string
		edit  func(*Brand)
		field string
	}{
		{"owner", func(b *Brand) { b.UserID = uuid.Nil }, "user_id"},
		{"name", func(b *Brand) { b.Name = "  " }, "name"},
		{"identifiers", func(b *Brand) { b.Domain = "" }, "domain"},
		{"domain URL", func(b *Brand) { b.Domain = "https://koi.dev" }, "domain"},
		{"domain empty label", func(b *Brand) { b.Domain = "koi..dev" }, "domain"},
		{"domain hyphen", func(b *Brand) { b.Domain = "-koi.dev" }, "domain"},
		{"domain IP", func(b *Brand) { b.Domain = "127.0.0.1" }, "domain"},
		{"country unknown", func(b *Brand) { b.Country = "ZZ" }, "country"},
		{"country alias", func(b *Brand) { b.Country = "UK" }, "country"},
		{"country malformed", func(b *Brand) { b.Country = "ITA" }, "country"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := valid
			tt.edit(&b)
			var domainErr *Error
			require.ErrorAs(t, b.Validate(), &domainErr)
			require.Equal(t, tt.field, domainErr.Fields[0].Field)
		})
	}
}

func TestBrandValidateGitHubRepo(t *testing.T) {
	for _, raw := range []string{
		"http://github.com/koihq/koi",
		"https://example.com/koihq/koi",
		"https://github.com/koihq",
		"https://github.com/koihq/koi/issues",
		"https://github.com/koihq/koi?tab=readme",
		"https://github.com/koihq/koi#readme",
		"https://github.com:443/koihq/koi",
		"https://github.com/../koi",
		"https://github.com/koihq/%6boi",
	} {
		t.Run(raw, func(t *testing.T) {
			b := Brand{UserID: uuid.New(), Name: "Koi", GitHubRepo: raw, Country: "IT"}
			var domainErr *Error
			require.ErrorAs(t, b.Validate(), &domainErr)
			require.Equal(t, "github_repo", domainErr.Fields[0].Field)
		})
	}
}

func TestBrandCommanderCreate(t *testing.T) {
	store := NewMockStore(t)
	users := NewMockUserRepository(t)
	brands := NewMockBrandRepository(t)
	store.EXPECT().Atomic(mock.Anything, mock.Anything).RunAndReturn(
		func(_ context.Context, fn func(Store) error) error { return fn(store) },
	)
	store.EXPECT().UserRepo().Return(users)
	store.EXPECT().BrandRepo().Return(brands)
	userID := uuid.New()
	users.EXPECT().LockForUpdate(mock.Anything, userID).Return(nil)
	brands.EXPECT().Count(mock.Anything).Return(0, nil)
	brands.EXPECT().Create(mock.Anything, mock.MatchedBy(func(b *Brand) bool {
		return b.UserID == userID && b.Name == "Koi" && b.Domain == "koi.dev" &&
			b.Country == "IT" && b.Description == "AI visibility"
	})).RunAndReturn(func(_ context.Context, b *Brand) error {
		b.ID = uuid.New()
		return nil
	})

	brand, err := NewBrandCommander(store).Create(identityCtx(t, userID), BrandParams{
		Name: " Koi ", Domain: " koi.dev ", Country: " it ", Description: " AI visibility ",
	})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, brand.ID)
	require.Equal(t, userID, brand.UserID)
}

func TestBrandCommanderLimit(t *testing.T) {
	store := NewMockStore(t)
	users := NewMockUserRepository(t)
	brands := NewMockBrandRepository(t)
	store.EXPECT().Atomic(mock.Anything, mock.Anything).RunAndReturn(
		func(_ context.Context, fn func(Store) error) error { return fn(store) },
	)
	store.EXPECT().UserRepo().Return(users)
	store.EXPECT().BrandRepo().Return(brands)
	userID := uuid.New()
	users.EXPECT().LockForUpdate(mock.Anything, userID).Return(nil)
	brands.EXPECT().Count(mock.Anything).Return(1, nil)

	brand, err := NewBrandCommander(store).Create(identityCtx(t, userID), BrandParams{
		Name: "Koi", Domain: "koi.dev", Country: "IT",
	})
	require.Nil(t, brand)
	require.ErrorIs(t, err, ErrBrandLimitReached)
}

func TestBrandCommanderUpdate(t *testing.T) {
	store := NewMockStore(t)
	repo := NewMockBrandRepository(t)
	store.EXPECT().BrandRepo().Return(repo).Twice()
	userID, id := uuid.New(), uuid.New()
	brand := &Brand{BaseEntity: BaseEntity{ID: id}, UserID: userID, Name: "Koi", Domain: "koi.dev", Country: "IT"}
	repo.EXPECT().Get(mock.Anything, id).Return(brand, nil)
	repo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(b *Brand) bool {
		return b.ID == id && b.UserID == userID && b.Name == "Koi" && b.Domain == "" &&
			b.GitHubRepo == "https://github.com/koihq/koi" && b.Description == "AI visibility"
	})).Return(nil)
	empty, repoURL, description := "", " https://github.com/koihq/koi ", " AI visibility "

	updated, err := NewBrandCommander(store).Update(identityCtx(t, userID), id, BrandPatch{
		Domain: &empty, GitHubRepo: &repoURL, Description: &description,
	})
	require.NoError(t, err)
	require.Equal(t, userID, updated.UserID)
	require.Equal(t, "AI visibility", updated.Description)
}

func TestBrandCommanderUpdateRejectsInvalidAndOtherOwner(t *testing.T) {
	userID, id := uuid.New(), uuid.New()
	store := NewMockStore(t)
	repo := NewMockBrandRepository(t)
	store.EXPECT().BrandRepo().Return(repo)
	repo.EXPECT().Get(mock.Anything, id).Return(nil, ErrNotFound)
	name := "Stolen"
	_, err := NewBrandCommander(store).Update(identityCtx(t, userID), id, BrandPatch{Name: &name})
	require.ErrorIs(t, err, ErrNotFound)

	store2 := NewMockStore(t)
	repo2 := NewMockBrandRepository(t)
	store2.EXPECT().BrandRepo().Return(repo2)
	repo2.EXPECT().Get(mock.Anything, id).Return(&Brand{
		BaseEntity: BaseEntity{ID: id}, UserID: userID, Name: "Koi", Domain: "koi.dev", Country: "IT",
	}, nil)
	invalid := "ZZ"
	_, err = NewBrandCommander(store2).Update(identityCtx(t, userID), id, BrandPatch{Country: &invalid})
	var domainErr *Error
	require.ErrorAs(t, err, &domainErr)
	require.Equal(t, "country", domainErr.Fields[0].Field)
}

func TestBrandQuerierPages(t *testing.T) {
	store := NewMockStore(t)
	repo := NewMockBrandRepository(t)
	store.EXPECT().BrandRepo().Return(repo)
	userID := uuid.New()
	repo.EXPECT().List(mock.Anything, PageParams{Page: 2, PageSize: 10}).Return(&Page[Brand]{
		Items: []Brand{{UserID: userID}}, TotalItems: 21, TotalPages: 3,
		CurrentPage: 2, HasNext: true, HasPrev: true,
	}, nil)
	page, err := NewBrandQuerier(store).List(identityCtx(t, userID), PageParams{Page: 2, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, &Page[Brand]{
		Items: []Brand{{UserID: userID}}, TotalItems: 21, TotalPages: 3,
		CurrentPage: 2, HasNext: true, HasPrev: true,
	}, page)

	repo.EXPECT().List(mock.Anything, PageParams{Page: 10, PageSize: 10}).Return(&Page[Brand]{
		Items: []Brand{}, TotalItems: 21, TotalPages: 3, CurrentPage: 10, HasPrev: true,
	}, nil)
	page, err = NewBrandQuerier(store).List(identityCtx(t, userID), PageParams{Page: 10, PageSize: 10})
	require.NoError(t, err)
	require.Empty(t, page.Items)
	require.EqualValues(t, 21, page.TotalItems)
	require.EqualValues(t, 3, page.TotalPages)
	require.Equal(t, 10, page.CurrentPage)
	require.False(t, page.HasNext)
	require.True(t, page.HasPrev)
}

func TestBrandQuerierEmptyPage(t *testing.T) {
	store := NewMockStore(t)
	repo := NewMockBrandRepository(t)
	store.EXPECT().BrandRepo().Return(repo)
	userID := uuid.New()
	repo.EXPECT().List(mock.Anything, PageParams{Page: 1, PageSize: 10}).Return(&Page[Brand]{Items: []Brand{}, CurrentPage: 1}, nil)
	page, err := NewBrandQuerier(store).List(identityCtx(t, userID), PageParams{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, &Page[Brand]{Items: []Brand{}, CurrentPage: 1}, page)
}

func TestSupportedCountriesMatchBrandValidation(t *testing.T) {
	codes := SupportedCountries()
	require.Contains(t, codes, "IT")
	require.Contains(t, codes, "XK")
	require.NotContains(t, codes, "ZZ")
	require.NotContains(t, codes, "AC")
	require.True(t, slices.IsSorted(codes))
	require.Len(t, slices.Compact(slices.Clone(codes)), len(codes))
	for _, code := range codes {
		require.NoError(t, (Brand{UserID: uuid.New(), Name: "Koi", Domain: "koi.dev", Country: code}).Validate())
	}
	for _, code := range []string{"AC", "ZZ"} {
		var domainErr *Error
		require.ErrorAs(t, (Brand{UserID: uuid.New(), Name: "Koi", Domain: "koi.dev", Country: code}).Validate(), &domainErr)
		require.Equal(t, "country", domainErr.Fields[0].Field)
	}
	codes[0] = "ZZ"
	require.Equal(t, "AD", SupportedCountries()[0])
}

func TestSearchSupportedCountries(t *testing.T) {
	all := SearchSupportedCountries("")
	require.Len(t, all, len(SupportedCountries()))
	require.Equal(t, all, SearchSupportedCountries(" \t\n"))
	nameOrder := collate.New(language.English)
	require.True(t, slices.IsSortedFunc(all, func(a, b CountryOption) int {
		if order := nameOrder.CompareString(a.Name, b.Name); order != 0 {
			return order
		}
		return strings.Compare(a.Code, b.Code)
	}))
	require.Equal(t, []CountryOption{{Code: "IT", Name: "Italy"}}, SearchSupportedCountries("it"))
	require.Equal(t, []CountryOption{{Code: "IT", Name: "Italy"}}, SearchSupportedCountries("ITALY"))
	require.Empty(t, SearchSupportedCountries("tal"))
	require.Len(t, SearchSupportedCountries("s"), 20)
	require.True(t, slices.ContainsFunc(all, func(c CountryOption) bool { return c.Code == "XK" }))
}
