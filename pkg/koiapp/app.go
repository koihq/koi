package koiapp

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/alexedwards/scs/postgresstore"
	"github.com/alexedwards/scs/v2"
	"github.com/joho/godotenv"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverdatabasesql"
	"github.com/riverqueue/river/rivermigrate"

	"github.com/koihq/koi/pkg/koiapi"
	"github.com/koihq/koi/pkg/koidatabase"
	"github.com/koihq/koi/pkg/koidomain"
	"github.com/koihq/koi/pkg/koiproviders/logodev"
	"github.com/koihq/koi/pkg/koiproviders/openrouter"
)

type Config struct {
	HTTPAddr          string
	DatabaseURL       string
	SessionSecret     string
	OpenRouterAPIKey  string
	WorkerConcurrency int
	ProviderTimeout   time.Duration
	DBMaxOpenConns    int
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
	workerConcurrency, err := positiveIntEnv("KOI_WORKER_CONCURRENCY", 4)
	if err != nil {
		return Config{}, err
	}
	if workerConcurrency > river.QueueNumWorkersMax {
		return Config{}, fmt.Errorf("KOI_WORKER_CONCURRENCY must be at most %d", river.QueueNumWorkersMax)
	}
	dbMaxOpenConns, err := positiveIntEnv("KOI_DB_MAX_OPEN_CONNS", 12)
	if err != nil {
		return Config{}, err
	}
	providerTimeout := 3 * time.Minute
	if raw, ok := os.LookupEnv("KOI_PROVIDER_TIMEOUT"); ok {
		providerTimeout, err = time.ParseDuration(raw)
		if err != nil || providerTimeout <= 0 {
			return Config{}, fmt.Errorf("KOI_PROVIDER_TIMEOUT must be a positive duration")
		}
	}
	return Config{
		HTTPAddr:          httpAddr,
		DatabaseURL:       databaseURL,
		SessionSecret:     sessionSecret,
		OpenRouterAPIKey:  strings.TrimSpace(os.Getenv("KOI_OPENROUTER_API_KEY")),
		WorkerConcurrency: workerConcurrency,
		ProviderTimeout:   providerTimeout,
		DBMaxOpenConns:    dbMaxOpenConns,
	}, nil
}

func positiveIntEnv(key string, fallback int) (int, error) {
	raw, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}
	return n, nil
}

func requireEnv(key string) (string, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return "", fmt.Errorf("required env %s is not set", key)
	}
	return v, nil
}

type App struct {
	cfg      Config
	workers  *river.Workers
	provider *openrouter.Client
}

func NewKoiApp() (*App, error) {
	_ = godotenv.Load()
	initLogger()
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}
	return &App{cfg: cfg}, nil
}

func initLogger() {
	level := slog.LevelInfo
	switch strings.ToLower(os.Getenv("KOI_LOG_LEVEL")) {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	opts := &slog.HandlerOptions{Level: level}
	var h slog.Handler
	if strings.EqualFold(os.Getenv("KOI_LOG_FORMAT"), "json") {
		h = slog.NewJSONHandler(os.Stderr, opts)
	} else {
		h = slog.NewTextHandler(os.Stderr, opts)
	}
	slog.SetDefault(slog.New(h))
}

func (a *App) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	listener, err := net.Listen("tcp", a.cfg.HTTPAddr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	return a.run(ctx, listener)
}

func (a *App) run(ctx context.Context, listener net.Listener) error {
	defer listener.Close()
	db, err := koidatabase.Open(a.cfg.DatabaseURL)
	var sqlDB *sql.DB
	if err != nil {
		slog.Error("database unavailable; /health will report db=down", "err", err)
	} else {
		sqlDB, err = db.DB()
		if err != nil {
			return fmt.Errorf("sql db: %w", err)
		}
		defer sqlDB.Close()
		configurePool(sqlDB, a.cfg.DBMaxOpenConns)
		if err := koidatabase.Migrate(db); err != nil {
			return fmt.Errorf("migrate koi: %w", err)
		}
		if err := migrateRiver(ctx, sqlDB); err != nil {
			return fmt.Errorf("migrate river: %w", err)
		}
		if err := koidatabase.NewStore(db).ReconcileDraftRuns(ctx); err != nil {
			return fmt.Errorf("reconcile drafting runs: %w", err)
		}
	}
	sess := scs.New()
	sess.Lifetime = 30 * 24 * time.Hour
	sess.Cookie.Name = "koi_session"
	sess.Cookie.HttpOnly = true
	sess.Cookie.SameSite = http.SameSiteLaxMode
	sess.Cookie.Path = "/"
	sess.Cookie.Secure = false
	sess.Cookie.Persist = true
	if sqlDB != nil {
		sess.Store = postgresstore.New(sqlDB)
	}

	var searcher koidomain.CompanySearcher
	if key := strings.TrimSpace(os.Getenv("LOGODEV_SECRET_KEY")); key != "" {
		searcher = logodev.NewSearcher(key)
	}
	var riverClient *river.Client[*sql.Tx]
	workers := a.workers
	if sqlDB != nil && a.cfg.OpenRouterAPIKey != "" {
		if workers == nil {
			workers = river.NewWorkers()
			if a.provider == nil {
				a.provider = openrouter.New(a.cfg.OpenRouterAPIKey, a.cfg.ProviderTimeout)
			}
			river.AddWorker(workers, &draftWorker{store: koidatabase.NewStore(db), provider: a.provider})
		}
		config := &river.Config{
			Queues:  map[string]river.QueueConfig{"drafting": {MaxWorkers: a.cfg.WorkerConcurrency}},
			Workers: workers, SoftStopTimeout: 60 * time.Second,
		}
		riverClient, err = river.NewClient(riverdatabasesql.New(sqlDB), config)
		if err != nil {
			return fmt.Errorf("create river client: %w", err)
		}
		if err := riverClient.Start(context.Background()); err != nil {
			return fmt.Errorf("start river client: %w", err)
		}
	}
	store := koidatabase.NewStore(db, riverClient)
	server := &http.Server{Handler: koiapi.NewServer(store, sess, searcher, a.cfg.OpenRouterAPIKey != "" && riverClient != nil)}
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(listener) }()

	slog.Info("koi listening", "addr", listener.Addr().String())
	var runErr error
	select {
	case <-ctx.Done():
	case err := <-serveErr:
		if !errors.Is(err, http.ErrServerClosed) {
			runErr = err
		}
	}
	httpCtx, cancelHTTP := context.WithTimeout(context.Background(), 10*time.Second)
	if err := server.Shutdown(httpCtx); err != nil {
		runErr = errors.Join(runErr, fmt.Errorf("shutdown http: %w", err))
		_ = server.Close()
	}
	cancelHTTP()
	if riverClient != nil {
		riverCtx, cancelRiver := context.WithTimeout(context.Background(), 60*time.Second)
		if err := riverClient.Stop(riverCtx); err != nil {
			runErr = errors.Join(runErr, fmt.Errorf("stop river: %w", err))
		}
		cancelRiver()
	}
	return runErr
}

func configurePool(db *sql.DB, maxOpen int) {
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(min(4, maxOpen))
}

func migrateRiver(ctx context.Context, db *sql.DB) error {
	migrator, err := rivermigrate.New(riverdatabasesql.New(db), nil)
	if err != nil {
		return err
	}
	_, err = migrator.Migrate(ctx, rivermigrate.DirectionUp, nil)
	return err
}
