package koidatabase

import (
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"github.com/koihq/koi/pkg/koidomain"
)

func TestPromptAndCompetitorSchema(t *testing.T) {
	for _, entity := range []struct {
		value any
		table string
	}{
		{&koidomain.Prompt{}, "prompts"},
		{&koidomain.Competitor{}, "competitors"},
	} {
		s, err := schema.Parse(entity.value, &sync.Map{}, schema.NamingStrategy{})
		require.NoError(t, err)
		require.Equal(t, entity.table, s.Table)
		require.Equal(t, "brand_id", s.LookUpField("brand_id").DBName)
		relation := s.Relationships.Relations["Brand"]
		require.NotNil(t, relation)
		require.Equal(t, "CASCADE", relation.ParseConstraint().OnDelete)
	}

	s, err := schema.Parse(&koidomain.Competitor{}, &sync.Map{}, schema.NamingStrategy{})
	require.NoError(t, err)
	require.Nil(t, s.LookUpField("aliases"))
	require.NotNil(t, s.LookUpField("github_repo"))
	require.Empty(t, s.ParseIndexes()) // Partial uniqueness is installed explicitly by Migrate.
}

func TestPromptAndCompetitorQueriesAreBrandScoped(t *testing.T) {
	db, err := gorm.Open(postgres.Open("host=localhost user=koi dbname=koi sslmode=disable"), &gorm.Config{
		DryRun: true, DisableAutomaticPing: true, SkipDefaultTransaction: true,
	})
	require.NoError(t, err)
	var queries []string
	require.NoError(t, db.Callback().Query().After("gorm:query").Register("test:capture_child_query", func(tx *gorm.DB) {
		queries = append(queries, tx.Statement.SQL.String())
	}))
	require.NoError(t, db.Callback().Update().After("gorm:update").Register("test:capture_child_update", func(tx *gorm.DB) {
		queries = append(queries, tx.Statement.SQL.String())
	}))
	require.NoError(t, db.Callback().Delete().After("gorm:delete").Register("test:capture_child_delete", func(tx *gorm.DB) {
		queries = append(queries, tx.Statement.SQL.String())
	}))

	brandID, childID := uuid.New(), uuid.New()
	ctx := identityCtx(t, uuid.New())
	unconfirmed := false
	prompts := NewGormPromptRepository(db)
	_, err = prompts.List(ctx, koidomain.PromptFilter{
		BrandID: brandID, Query: "100%_Koi", Confirmed: &unconfirmed, Page: koidomain.PageParams{Page: 2, PageSize: 10},
	})
	require.NoError(t, err)
	_, err = prompts.Get(ctx, koidomain.PromptFilter{ID: childID, BrandID: brandID})
	require.NoError(t, err)
	text, confirmed := "Updated", true
	filter := koidomain.PromptFilter{ID: childID, BrandID: brandID}
	require.ErrorIs(t, prompts.Update(ctx, filter, koidomain.PromptUpdate{Text: &text}), koidomain.ErrNotFound)
	require.ErrorIs(t, prompts.Delete(ctx, filter), koidomain.ErrNotFound)
	require.ErrorIs(t, prompts.Update(ctx, filter, koidomain.PromptUpdate{Confirmed: &confirmed}), koidomain.ErrNotFound)
	require.NoError(t, prompts.Update(ctx, koidomain.PromptFilter{
		BrandID: brandID, Confirmed: &unconfirmed,
	}, koidomain.PromptUpdate{Confirmed: &confirmed}))

	competitors := NewGormCompetitorRepository(db)
	_, err = competitors.List(ctx, koidomain.CompetitorFilter{
		BrandID: brandID, Query: "Acme", Page: koidomain.PageParams{Page: 1, PageSize: 10},
	})
	require.NoError(t, err)
	_, err = competitors.Get(ctx, koidomain.CompetitorFilter{ID: childID, BrandID: brandID})
	require.NoError(t, err)
	competitorConfirmed := true
	require.ErrorIs(t, competitors.Update(ctx, koidomain.CompetitorFilter{
		ID: childID, BrandID: brandID,
	}, koidomain.CompetitorUpdate{Confirmed: &competitorConfirmed}), koidomain.ErrNotFound)

	require.Len(t, queries, 9)
	for _, query := range queries {
		require.Contains(t, query, "brand_id =")
		require.Contains(t, query, "brands.user_id =")
	}
	require.Contains(t, queries[0], "confirmed =")
	require.Contains(t, queries[0], "text ILIKE")
	require.NotContains(t, queries[0], "100%_Koi")
	require.Equal(t, `%100!%!_Koi%`, likePattern("100%_Koi"))
	require.Contains(t, queries[6], "name ILIKE")
	require.Contains(t, queries[6], "domain ILIKE")
	require.NotContains(t, queries[6], "aliases")
	update := strings.ToLower(queries[2])
	require.Contains(t, update, "text")
	require.NotContains(t, update, "source")
	require.NotContains(t, update, "confirmed")
	competitorUpdate := strings.ToLower(queries[8])
	require.Contains(t, competitorUpdate, "confirmed")
	require.NotContains(t, competitorUpdate, "source")

	listSQL := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		var items []koidomain.Prompt
		return applyPromptFilter(tx, koidomain.PromptFilter{BrandID: brandID}).
			Order("created_at ASC, id ASC").Limit(10).Offset(10).Find(&items)
	})
	require.Contains(t, listSQL, "ORDER BY created_at ASC, id ASC")
	require.Contains(t, listSQL, "LIMIT 10 OFFSET 10")
}

