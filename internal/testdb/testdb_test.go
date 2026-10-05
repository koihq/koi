package testdb

import (
	"database/sql"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestURLIsolatesAndCleansUpSchema(t *testing.T) {
	admin, err := sql.Open("pgx", databaseURL(t))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, admin.Close()) })
	var first string
	for i := range 2 {
		var schema string
		t.Run(string(rune('a'+i)), func(t *testing.T) {
			scoped := URL(t)
			u, err := url.Parse(scoped)
			require.NoError(t, err)
			schema = u.Query().Get("search_path")
			require.NotEmpty(t, schema)
			if first != "" {
				require.NotEqual(t, first, schema)
			}
			db, err := sql.Open("pgx", scoped)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, db.Close()) })
			var current string
			require.NoError(t, db.QueryRow("SELECT current_schema()").Scan(&current))
			require.Equal(t, schema, current)
		})
		var remaining int
		require.NoError(t, admin.QueryRow("SELECT count(*) FROM pg_namespace WHERE nspname = $1", schema).Scan(&remaining))
		require.Zero(t, remaining)
		first = schema
	}
}
