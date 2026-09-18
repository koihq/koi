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
		DryRun: true, DisableAutomaticPing: true,
	})
	require.NoError(t, err)
	var queries []string
	require.NoError(t, db.Callback().Query().After("gorm:query").Register("test:capture", func(tx *gorm.DB) {
		queries = append(queries, tx.Statement.SQL.String())
	}))

	id := uuid.New()
	require.NoError(t, NewGormUserRepository(db).LockForUpdate(t.Context(), id))
	_, err = NewGormBrandRepository(db).ListByUserID(t.Context(), id)
	require.NoError(t, err)
	require.Len(t, queries, 2)
	require.Contains(t, strings.ToUpper(queries[0]), "FOR UPDATE")
	require.Contains(t, queries[0], "id =")
	require.Contains(t, queries[1], "user_id =")
}
