package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/koihq/koi/internal/demoseed"
	"github.com/koihq/koi/pkg/koidatabase"
)

func main() {
	reset := flag.Bool("reset", false, "recreate only the three demo brands using today's dates")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, *reset); err != nil {
		slog.Error("demo seed failed", "err", err)
		os.Exit(1)
	}

	fmt.Printf("Demo ready: http://localhost:5173\nEmail: %s\nPassword: %s\n", demoseed.Email, demoseed.Password)
	fmt.Println("Brands (US): Koi · Storico, Koi · Vuoto, Koi · Prima raccolta")
	fmt.Println("Answers and metrics are simulated. No providers were contacted or jobs queued.")
	fmt.Println("Existing scenarios are preserved; make seed-demo-reset refreshes the demo dates.")
}

func run(ctx context.Context, reset bool) error {
	_ = godotenv.Load()

	mode := strings.TrimSpace(os.Getenv("KOI_DEPLOYMENT_MODE"))
	if mode != "" && mode != "selfhosted" {
		return fmt.Errorf("demo seed requires KOI_DEPLOYMENT_MODE=selfhosted")
	}
	dsn := strings.TrimSpace(os.Getenv("KOI_DATABASE_URL"))
	if dsn == "" {
		return fmt.Errorf("KOI_DATABASE_URL is required")
	}

	db, err := koidatabase.Open(dsn)
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	if err := koidatabase.Migrate(db.WithContext(ctx)); err != nil {
		return fmt.Errorf("migrate demo database: %w", err)
	}

	return demoseed.Run(ctx, db, time.Now(), reset)
}