func TestProtectedRepositoriesFailWithoutIdentity(t *testing.T) {
	db, err := gorm.Open(postgres.Open("host=localhost user=koi dbname=koi sslmode=disable"), &gorm.Config{
		DryRun: true, DisableAutomaticPing: true, SkipDefaultTransaction: true,
	})
	require.NoError(t, err)
	ctx := t.Context()
	brandID := uuid.New()
	brands := NewGormBrandRepository(db)
	_, err = brands.Count(ctx)
	require.ErrorIs(t, err, koidomain.ErrUnauthorized)
	_, err = brands.List(ctx, koidomain.PageParams{Page: 1, PageSize: 10})
	require.ErrorIs(t, err, koidomain.ErrUnauthorized)
	_, err = brands.Get(ctx, brandID)
	require.ErrorIs(t, err, koidomain.ErrUnauthorized)
	require.ErrorIs(t, brands.Update(ctx, &koidomain.Brand{BaseEntity: koidomain.BaseEntity{ID: brandID}}), koidomain.ErrUnauthorized)
	require.ErrorIs(t, brands.Create(ctx, &koidomain.Brand{}), koidomain.ErrUnauthorized)

	prompts := NewGormPromptRepository(db)
	_, err = prompts.Get(ctx, koidomain.PromptFilter{BrandID: brandID})
	require.ErrorIs(t, err, koidomain.ErrUnauthorized)
	_, err = prompts.List(ctx, koidomain.PromptFilter{BrandID: brandID, Page: koidomain.PageParams{Page: 1, PageSize: 10}})
	require.ErrorIs(t, err, koidomain.ErrUnauthorized)
	require.ErrorIs(t, prompts.Update(ctx, koidomain.PromptFilter{BrandID: brandID}, koidomain.PromptUpdate{}), koidomain.ErrUnauthorized)
	require.ErrorIs(t, prompts.Delete(ctx, koidomain.PromptFilter{BrandID: brandID}), koidomain.ErrUnauthorized)
	require.ErrorIs(t, prompts.Create(ctx, &koidomain.Prompt{BrandID: brandID}), koidomain.ErrUnauthorized)

	competitors := NewGormCompetitorRepository(db)
	_, err = competitors.Get(ctx, koidomain.CompetitorFilter{BrandID: brandID})
	require.ErrorIs(t, err, koidomain.ErrUnauthorized)
	_, err = competitors.List(ctx, koidomain.CompetitorFilter{BrandID: brandID, Page: koidomain.PageParams{Page: 1, PageSize: 10}})
	require.ErrorIs(t, err, koidomain.ErrUnauthorized)
	require.ErrorIs(t, competitors.Update(ctx, koidomain.CompetitorFilter{BrandID: brandID}, koidomain.CompetitorUpdate{}), koidomain.ErrUnauthorized)
	require.ErrorIs(t, competitors.Delete(ctx, koidomain.CompetitorFilter{BrandID: brandID}), koidomain.ErrUnauthorized)
	require.ErrorIs(t, competitors.Create(ctx, &koidomain.Competitor{BrandID: brandID}), koidomain.ErrUnauthorized)
}
