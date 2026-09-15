package koiapp

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/alexedwards/scs/postgresstore"
	"github.com/alexedwards/scs/v2"
	"github.com/joho/godotenv"

	"github.com/koihq/koi/pkg/koiapi"
	"github.com/koihq/koi/pkg/koidatabase"
)

type Config struct {
	HTTPAddr      string
	DatabaseURL   string
	SessionSecret string
}

func LoadConfig() (Config, error) {
	_ = godotenv.Load()
	httpAddr, err := requireEnv("KOI_PORT")
	if err != nil {
		return Config{}, err
	}
	databaseURL, err := requireEnv("KOI_DATABASE_URL")
	if err != nil {
		return Config{}, err
	}
	sessionSecret, err := requireEnv("KOI_SESSION_SECRET")
	if err != nil {
		return Config{}, err
	}
	return Config{
		HTTPAddr:      httpAddr,
		DatabaseURL:   databaseURL,
		SessionSecret: sessionSecret,
	}, nil
}

func requireEnv(key string) (string, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return "", fmt.Errorf("required env %s is not set", key)
	}
	return v, nil
}

type App struct {
	cfg Config
}

func NewKoiApp() (*App, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}
	return &App{cfg: cfg}, nil
}

func (a *App) Run() error {
	db, err := koidatabase.Open(a.cfg.DatabaseURL)
	if err != nil {
		slog.Error("database unavailable; /health will report db=down", "err", err)
	} else if err := koidatabase.Migrate(db); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	store := koidatabase.NewStore(db)

	sess := scs.New()
	sess.Lifetime = 30 * 24 * time.Hour
	sess.Cookie.Name = "koi_session"
	sess.Cookie.HttpOnly = true
	sess.Cookie.SameSite = http.SameSiteLaxMode
	sess.Cookie.Path = "/"
	sess.Cookie.Secure = false
	sess.Cookie.Persist = true
	if db != nil {
		sqlDB, err := db.DB()
		if err != nil {
			return fmt.Errorf("sql db: %w", err)
		}
		sess.Store = postgresstore.New(sqlDB)
	}

	server := koiapi.NewServer(store, sess)

	slog.Info("koi listening", "addr", a.cfg.HTTPAddr)
	return http.ListenAndServe(a.cfg.HTTPAddr, server)
}
