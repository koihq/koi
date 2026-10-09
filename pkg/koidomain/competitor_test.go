package koidomain

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCompetitorValidationAndNormalization(t *testing.T) {
	competitor := Competitor{
		BrandID:   uuid.New(),
		Name:      "  Acme   Labs ",
		Domain:    " ACME.COM ",
		Source:    CompetitorSourceManual,
		Confirmed: true,
	}

	competitor.Normalize()
	require.Equal(t, "Acme Labs", competitor.Name)
	require.Equal(t, "acme.com", competitor.Domain)
	require.NoError(t, competitor.Validate())
	githubOnly := Competitor{
		BrandID:    uuid.New(),
		Name:       "Autosuggest",
		GitHubRepo: " HTTPS://GITHUB.COM/OWNER/REPO/ ",
		Source:     CompetitorSourceManual,
		Confirmed:  true,
	}
	githubOnly.Normalize()
	require.Equal(t, "https://github.com/owner/repo", githubOnly.GitHubRepo)
	require.NoError(t, githubOnly.Validate())

	for _, invalid := range []Competitor{
		{
			Name:      "Acme",
			Domain:    "acme.com",
			Source:    CompetitorSourceManual,
			Confirmed: true,
		},
		{
			BrandID:   uuid.New(),
			Domain:    "acme.com",
			Source:    CompetitorSourceManual,
			Confirmed: true,
		},
		{
			BrandID:   uuid.New(),
			Name:      "Acme",
			Domain:    "https://acme.com",
			Source:    CompetitorSourceManual,
			Confirmed: true,
		},
		{
			BrandID:   uuid.New(),
			Name:      "Acme",
			Source:    CompetitorSourceManual,
			Confirmed: true,
		},
		{
			BrandID:    uuid.New(),
			Name:       "Acme",
			GitHubRepo: "https://github.com/owner",
			Source:     CompetitorSourceManual,
			Confirmed:  true,
		},
		{
			BrandID:   uuid.New(),
			Name:      "Acme",
			Domain:    "acme.com",
			Source:    "imported",
			Confirmed: true,
		},
		{
			BrandID: uuid.New(),
			Name:    "Acme",
			Domain:  "acme.com",
			Source:  CompetitorSourceManual,
		},
	} {
		require.ErrorIs(t, invalid.Validate(), ErrValidation)
	}
}

func TestCompetitorManualCreateAndImmutableProvenance(t *testing.T) {
	store := NewMockStore(t)
	brands := NewMockBrandRepository(t)
	competitors := NewMockCompetitorRepository(t)
	userID, brandID, competitorID := uuid.New(), uuid.New(), uuid.New()
	store.EXPECT().BrandRepo().Return(brands)
	store.EXPECT().CompetitorRepo().Return(competitors).Once()
	brands.EXPECT().Get(mock.Anything, brandID).Return(&Brand{}, nil)
	competitors.EXPECT().Create(mock.Anything, mock.MatchedBy(func(competitor *Competitor) bool {
		return competitor.Name == "Acme" && competitor.Domain == "acme.com" && competitor.Source == CompetitorSourceManual && competitor.Confirmed
	})).RunAndReturn(func(_ context.Context, competitor *Competitor) error {
		competitor.ID = competitorID

		return nil
	})

	created, err := NewCompetitorCommander(store).Create(identityCtx(t, userID), brandID, CompetitorParams{
		Name:   " Acme ",
		Domain: " ACME.COM ",
	})

	require.NoError(t, err)
	require.Equal(t, competitorID, created.ID)
}

func TestCompetitorRejectsUnownedBrandBeforeChildLookup(t *testing.T) {
	store := NewMockStore(t)
	brands := NewMockBrandRepository(t)
	userID, brandID := uuid.New(), uuid.New()
	store.EXPECT().BrandRepo().Return(brands)
	brands.EXPECT().Get(mock.Anything, brandID).Return(nil, ErrNotFound)

	_, err := NewCompetitorQuerier(store, nil).List(identityCtx(t, userID), brandID, PageParams{Page: 1, PageSize: 10}, "", nil)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestCompetitorConfirmationUsesUpdate(t *testing.T) {
	store := NewMockStore(t)
	brands := NewMockBrandRepository(t)
	competitors := NewMockCompetitorRepository(t)
	userID, brandID, competitorID := uuid.New(), uuid.New(), uuid.New()
	store.EXPECT().BrandRepo().Return(brands).Once()
	store.EXPECT().CompetitorRepo().Return(competitors).Twice()
	brands.EXPECT().Get(mock.Anything, brandID).Return(&Brand{}, nil).Once()
	competitor := &Competitor{BaseEntity: BaseEntity{ID: competitorID}, BrandID: brandID}
	filter := CompetitorFilter{ID: competitorID, BrandID: brandID}
	competitors.EXPECT().Get(mock.Anything, filter).Return(competitor, nil)
	competitors.EXPECT().Update(mock.Anything, filter, mock.MatchedBy(func(update CompetitorUpdate) bool {
		return update.Confirmed != nil && *update.Confirmed
	})).Return(nil)

	confirmed, err := NewCompetitorCommander(store).Confirm(identityCtx(t, userID), brandID, competitorID)
	require.NoError(t, err)
	require.True(t, confirmed.Confirmed)

	visibility := NewMockVisibilityRepository(t)
	store.EXPECT().VisibilityRepo().Return(visibility).Once()
	visibility.EXPECT().FinishReview(mock.Anything, brandID, "competitors").Return(nil).Once()

	require.NoError(t, NewCompetitorCommander(store).ConfirmAll(identityCtx(t, userID), brandID))
}

func TestCompetitorSearchRequiresOwnedBrandAndProvider(t *testing.T) {
	store := NewMockStore(t)
	brands := NewMockBrandRepository(t)
	brandID := uuid.New()
	store.EXPECT().BrandRepo().Return(brands).Twice()
	brands.EXPECT().Get(mock.Anything, brandID).Return(&Brand{}, nil).Once()
	brands.EXPECT().Get(mock.Anything, brandID).Return(nil, ErrNotFound).Once()

	_, err := NewCompetitorQuerier(store, nil).Search(identityCtx(t, uuid.New()), brandID, "acme")
	require.ErrorIs(t, err, ErrFeatureUnavailable)
	_, err = NewCompetitorQuerier(store, nil).Search(identityCtx(t, uuid.New()), brandID, "acme")
	require.ErrorIs(t, err, ErrNotFound)
}
