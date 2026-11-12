// Command server runs Koi for the Playwright end-to-end suite: the real binary's app, embedded UI
// and Postgres, in a schema of its own, with every optional service turned off. It seeds the demo
// account, writes the database URL for tests that run `koi admin`, and drops the schema on exit.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"

	"github.com/koihq/koi/pkg/app"
)

const port = "8090"

// Optional integrations stay off so the suite proves the self-hosted core works without them.
var optional = []string{
	"KOI_CLORO_API_KEY",
	"KOI_LIVE_COLLECTION",
	"KOI_OPENROUTER_API_KEY",
	"KOI_RESEND_API_KEY",
	"KOI_EMAIL_FROM",
	"KOI_STRIPE_SECRET_KEY",
	"KOI_STRIPE_WEBHOOK_SECRET",
	"KOI_STRIPE_PRICE_ID",
	"KOI_POSTHOG_KEY",
	"KOI_POSTHOG_HOST",
	"KOI_MCP_PUBLIC_URL",
	"KOI_CLIENT_IP_HEADER",
	"LOGODEV_SECRET_KEY",
}

func main() {
	if err := run(); err != nil {
		slog.Error("e2e server exited", "err", err)
		os.Exit(1)
	}
}

func run() error {
	_ = godotenv.Load()

	base := os.Getenv("KOI_DATABASE_URL")
	if base == "" {
		return fmt.Errorf("KOI_DATABASE_URL is required")
	}

	schema := "koi_e2e_" + strings.ReplaceAll(uuid.NewString(), "-", "")

	admin, err := sql.Open("pgx", base)
	if err != nil {
		return err
	}
	defer admin.Close()

	if _, err := admin.Exec("CREATE SCHEMA " + schema); err != nil {
		return fmt.Errorf("create schema: %w", err)
	}
	defer func() {
		if _, err := admin.Exec("DROP SCHEMA " + schema + " CASCADE"); err != nil {
			slog.Error("drop e2e schema", "schema", schema, "err", err)
		}
	}()

	scoped, err := withSearchPath(base, schema)
	if err != nil {
		return err
	}

	if err := configure(scoped); err != nil {
		return err
	}

	application, err := app.New()
	if err != nil {
		return err
	}

	done := make(chan error, 1)
	go func() { done <- application.Run() }()

	if err := waitHealthy(done); err != nil {
		return err
	}

	if err := app.RunAdmin(context.Background(), []string{"seed-demo"}, strings.NewReader(""), io.Discard); err != nil {
		return fmt.Errorf("seed demo: %w", err)
	}

	// Playwright waits for this line, so tests start only after the demo account exists.
	fmt.Println("koi e2e ready on http://127.0.0.1:" + port)

	return <-done
}

func withSearchPath(raw, schema string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", err
	}

	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()

	return parsed.String(), nil
}

// configure sets the environment before the app loads `.env`, which never overrides a set value.
func configure(databaseURL string) error {
	values := map[string]string{
		"KOI_DATABASE_URL":    databaseURL,
		"KOI_PORT":            ":" + port,
		"KOI_PUBLIC_URL":      "http://127.0.0.1:" + port,
		"KOI_DEPLOYMENT_MODE": "selfhosted",
		"KOI_ALLOW_SIGNUP":    "true",
		"KOI_LOG_LEVEL":       "warn",
	}
	for _, key := range optional {
		values[key] = ""
	}

	for key, value := range values {
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}

	return writeDatabaseURL(databaseURL)
}

// writeDatabaseURL lets tests run `koi admin` against the same schema.
func writeDatabaseURL(databaseURL string) error {
	path := filepath.Join("ui", ".e2e", "database-url")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	return os.WriteFile(path, []byte(databaseURL), 0o600)
}

func waitHealthy(done <-chan error) error {
	client := http.Client{Timeout: time.Second}
	deadline := time.Now().Add(60 * time.Second)

	for time.Now().Before(deadline) {
		select {
		case err := <-done:
			return fmt.Errorf("app stopped before it was healthy: %w", err)
		default:
		}

		resp, err := client.Get("http://127.0.0.1:" + port + "/health")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}

		time.Sleep(200 * time.Millisecond)
	}

	return fmt.Errorf("app was not healthy within 60s")
}
