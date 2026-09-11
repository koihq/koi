package koiapp

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"github.com/koihq/koi/pkg/koiapi"
	"github.com/koihq/koi/pkg/koidatabase"
)

type Config struct {
	HTTPAddr    string
	DatabaseURL string
}

func LoadConfig() Config {
	_ = godotenv.Load()
	return Config{
		HTTPAddr:    env("KOI_HTTP_ADDR", ":8080"),
		DatabaseURL: env("KOI_DATABASE_URL", "postgres://koi:koi@localhost:5432/koi?sslmode=disable"),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

type App struct {
	cfg Config
}

func NewKoiApp() *App {
	return &App{cfg: LoadConfig()}
}

func (a *App) Run() error {
	db, err := koidatabase.Open(a.cfg.DatabaseURL)
	if err != nil {
		slog.Error("database unavailable; /health will report db=down", "err", err)
	}
	store := koidatabase.NewStore(db)
	server := koiapi.NewServer(store)

	slog.Info("koi listening", "addr", a.cfg.HTTPAddr)
	return http.ListenAndServe(a.cfg.HTTPAddr, server)
}
