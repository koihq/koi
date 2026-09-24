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
		BrandID: uuid.New(), Name: "  Acme   Labs ", Domain: " ACME.COM ",
		Aliases: []string{" Acme ", "", "acme", "  Acme AI  "}, Source: CompetitorSourceManual, Confirmed: true,
	}
	competitor.Normalize()
	require.Equal(t, "Acme Labs", competitor.Name)
	require.Equal(t, "acme.com", competitor.Domain)
	require.Equal(t, []string{"Acme", "Acme AI"}, competitor.Aliases)
	require.NoError(t, competitor.Validate())

	for _, invalid := range []Competitor{
		{Name: "Acme", Domain: "acme.com", Source: CompetitorSourceManual, Confirmed: true},
		{BrandID: uuid.New(), Domain: "acme.com", Source: CompetitorSourceManual, Confirmed: true},
		{BrandID: uuid.New(), Name: "Acme", Domain: "https://acme.com", Source: CompetitorSourceManual, Confirmed: true},
		{BrandID: uuid.New(), Name: "Acme", Domain: "acme.com", Source: "imported", Confirmed: true},
		{BrandID: uuid.New(), Name: "Acme", Domain: "acme.com", Source: CompetitorSourceManual},
	} {
		require.ErrorIs(t, invalid.Validate(), ErrValidation)
	}
}

func TestCompetitorManualCreateAndImmutableProvenance(t *testing.T) {
	store := NewMockStore(t)
	brands := NewMockBrandRepository(t)
	competitors := NewMockCompetitorRepository(t)
	userID, brandID, competitorID := uuid.New(), uuid.New(), uuid.New()
	store.EXPECT().BrandRepo().Return(brands).Twice()
	store.EXPECT().CompetitorRepo().Return(competitors).Times(3)
	brands.EXPECT().GetByIDAndUserID(mock.Anything, brandID, userID).Return(&Brand{}, nil).Twice()
	competitors.EXPECT().Create(mock.Anything, mock.MatchedBy(func(competitor *Competitor) bool {
		return competitor.Name == "Acme" && competitor.Domain == "acme.com" && competitor.Source == CompetitorSourceManual && competitor.Confirmed
	})).RunAndReturn(func(_ context.Context, competitor *Competitor) error {
		competitor.ID = competitorID
		return nil
	})

	created, err := NewCompetitorCommander(store).Create(t.Context(), userID, brandID, CompetitorParams{
		Name: " Acme ", Domain: " ACME.COM ", Aliases: []string{"Acme AI", "acme ai"},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"Acme AI"}, created.Aliases)

	observed := &Competitor{
		BaseEntity: BaseEntity{ID: competitorID}, BrandID: brandID, Name: "Acme", Domain: "acme.com",
		Source: CompetitorSourceObserved, Confirmed: false,
	}
	competitors.EXPECT().Get(mock.Anything, CompetitorFilter{ID: competitorID, BrandID: brandID}).Return(observed, nil)
	competitors.EXPECT().Update(mock.Anything, CompetitorFilter{ID: competitorID, BrandID: brandID}, mock.MatchedBy(func(update CompetitorUpdate) bool {
		return update.Name != nil && *update.Name == "Acme AI" && update.Confirmed == nil
	})).Return(nil)
	name := "Acme AI"
	updated, err := NewCompetitorCommander(store).Update(t.Context(), userID, brandID, competitorID, CompetitorPatch{Name: &name})
	require.NoError(t, err)
	require.Equal(t, CompetitorSourceObserved, updated.Source)
	require.False(t, updated.Confirmed)
}

func TestCompetitorRejectsUnownedBrandBeforeChildLookup(t *testing.T) {
	store := NewMockStore(t)
	brands := NewMockBrandRepository(t)
	userID, brandID := uuid.New(), uuid.New()
	store.EXPECT().BrandRepo().Return(brands)
	brands.EXPECT().GetByIDAndUserID(mock.Anything, brandID, userID).Return(nil, ErrNotFound)

	_, err := NewCompetitorQuerier(store).List(t.Context(), userID, brandID, PageParams{Page: 1, PageSize: 10}, "", nil)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestCompetitorConfirmationUsesUpdate(t *testing.T) {
	store := NewMockStore(t)
	brands := NewMockBrandRepository(t)
	competitors := NewMockCompetitorRepository(t)
	userID, brandID, competitorID := uuid.New(), uuid.New(), uuid.New()
	store.EXPECT().BrandRepo().Return(brands).Twice()
	store.EXPECT().CompetitorRepo().Return(competitors).Times(3)
	brands.EXPECT().GetByIDAndUserID(mock.Anything, brandID, userID).Return(&Brand{}, nil).Twice()
	competitor := &Competitor{BaseEntity: BaseEntity{ID: competitorID}, BrandID: brandID}
	filter := CompetitorFilter{ID: competitorID, BrandID: brandID}
	competitors.EXPECT().Get(mock.Anything, filter).Return(competitor, nil)
	competitors.EXPECT().Update(mock.Anything, filter, mock.MatchedBy(func(update CompetitorUpdate) bool {
		return update.Confirmed != nil && *update.Confirmed
	})).Return(nil)

	confirmed, err := NewCompetitorCommander(store).Confirm(t.Context(), userID, brandID, competitorID)
	require.NoError(t, err)
	require.True(t, confirmed.Confirmed)

	competitors.EXPECT().Update(mock.Anything, mock.MatchedBy(func(filter CompetitorFilter) bool {
		return filter.BrandID == brandID && filter.Confirmed != nil && !*filter.Confirmed
	}), mock.MatchedBy(func(update CompetitorUpdate) bool {
		return update.Confirmed != nil && *update.Confirmed
	})).Return(nil)
	require.NoError(t, NewCompetitorCommander(store).ConfirmAll(t.Context(), userID, brandID))
}
