package domain

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

func TestCompetitorRejectsUnownedBrandBeforeChildLookup(t *testing.T) {
	store := NewMockStore(t)
	brands := NewMockBrandRepository(t)
	userID, brandID := uuid.New(), uuid.New()
	store.EXPECT().BrandRepo().Return(brands)
	brands.EXPECT().Get(mock.Anything, brandID).Return(nil, ErrNotFound)

	_, err := NewCompetitorQuerier(store, nil).List(identityCtx(t, userID), CompetitorFilter{BrandFilter: BrandFilter{BrandID: brandID}}, PageParams{Page: 1, PageSize: 10})
	require.ErrorIs(t, err, ErrNotFound)
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

// competitorCommandDeps runs Atomic against the same mocks and records the order of the steps.
type competitorCommandDeps struct {
	store       *MockStore
	brands      *MockBrandRepository
	competitors *MockCompetitorRepository
	visibility  *MockVisibilityRepository
	steps       []string
}

func newCompetitorCommandDeps(t *testing.T) *competitorCommandDeps {
	t.Helper()

	d := &competitorCommandDeps{
		store:       NewMockStore(t),
		brands:      NewMockBrandRepository(t),
		competitors: NewMockCompetitorRepository(t),
		visibility:  NewMockVisibilityRepository(t),
	}
	d.store.EXPECT().BrandRepo().Return(d.brands).Maybe()
	d.store.EXPECT().CompetitorRepo().Return(d.competitors).Maybe()
	d.store.EXPECT().VisibilityRepo().Return(d.visibility).Maybe()
	d.store.EXPECT().Atomic(mock.Anything, mock.Anything).RunAndReturn(
		func(_ context.Context, fn func(Store) error) error { return fn(d.store) },
	).Maybe()

	return d
}

func (d *competitorCommandDeps) lockBrand(brandID uuid.UUID) {
	d.brands.EXPECT().LockForUpdate(mock.Anything, brandID).Run(func(context.Context, uuid.UUID) {
		d.steps = append(d.steps, "lock")
	}).Return(&Brand{BaseEntity: BaseEntity{ID: brandID}}, nil).Once()
}

func (d *competitorCommandDeps) noOthers(brandID uuid.UUID) {
	d.competitors.EXPECT().ForBrand(mock.Anything, brandID).Run(func(context.Context, uuid.UUID) {
		d.steps = append(d.steps, "check")
	}).Return(nil, nil).Once()
}

func (d *competitorCommandDeps) locked(competitor *Competitor) {
	d.competitors.EXPECT().Get(mock.Anything, competitor.ID).Return(competitor, nil).Twice()
	d.lockBrand(competitor.BrandID)
}

func (d *competitorCommandDeps) invalidateReview(brandID uuid.UUID) {
	d.visibility.EXPECT().InvalidateReview(mock.Anything, brandID, ReviewCompetitors).Run(func(context.Context, uuid.UUID, string) {
		d.steps = append(d.steps, "invalidate")
	}).Return(nil).Once()
}

func TestCompetitorCreateChecksURLsAndInvalidatesReviewInOrder(t *testing.T) {
	d := newCompetitorCommandDeps(t)
	brandID, competitorID := uuid.New(), uuid.New()
	d.lockBrand(brandID)
	d.noOthers(brandID)
	d.competitors.EXPECT().Create(mock.Anything, mock.MatchedBy(func(competitor *Competitor) bool {
		return competitor.Name == "Acme" && competitor.Domain == "acme.com" && competitor.Source == CompetitorSourceManual && competitor.Confirmed
	})).RunAndReturn(func(_ context.Context, competitor *Competitor) error {
		d.steps = append(d.steps, "create")
		competitor.ID = competitorID

		return nil
	}).Once()
	d.invalidateReview(brandID)

	created, err := NewCompetitorCommander(d.store).Create(t.Context(), brandID, CreateCompetitorParams{
		Name:   " Acme ",
		Domain: " ACME.COM ",
	})
	require.NoError(t, err)
	require.Equal(t, competitorID, created.ID)
	require.Equal(t, []string{"lock", "check", "create", "invalidate"}, d.steps)
}

func TestCompetitorCreateChecksTheBrandBeforeValidating(t *testing.T) {
	d := newCompetitorCommandDeps(t)
	brandID := uuid.New()
	d.brands.EXPECT().LockForUpdate(mock.Anything, brandID).Return(nil, ErrNotFound).Once()

	_, err := NewCompetitorCommander(d.store).Create(t.Context(), brandID, CreateCompetitorParams{Name: " "})
	require.ErrorIs(t, err, ErrNotFound)
}

func TestCompetitorUpdateKeepsProvenanceConfirmationAndReview(t *testing.T) {
	d := newCompetitorCommandDeps(t)
	competitor := &Competitor{
		BaseEntity: BaseEntity{ID: uuid.New()},
		BrandID:    uuid.New(),
		Name:       "Old",
		Domain:     "old.dev",
		Source:     CompetitorSourceDrafted,
	}
	d.locked(competitor)
	d.noOthers(competitor.BrandID)
	d.competitors.EXPECT().Update(mock.Anything, competitor).Return(nil).Once()

	// No InvalidateReview expectation: the mock fails the test if the review is touched.
	name := " New   name "
	updated, err := NewCompetitorCommander(d.store).Update(t.Context(), competitor.ID, UpdateCompetitorParams{Name: &name})
	require.NoError(t, err)
	require.Equal(t, "New name", updated.Name)
	require.Equal(t, CompetitorSourceDrafted, updated.Source)
	require.False(t, updated.Confirmed)
}

func TestCompetitorEmptyUpdateWritesAndValidatesNothing(t *testing.T) {
	d := newCompetitorCommandDeps(t)
	legacy := &Competitor{BaseEntity: BaseEntity{ID: uuid.New()}, BrandID: uuid.New(), Name: " "}
	d.locked(legacy)

	got, err := NewCompetitorCommander(d.store).Update(t.Context(), legacy.ID, UpdateCompetitorParams{})
	require.NoError(t, err)
	require.Same(t, legacy, got)
}

func TestCompetitorConfirmRequiresAnIdentityAndUniqueURLs(t *testing.T) {
	brandID := uuid.New()

	anonymous := newCompetitorCommandDeps(t)
	observed := &Competitor{BaseEntity: BaseEntity{ID: uuid.New()}, BrandID: brandID, Name: "Acme", Source: CompetitorSourceObserved}
	anonymous.locked(observed)
	_, err := NewCompetitorCommander(anonymous.store).Confirm(t.Context(), observed.ID)
	require.ErrorIs(t, err, ErrValidation)

	taken := newCompetitorCommandDeps(t)
	drafted := &Competitor{BaseEntity: BaseEntity{ID: uuid.New()}, BrandID: brandID, Name: "Acme", Domain: "acme.com", Source: CompetitorSourceDrafted}
	taken.locked(drafted)
	taken.competitors.EXPECT().ForBrand(mock.Anything, brandID).Return([]Competitor{
		*drafted,
		{BaseEntity: BaseEntity{ID: uuid.New()}, BrandID: brandID, Name: "Other", Domain: "acme.com"},
	}, nil).Once()
	_, err = NewCompetitorCommander(taken.store).Confirm(t.Context(), drafted.ID)

	var domainErr *Error
	require.ErrorAs(t, err, &domainErr)
	require.Equal(t, []FieldError{{Field: "domain", Code: FieldTaken}}, domainErr.Fields)

	valid := newCompetitorCommandDeps(t)
	ready := &Competitor{BaseEntity: BaseEntity{ID: uuid.New()}, BrandID: brandID, Name: "Acme", Domain: "acme.com", Source: CompetitorSourceDrafted}
	valid.locked(ready)
	valid.noOthers(brandID)
	valid.competitors.EXPECT().Update(mock.Anything, ready).Return(nil).Once()
	confirmed, err := NewCompetitorCommander(valid.store).Confirm(t.Context(), ready.ID)
	require.NoError(t, err)
	require.True(t, confirmed.Confirmed)

	valid.visibility.EXPECT().FinishReview(mock.Anything, brandID, ReviewCompetitors).Return(nil).Once()
	require.NoError(t, NewCompetitorCommander(valid.store).ConfirmAll(t.Context(), brandID))
}

func TestCompetitorDeleteDismissesObservedAndInvalidatesOnlyReviewedSets(t *testing.T) {
	brandID := uuid.New()

	suggestion := newCompetitorCommandDeps(t)
	observed := &Competitor{BaseEntity: BaseEntity{ID: uuid.New()}, BrandID: brandID, Name: "Acme", Source: CompetitorSourceObserved}
	suggestion.locked(observed)
	suggestion.competitors.EXPECT().Observations(mock.Anything, brandID).Return(nil, nil).Once()
	suggestion.competitors.EXPECT().SaveObservations(mock.Anything, mock.MatchedBy(func(states []CompetitorObservation) bool {
		return len(states) == 1 && states[0].Dismissed && states[0].ID == uuid.Nil
	})).Run(func(context.Context, []CompetitorObservation) {
		suggestion.steps = append(suggestion.steps, "dismiss")
	}).Return(nil).Once()
	suggestion.competitors.EXPECT().Delete(mock.Anything, observed.ID).Run(func(context.Context, uuid.UUID) {
		suggestion.steps = append(suggestion.steps, "delete")
	}).Return(nil).Once()

	require.NoError(t, NewCompetitorCommander(suggestion.store).Delete(t.Context(), observed.ID))
	require.Equal(t, []string{"lock", "dismiss", "delete"}, suggestion.steps)

	reviewed := newCompetitorCommandDeps(t)
	confirmedObserved := &Competitor{BaseEntity: BaseEntity{ID: uuid.New()}, BrandID: brandID, Name: "Acme", Domain: "acme.com", Source: CompetitorSourceObserved, Confirmed: true}
	reviewed.locked(confirmedObserved)
	reviewed.competitors.EXPECT().Observations(mock.Anything, brandID).Return(nil, nil).Once()
	reviewed.competitors.EXPECT().SaveObservations(mock.Anything, mock.Anything).Return(nil).Once()
	reviewed.competitors.EXPECT().Delete(mock.Anything, confirmedObserved.ID).Return(nil).Once()
	reviewed.invalidateReview(brandID)
	require.NoError(t, NewCompetitorCommander(reviewed.store).Delete(t.Context(), confirmedObserved.ID))

	manual := newCompetitorCommandDeps(t)
	added := &Competitor{BaseEntity: BaseEntity{ID: uuid.New()}, BrandID: brandID, Name: "Acme", Domain: "acme.com", Source: CompetitorSourceManual, Confirmed: true}
	manual.locked(added)
	manual.competitors.EXPECT().Delete(mock.Anything, added.ID).Return(nil).Once()
	manual.invalidateReview(brandID)
	require.NoError(t, NewCompetitorCommander(manual.store).Delete(t.Context(), added.ID))
}

func TestCompetitorConflictsWithReportsTheField(t *testing.T) {
	self := Competitor{
		BaseEntity:     BaseEntity{ID: uuid.New()},
		Name:           "Acme",
		Domain:         "acme.com",
		GitHubRepo:     "https://github.com/acme/app",
		AdditionalURLs: []string{"https://acme.tools"},
	}

	require.NoError(t, self.ConflictsWith([]Competitor{self}), "a competitor does not conflict with itself")
	require.NoError(t, self.ConflictsWith([]Competitor{{BaseEntity: BaseEntity{ID: uuid.New()}, Domain: "other.com"}}))

	for field, other := range map[string]Competitor{
		"domain":          {BaseEntity: BaseEntity{ID: uuid.New()}, Domain: "acme.com"},
		"github_repo":     {BaseEntity: BaseEntity{ID: uuid.New()}, GitHubRepo: "https://github.com/acme/app"},
		"additional_urls": {BaseEntity: BaseEntity{ID: uuid.New()}, Domain: "acme.tools"},
	} {
		var domainErr *Error
		require.ErrorAs(t, self.ConflictsWith([]Competitor{other}), &domainErr, field)
		require.Equal(t, []FieldError{{Field: field, Code: FieldTaken}}, domainErr.Fields)
	}
}

func TestCompetitorDismissalsMatchExistingObservationsOrCreateOne(t *testing.T) {
	competitor := Competitor{
		BaseEntity: BaseEntity{ID: uuid.New()},
		BrandID:    uuid.New(),
		Name:       "Acme",
		Domain:     "acme.com",
		Source:     CompetitorSourceObserved,
	}

	created := competitor.Dismissals(nil)
	require.Len(t, created, 1)
	require.True(t, created[0].Dismissed)
	require.Equal(t, uuid.Nil, created[0].ID)
	require.Equal(t, competitor.BrandID, created[0].BrandID)
	require.Equal(t, AppearanceNameKey("Acme"), created[0].Key)

	matching := CompetitorObservation{BaseEntity: BaseEntity{ID: uuid.New()}, Identity: Competitor{BaseEntity: BaseEntity{ID: competitor.ID}, Name: "Acme Inc"}}
	unrelated := CompetitorObservation{BaseEntity: BaseEntity{ID: uuid.New()}, Identity: Competitor{BaseEntity: BaseEntity{ID: uuid.New()}, Name: "Unrelated"}}

	dismissed := competitor.Dismissals([]CompetitorObservation{matching, unrelated})
	require.Len(t, dismissed, 1)
	require.Equal(t, matching.ID, dismissed[0].ID)
	require.True(t, dismissed[0].Dismissed)
	require.Contains(t, dismissed[0].Identity.Aliases, "Acme Inc")
}
