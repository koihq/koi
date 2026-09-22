package koidatabase

import (
	"reflect"
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
	aliases := s.LookUpField("aliases")
	require.NotNil(t, aliases.Serializer)
	source := koidomain.Competitor{Aliases: []string{"Acme", "Acme AI"}}
	value, err := aliases.Serializer.Value(t.Context(), aliases, reflect.ValueOf(&source).Elem(), source.Aliases)
	require.NoError(t, err)
	require.Equal(t, `["Acme","Acme AI"]`, value)
	var destination koidomain.Competitor
	require.NoError(t, aliases.Serializer.Scan(t.Context(), aliases, reflect.ValueOf(&destination).Elem(), value))
	require.Equal(t, source.Aliases, destination.Aliases)
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
	prompts := NewGormPromptRepository(db)
	_, err = prompts.ListByBrandID(t.Context(), brandID)
	require.NoError(t, err)
	_, err = prompts.GetByIDAndBrandID(t.Context(), childID, brandID)
	require.NoError(t, err)
	require.ErrorIs(t, prompts.UpdateByIDAndBrandID(t.Context(), brandID, &koidomain.Prompt{BaseEntity: koidomain.BaseEntity{ID: childID}}), koidomain.ErrNotFound)
	require.ErrorIs(t, prompts.DeleteByIDAndBrandID(t.Context(), childID, brandID), koidomain.ErrNotFound)
	require.ErrorIs(t, prompts.ConfirmByIDAndBrandID(t.Context(), childID, brandID), koidomain.ErrNotFound)
	require.NoError(t, prompts.ConfirmAllByBrandID(t.Context(), brandID))

	competitors := NewGormCompetitorRepository(db)
	_, err = competitors.ListByBrandID(t.Context(), brandID)
	require.NoError(t, err)
	require.ErrorIs(t, competitors.UpdateByIDAndBrandID(t.Context(), brandID, &koidomain.Competitor{
		BaseEntity: koidomain.BaseEntity{ID: childID}, Aliases: []string{},
	}), koidomain.ErrNotFound)

	require.Len(t, queries, 8)
	for _, query := range queries {
		require.Contains(t, query, "brand_id =")
	}
	require.Contains(t, queries[0], "ORDER BY created_at ASC, id ASC")
	require.Contains(t, queries[6], "ORDER BY created_at ASC, id ASC")
	update := strings.ToLower(queries[2])
	require.Contains(t, update, "text")
	require.NotContains(t, update, "source")
	require.NotContains(t, update, "confirmed")
	competitorUpdate := strings.ToLower(queries[7])
	require.Contains(t, competitorUpdate, "aliases")
	require.NotContains(t, competitorUpdate, "source")
	require.NotContains(t, competitorUpdate, "confirmed")
}
