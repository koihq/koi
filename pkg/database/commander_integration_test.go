package database

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/koihq/koi/pkg/domain"
)

var errReviewFailed = errors.New("review failed")

// failingReviewStore fails the last step of a prompt command, inside its transaction.
type failingReviewStore struct {
	domain.Store
}

func (s failingReviewStore) Atomic(ctx context.Context, fn func(domain.Store) error) error {
	return s.Store.Atomic(ctx, func(tx domain.Store) error {
		return fn(failingReviewStore{tx})
	})
}

func (s failingReviewStore) VisibilityRepo() domain.VisibilityRepository {
	return failingReviewRepository{s.Store.VisibilityRepo()}
}

type failingReviewRepository struct {
	domain.VisibilityRepository
}

func (failingReviewRepository) InvalidateReview(context.Context, uuid.UUID, string) error {
	return errReviewFailed
}

func TestPromptCommandsRollBackWhenReviewInvalidationFails(t *testing.T) {
	f := newRepositoryFixture(t)
	store := NewStore(f.db)
	commander := domain.NewPromptCommander(store)
	failing := domain.NewPromptCommander(failingReviewStore{store})

	prompt, err := commander.Create(f.ctx, f.brand.ID, domain.CreatePromptParams{Text: "Original"})
	require.NoError(t, err)

	_, err = failing.Create(f.ctx, f.brand.ID, domain.CreatePromptParams{Text: "Rolled back"})
	require.ErrorIs(t, err, errReviewFailed)

	var created int64
	require.NoError(t, f.db.Model(&domain.Prompt{}).Where("text = ?", "Rolled back").Count(&created).Error)
	require.Zero(t, created)

	text := "Changed"
	_, err = failing.Update(f.ctx, prompt.ID, domain.UpdatePromptParams{Text: &text})
	require.ErrorIs(t, err, errReviewFailed)

	require.ErrorIs(t, failing.Delete(f.ctx, prompt.ID), errReviewFailed)

	stored, err := store.PromptRepo().Get(f.ctx, prompt.ID)
	require.NoError(t, err)
	require.Equal(t, "Original", stored.Text)
}

func TestRepositoryTransactionInsideAtomicRollsBackWithIt(t *testing.T) {
	f := newRepositoryFixture(t)
	store := NewStore(f.db)
	_, err := domain.NewPromptCommander(store).Create(f.ctx, f.brand.ID, domain.CreatePromptParams{Text: "Question"})
	require.NoError(t, err)

	// FinishReview opens its own transaction, which becomes a savepoint inside Atomic.
	err = store.Atomic(f.ctx, func(tx domain.Store) error {
		if err := tx.VisibilityRepo().FinishReview(f.ctx, f.brand.ID, domain.ReviewPrompts); err != nil {
			return err
		}

		return errReviewFailed
	})
	require.ErrorIs(t, err, errReviewFailed)

	brand, err := store.BrandRepo().Get(f.ctx, f.brand.ID)
	require.NoError(t, err)
	require.Nil(t, brand.PromptsReviewedAt)
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

func TestPromptCommandsInvalidateReviewAndKeepUniqueTextError(t *testing.T) {
	f := newRepositoryFixture(t)
	store := NewStore(f.db)
	commander := domain.NewPromptCommander(store)

	prompt, err := commander.Create(f.ctx, f.brand.ID, domain.CreatePromptParams{Text: "Reviewed"})
	require.NoError(t, err)

	require.NoError(t, store.VisibilityRepo().FinishReview(f.ctx, f.brand.ID, domain.ReviewPrompts))

	reviewed, err := store.BrandRepo().Get(f.ctx, f.brand.ID)
	require.NoError(t, err)
	require.NotNil(t, reviewed.PromptsReviewedAt)

	_, err = commander.Update(f.ctx, prompt.ID, domain.UpdatePromptParams{})
	require.NoError(t, err)

	unchanged, err := store.BrandRepo().Get(f.ctx, f.brand.ID)
	require.NoError(t, err)
	require.NotNil(t, unchanged.PromptsReviewedAt, "an empty update changes nothing to review")

	text := "Changed"
	_, err = commander.Update(f.ctx, prompt.ID, domain.UpdatePromptParams{Text: &text})
	require.NoError(t, err)

	invalidated, err := store.BrandRepo().Get(f.ctx, f.brand.ID)
	require.NoError(t, err)
	require.Nil(t, invalidated.PromptsReviewedAt)

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
	require.NoError(t, store.VisibilityRepo().FinishReview(f.ctx, f.brand.ID, domain.ReviewCompetitors))

	created, err := commander.Create(f.ctx, f.brand.ID, domain.CreateCompetitorParams{Name: "Acme", Domain: "ACME.com"})
	require.NoError(t, err)

	brand, err := store.BrandRepo().Get(f.ctx, f.brand.ID)
	require.NoError(t, err)
	require.Nil(t, brand.CompetitorsReviewedAt, "adding a competitor asks for a new review")

	_, err = commander.Create(f.ctx, f.brand.ID, domain.CreateCompetitorParams{Name: "Copy", Domain: "acme.com"})

	var validation *domain.Error
	require.ErrorAs(t, err, &validation)
	require.Equal(t, []domain.FieldError{{Field: "domain", Code: domain.FieldTaken}}, validation.Fields)

	require.NoError(t, store.VisibilityRepo().FinishReview(f.ctx, f.brand.ID, domain.ReviewCompetitors))

	aliases := []string{"Acme Labs"}
	updated, err := commander.Update(f.ctx, created.ID, domain.UpdateCompetitorParams{Aliases: &aliases})
	require.NoError(t, err)
	require.Equal(t, aliases, updated.Aliases)

	stored, err := store.CompetitorRepo().Get(f.ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, aliases, stored.Aliases)
	require.Equal(t, domain.CompetitorSourceManual, stored.Source)

	brand, err = store.BrandRepo().Get(f.ctx, f.brand.ID)
	require.NoError(t, err)
	require.NotNil(t, brand.CompetitorsReviewedAt, "editing an identity keeps the review")

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
