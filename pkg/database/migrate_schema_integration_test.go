package database

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/koihq/koi/internal/testdb"
)

// The migrations and the GORM structs must describe the same schema. When this fails, add a
// migration for the struct change: `make migration name=...`.
func TestMigrationsMatchModels(t *testing.T) {
	migrated, err := Open(testdb.URL(t))
	require.NoError(t, err)
	require.NoError(t, Migrate(migrated))

	modeled, err := Open(testdb.URL(t))
	require.NoError(t, err)
	require.NoError(t, modeled.AutoMigrate(models()...))

	require.Equal(t, describeSchema(t, modeled), describeSchema(t, migrated))
}

func TestMigrateIsIdempotent(t *testing.T) {
	db, err := Open(testdb.URL(t))
	require.NoError(t, err)

	require.NoError(t, Migrate(db))
	before := describeSchema(t, db)

	require.NoError(t, Migrate(db))
	require.Equal(t, before, describeSchema(t, db))

	var version int64
	require.NoError(t, db.Raw(`SELECT max(version_id) FROM goose_db_version`).Scan(&version).Error)
	require.Equal(t, int64(1), version)
}

type schemaDescription struct {
	Columns     []string
	Indexes     []string
	Constraints []string
}

var schemaPrefix = regexp.MustCompile(`koi_test_[0-9a-f]+\.`)

// describeSchema lists the current schema's columns, indexes and constraints by name, without
// column order or the schema name. goose's own version table is left out.
func describeSchema(t *testing.T, db *gorm.DB) schemaDescription {
	t.Helper()

	var description schemaDescription

	require.NoError(t, db.Raw(`SELECT concat_ws(' ', table_name, column_name, data_type, udt_name,
			coalesce(character_maximum_length::text, '-'), is_nullable, coalesce(column_default, '-'))
		FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name <> 'goose_db_version'
		ORDER BY table_name, column_name`).Scan(&description.Columns).Error)

	require.NoError(t, db.Raw(`SELECT concat_ws(' ', tablename, indexname, indexdef)
		FROM pg_indexes
		WHERE schemaname = current_schema() AND tablename <> 'goose_db_version'
		ORDER BY tablename, indexname`).Scan(&description.Indexes).Error)

	require.NoError(t, db.Raw(`SELECT concat_ws(' ', rel.relname, con.conname, pg_get_constraintdef(con.oid))
		FROM pg_constraint con
		JOIN pg_class rel ON rel.oid = con.conrelid
		WHERE con.connamespace = current_schema()::regnamespace AND rel.relname <> 'goose_db_version'
		ORDER BY rel.relname, con.conname`).Scan(&description.Constraints).Error)

	for i, index := range description.Indexes {
		description.Indexes[i] = schemaPrefix.ReplaceAllString(index, "")
	}

	for i, constraint := range description.Constraints {
		description.Constraints[i] = schemaPrefix.ReplaceAllString(constraint, "")
	}

	return description
}
