package database

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/domain"
)

var errForcedRollback = errors.New("forced rollback")

func TestRepositoryTransactionInsideAtomicRollsBackWithIt(t *testing.T) {
	f := newRepositoryFixture(t)
	store := NewStore(f.db)
	_, err := domain.NewPromptCommander(store).Create(f.ctx, f.brand.ID, domain.CreatePromptParams{Text: "Question"})
	require.NoError(t, err)

	// StartTracking opens its own transaction, which becomes a savepoint inside Atomic.
	err = store.Atomic(f.ctx, func(tx domain.Store) error {
		if err := tx.VisibilityRepo().StartTracking(f.ctx, f.brand.ID); err != nil {
			return err
		}

		return errForcedRollback
	})
	require.ErrorIs(t, err, errForcedRollback)

	brand, err := store.BrandRepo().Get(f.ctx, f.brand.ID)
	require.NoError(t, err)
	require.Nil(t, brand.TrackingConfirmedAt)
}

func TestPromptCommandsOfAnotherOrganizationAreNotFound(t *testing.T) {
	f := newRepositoryFixture(t)
	commander := domain.NewPromptCommander(NewStore(f.db))

	prompt, err := commander.Create(f.ctx, f.brand.ID, domain.CreatePromptParams{Text: "Private"})
	require.NoError(t, err)

	_, err = commander.Create(f.otherCtx, f.brand.ID, domain.CreatePromptParams{Text: "Intruder"})
	require.ErrorIs(t, err, domain.ErrNotFound)

	text := "Hijacked"
	_, err = commander.Update(f.otherCtx, prompt.ID, domain.UpdatePromptParams{Text: &text})
	require.ErrorIs(t, err, domain.ErrNotFound)

	_, err = commander.Confirm(f.otherCtx, prompt.ID)
	require.ErrorIs(t, err, domain.ErrNotFound)

	require.ErrorIs(t, commander.Delete(f.otherCtx, prompt.ID), domain.ErrNotFound)

	_, err = commander.Create(t.Context(), f.brand.ID, domain.CreatePromptParams{Text: "Anonymous"})
	require.ErrorIs(t, err, domain.ErrUnauthorized)
}

func TestPromptCommandsKeepTrackingAndUniqueTextError(t *testing.T) {
	f := newRepositoryFixture(t)
	store := NewStore(f.db)
	commander := domain.NewPromptCommander(store)

	prompt, err := commander.Create(f.ctx, f.brand.ID, domain.CreatePromptParams{Text: "Tracked"})
	require.NoError(t, err)

	require.NoError(t, store.VisibilityRepo().StartTracking(f.ctx, f.brand.ID))

	text := "Changed"
	_, err = commander.Update(f.ctx, prompt.ID, domain.UpdatePromptParams{Text: &text})
	require.NoError(t, err)

	brand, err := store.BrandRepo().Get(f.ctx, f.brand.ID)
	require.NoError(t, err)
	require.NotNil(t, brand.TrackingConfirmedAt, "editing a prompt does not ask for a new confirmation")

	_, err = commander.Create(f.ctx, f.brand.ID, domain.CreatePromptParams{Text: "changed"})

	var validation *domain.Error
	require.ErrorAs(t, err, &validation)
	require.Equal(t, []domain.FieldError{
		{
			Field: "text",
			Code:  domain.FieldTaken,
		},
	}, validation.Fields)
}

func TestCompetitorCommandsAgainstTheDatabase(t *testing.T) {
	f := newRepositoryFixture(t)
	store := NewStore(f.db)
	commander := domain.NewCompetitorCommander(store)

	require.NoError(t, f.db.Create(&domain.Prompt{
		BrandID:   f.brand.ID,
		Text:      "Question",
		Source:    domain.PromptSourceManual,
		Confirmed: true,
	}).Error)
	require.NoError(t, store.VisibilityRepo().StartTracking(f.ctx, f.brand.ID))

	created, err := commander.Create(f.ctx, f.brand.ID, domain.CreateCompetitorParams{Name: "Acme", Domain: "ACME.com"})
	require.NoError(t, err)

	brand, err := store.BrandRepo().Get(f.ctx, f.brand.ID)
	require.NoError(t, err)
	require.NotNil(t, brand.TrackingConfirmedAt, "adding a competitor does not ask for a new confirmation")

	_, err = commander.Create(f.ctx, f.brand.ID, domain.CreateCompetitorParams{Name: "Copy", Domain: "acme.com"})

	var validation *domain.Error
	require.ErrorAs(t, err, &validation)
	require.Equal(t, []domain.FieldError{{Field: "domain", Code: domain.FieldTaken}}, validation.Fields)

	aliases := []string{"Acme Labs"}
	updated, err := commander.Update(f.ctx, created.ID, domain.UpdateCompetitorParams{Aliases: &aliases})
	require.NoError(t, err)
	require.Equal(t, aliases, updated.Aliases)

	stored, err := store.CompetitorRepo().Get(f.ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, aliases, stored.Aliases)
	require.Equal(t, domain.CompetitorSourceManual, stored.Source)

	_, err = commander.Update(f.otherCtx, created.ID, domain.UpdateCompetitorParams{Aliases: &aliases})
	require.ErrorIs(t, err, domain.ErrNotFound)
	require.ErrorIs(t, commander.Delete(f.otherCtx, created.ID), domain.ErrNotFound)

	_, err = commander.Create(f.otherCtx, f.brand.ID, domain.CreateCompetitorParams{Name: "Intruder", Domain: "intruder.dev"})
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestDeletingAnObservedCompetitorSavesItsDismissal(t *testing.T) {
	f := newRepositoryFixture(t)
	store := NewStore(f.db)

	observed := &domain.Competitor{
		BrandID: f.brand.ID,
		Name:    "Spotted",
		Source:  domain.CompetitorSourceObserved,
	}
	require.NoError(t, store.CompetitorRepo().Create(f.ctx, observed))

	require.NoError(t, domain.NewCompetitorCommander(store).Delete(f.ctx, observed.ID))

	_, err := store.CompetitorRepo().Get(f.ctx, observed.ID)
	require.ErrorIs(t, err, domain.ErrNotFound)

	observations, err := store.CompetitorRepo().Observations(f.ctx, f.brand.ID)
	require.NoError(t, err)
	require.Len(t, observations, 1)
	require.True(t, observations[0].Dismissed)

	foreign, err := store.CompetitorRepo().Observations(f.otherCtx, f.brand.ID)
	require.NoError(t, err)
	require.Empty(t, foreign)
}
