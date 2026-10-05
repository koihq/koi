package testdb

import (
	"database/sql"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
)

func databaseURL(t *testing.T) string {
	t.Helper()
	if dsn := os.Getenv("KOI_DATABASE_URL"); dsn != "" {
		return dsn
	}
	_, file, _, _ := runtime.Caller(0)
	values, err := godotenv.Read(filepath.Join(filepath.Dir(file), "..", "..", ".env"))
	if err != nil {
		t.Fatalf("KOI_DATABASE_URL is unset and repo .env could not be read: %v", err)
	}
	if values["KOI_DATABASE_URL"] == "" {
		t.Fatal("KOI_DATABASE_URL is unset in the environment and repo .env")
	}
	return values["KOI_DATABASE_URL"]
}

// URL creates a schema in the existing test database. Callers own migrations.
func URL(t *testing.T) string {
	t.Helper()
	dsn := databaseURL(t)
	admin, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	if err := admin.PingContext(t.Context()); err != nil {
		_ = admin.Close()
		t.Fatal(err)
	}
	schema := "koi_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := admin.ExecContext(t.Context(), "CREATE SCHEMA "+schema); err != nil {
		_ = admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec("DROP SCHEMA " + schema + " CASCADE"); err != nil {
			t.Error(err)
		}
		if err := admin.Close(); err != nil {
			t.Error(err)
		}
	})
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	query := u.Query()
	query.Set("search_path", schema)
	u.RawQuery = query.Encode()
	return u.String()
}
