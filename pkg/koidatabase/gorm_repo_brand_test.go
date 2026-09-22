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

func TestBrandSchema(t *testing.T) {
	s, err := schema.Parse(&koidomain.Brand{}, &sync.Map{}, schema.NamingStrategy{})
	require.NoError(t, err)
	require.Equal(t, "brands", s.Table)
	require.Equal(t, "github_repo", s.LookUpField("github_repo").DBName)
	require.Equal(t, "description", s.LookUpField("description").DBName)
	require.Equal(t, "gen_random_uuid()", s.LookUpField("id").DefaultValue)

	var ownerIndex *schema.Index
	for _, index := range s.ParseIndexes() {
		if index.Name == "idx_brands_user_id" {
			ownerIndex = index
			break
		}
	}
	require.NotNil(t, ownerIndex)
	require.Empty(t, ownerIndex.Class)
	require.Equal(t, "user_id", ownerIndex.Fields[0].Field.DBName)
}

func TestBrandQueriesScopeAndLock(t *testing.T) {
	db, err := gorm.Open(postgres.Open("host=localhost user=koi dbname=koi sslmode=disable"), &gorm.Config{
		DryRun: true, DisableAutomaticPing: true, SkipDefaultTransaction: true,
	})
	require.NoError(t, err)
	var queries []string
	var args [][]any
	require.NoError(t, db.Callback().Query().After("gorm:query").Register("test:capture", func(tx *gorm.DB) {
		queries = append(queries, tx.Statement.SQL.String())
		args = append(args, append([]any(nil), tx.Statement.Vars...))
	}))
	require.NoError(t, db.Callback().Update().After("gorm:update").Register("test:capture_update", func(tx *gorm.DB) {
		queries = append(queries, tx.Statement.SQL.String())
		args = append(args, append([]any(nil), tx.Statement.Vars...))
	}))

	id := uuid.New()
	require.NoError(t, NewGormUserRepository(db).LockForUpdate(t.Context(), id))
	_, err = NewGormBrandRepository(db).CountByUserID(t.Context(), id)
	require.NoError(t, err)
	_, err = NewGormBrandRepository(db).ListByUserID(t.Context(), id, 10, 20)
	require.NoError(t, err)
	_, err = NewGormBrandRepository(db).GetByIDAndUserID(t.Context(), uuid.New(), id)
	require.NoError(t, err)
	require.ErrorIs(t, NewGormBrandRepository(db).UpdateByIDAndUserID(t.Context(), id, &koidomain.Brand{
		BaseEntity: koidomain.BaseEntity{ID: uuid.New()}, UserID: uuid.New(), Name: "Koi",
	}), koidomain.ErrNotFound)
	require.Len(t, queries, 5)
	require.Contains(t, strings.ToUpper(queries[0]), "FOR UPDATE")
	require.Contains(t, queries[0], "id =")
	for _, query := range queries[1:] {
		require.Contains(t, query, "user_id =")
	}
	require.Contains(t, queries[2], "ORDER BY created_at DESC, id DESC")
	require.Contains(t, queries[2], "LIMIT")
	require.Contains(t, queries[2], "OFFSET")
	require.Contains(t, args[2], 10)
	require.Contains(t, args[2], 20)
	require.Contains(t, queries[3], "id =")
	require.Contains(t, queries[4], "id =")
	for _, queryArgs := range args[1:] {
		require.Contains(t, queryArgs, id)
	}
}
