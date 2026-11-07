package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
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

	"github.com/koihq/koi/pkg/api"
	"github.com/koihq/koi/pkg/database"
	"github.com/koihq/koi/pkg/domain"
	"github.com/koihq/koi/pkg/providers/cloro"
	"github.com/koihq/koi/pkg/providers/logodev"
	"github.com/koihq/koi/pkg/providers/openrouter"
	resendprovider "github.com/koihq/koi/pkg/providers/resend"
	"github.com/koihq/koi/pkg/providers/sitelookup"
	"github.com/koihq/koi/pkg/providers/stripe"
)

type Config struct {
	DeploymentMode      string
	HTTPAddr            string
	DatabaseURL         string
	SessionLifetime     time.Duration
	OpenRouterAPIKey    string
	CloroAPIKey         string
	ResendAPIKey        string
	EmailFrom           string
	WorkerConcurrency   int
	ProviderTimeout     time.Duration
	DBMaxOpenConns      int
	StripeSecretKey     string
	StripeWebhookSecret string
	StripePriceID       string
	PublicURL           string
	FirstLookDailyLimit int
	AllowSignup         bool
	ClientIPHeader      string
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

	sessionLifetime := 8 * time.Hour
	if raw, ok := os.LookupEnv("KOI_SESSION_LIFETIME"); ok {
		sessionLifetime, err = time.ParseDuration(raw)
		if err != nil || sessionLifetime <= 0 {
			return Config{}, fmt.Errorf("KOI_SESSION_LIFETIME must be a positive duration")
		}
	}

	mode := strings.TrimSpace(os.Getenv("KOI_DEPLOYMENT_MODE"))
	if mode == "" {
		mode = "selfhosted"
	}
	if mode != "selfhosted" && mode != "cloud" {
		return Config{}, fmt.Errorf("KOI_DEPLOYMENT_MODE must be selfhosted or cloud")
	}
	cloroAPIKey := strings.TrimSpace(os.Getenv("KOI_CLORO_API_KEY"))
	resendAPIKey := strings.TrimSpace(os.Getenv("KOI_RESEND_API_KEY"))
	emailFrom := strings.TrimSpace(os.Getenv("KOI_EMAIL_FROM"))
	if (resendAPIKey == "") != (emailFrom == "") {
		return Config{}, fmt.Errorf("KOI_RESEND_API_KEY and KOI_EMAIL_FROM must be set together")
	}
	stripeSecretKey := strings.TrimSpace(os.Getenv("KOI_STRIPE_SECRET_KEY"))
	stripeWebhookSecret := strings.TrimSpace(os.Getenv("KOI_STRIPE_WEBHOOK_SECRET"))
	stripePriceID := strings.TrimSpace(os.Getenv("KOI_STRIPE_PRICE_ID"))
	publicURL := strings.TrimSpace(os.Getenv("KOI_PUBLIC_URL"))
	if mode == "cloud" && (cloroAPIKey == "" || stripeSecretKey == "" || stripeWebhookSecret == "" ||
		stripePriceID == "" || publicURL == "" || resendAPIKey == "") {
		return Config{}, fmt.Errorf("cloud mode requires KOI_CLORO_API_KEY, KOI_STRIPE_SECRET_KEY, KOI_STRIPE_WEBHOOK_SECRET, KOI_STRIPE_PRICE_ID, KOI_PUBLIC_URL, KOI_RESEND_API_KEY and KOI_EMAIL_FROM")
	}
	if publicURL != "" {
		publicURL = strings.TrimRight(publicURL, "/")
		parsed, err := url.Parse(publicURL)
		if err != nil || parsed.Host == "" || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" ||
			(parsed.Scheme != "https" && !(parsed.Scheme == "http" && (parsed.Hostname() == "localhost" || parsed.Hostname() == "127.0.0.1"))) {
			return Config{}, fmt.Errorf("KOI_PUBLIC_URL must be an HTTPS origin (HTTP is allowed for localhost)")
		}
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

	firstLookDailyLimit, err := positiveIntEnv("KOI_FIRST_LOOK_DAILY_LIMIT", domain.DefaultFirstLookDailyLimit)
	if err != nil {
		return Config{}, err
	}

	// Self-hosted instances spend their owner's Cloro key, so only the first user may join by default.
	allowSignup := mode == "cloud"
	if raw := strings.TrimSpace(os.Getenv("KOI_ALLOW_SIGNUP")); raw != "" {
		allowSignup, err = strconv.ParseBool(raw)
		if err != nil {
			return Config{}, fmt.Errorf("KOI_ALLOW_SIGNUP must be true or false")
		}
	}

	providerTimeout := 3 * time.Minute
	if raw, ok := os.LookupEnv("KOI_PROVIDER_TIMEOUT"); ok {
		providerTimeout, err = time.ParseDuration(raw)
		if err != nil || providerTimeout <= 0 {
			return Config{}, fmt.Errorf("KOI_PROVIDER_TIMEOUT must be a positive duration")
		}
	}
	if level := strings.ToLower(strings.TrimSpace(os.Getenv("KOI_LOG_LEVEL"))); level != "" &&
		level != "debug" && level != "info" && level != "warn" && level != "warning" && level != "error" {
		return Config{}, fmt.Errorf("KOI_LOG_LEVEL must be debug, info, warn, warning or error")
	}
	if format := strings.ToLower(strings.TrimSpace(os.Getenv("KOI_LOG_FORMAT"))); format != "" && format != "text" && format != "json" {
		return Config{}, fmt.Errorf("KOI_LOG_FORMAT must be text or json")
	}

	return Config{
		DeploymentMode:      mode,
		HTTPAddr:            httpAddr,
		DatabaseURL:         databaseURL,
		SessionLifetime:     sessionLifetime,
		OpenRouterAPIKey:    strings.TrimSpace(os.Getenv("KOI_OPENROUTER_API_KEY")),
		CloroAPIKey:         cloroAPIKey,
		ResendAPIKey:        resendAPIKey,
		EmailFrom:           emailFrom,
		WorkerConcurrency:   workerConcurrency,
		ProviderTimeout:     providerTimeout,
		DBMaxOpenConns:      dbMaxOpenConns,
		StripeSecretKey:     stripeSecretKey,
		StripeWebhookSecret: stripeWebhookSecret,
		StripePriceID:       stripePriceID,
		PublicURL:           publicURL,
		FirstLookDailyLimit: firstLookDailyLimit,
		AllowSignup:         allowSignup,
		ClientIPHeader:      strings.TrimSpace(os.Getenv("KOI_CLIENT_IP_HEADER")),
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
	if !ok || strings.TrimSpace(v) == "" {
		return "", fmt.Errorf("required env %s is not set", key)
	}

	return v, nil
}

type App struct {
	cfg           Config
	workers       *river.Workers
	provider      *openrouter.Client
	cloroProvider *cloro.Client
}

func New() (*App, error) {
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

	switch strings.ToLower(strings.TrimSpace(os.Getenv("KOI_LOG_LEVEL"))) {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	opts := &slog.HandlerOptions{Level: level}
	var h slog.Handler
	if strings.EqualFold(strings.TrimSpace(os.Getenv("KOI_LOG_FORMAT")), "json") {
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
	db, err := database.Open(a.cfg.DatabaseURL)
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
		if err := database.Migrate(db); err != nil {
			return fmt.Errorf("migrate koi: %w", err)
		}

		if err := migrateRiver(ctx, sqlDB); err != nil {
			return fmt.Errorf("migrate river: %w", err)
		}

		if err := database.NewStore(db).WithCloudMode(a.cfg.DeploymentMode == "cloud").ReconcileGenerationRuns(ctx); err != nil {
			return fmt.Errorf("reconcile generation runs: %w", err)
		}

		if err := database.NewStore(db).WithCloudMode(a.cfg.DeploymentMode == "cloud").ReconcileVisibilityRuns(ctx); err != nil {
			return fmt.Errorf("reconcile visibility runs: %w", err)
		}
	}

	sess := scs.New()
	sess.Lifetime = a.cfg.SessionLifetime
	sess.Cookie.Name = "koi_session"
	sess.Cookie.HttpOnly = true
	sess.Cookie.SameSite = http.SameSiteLaxMode
	sess.Cookie.Path = "/"
	sess.Cookie.Secure = strings.HasPrefix(a.cfg.PublicURL, "https://")
	sess.Cookie.Persist = true
	if sqlDB != nil {
		sess.Store = postgresstore.New(sqlDB)
	}

	var searcher domain.CompanySearcher
	if key := strings.TrimSpace(os.Getenv("LOGODEV_SECRET_KEY")); key != "" {
		searcher = logodev.NewSearcher(key)
	}
	var riverClient *river.Client[*sql.Tx]
	workers := a.workers
	// Workers start collection and emails, so their stores get the River client once it exists.
	var workerStores []*database.Store
	if sqlDB != nil && (a.cfg.OpenRouterAPIKey != "" || a.cfg.CloroAPIKey != "" || a.cfg.ResendAPIKey != "") {
		queues := map[string]river.QueueConfig{}
		if workers == nil {
			workers = river.NewWorkers()
			if a.cfg.OpenRouterAPIKey != "" {
				if a.provider == nil {
					a.provider = openrouter.New(a.cfg.OpenRouterAPIKey, a.cfg.ProviderTimeout)
				}
				generationStore := database.NewStore(db).WithCloudMode(a.cfg.DeploymentMode == "cloud").
					WithLiveCollection(a.cfg.CloroAPIKey != "").WithFirstLookDailyLimit(a.cfg.FirstLookDailyLimit)
				workerStores = append(workerStores, generationStore)
				river.AddWorker(workers, &generationWorker{
					store:    generationStore,
					provider: a.provider,
					timeout:  2*a.cfg.ProviderTimeout + 30*time.Second,
				})
			}
			if a.cfg.CloroAPIKey != "" {
				if a.cloroProvider == nil {
					a.cloroProvider = cloro.New(a.cfg.CloroAPIKey)
				}
				visibilityStore := database.NewStore(db).WithCloudMode(a.cfg.DeploymentMode == "cloud").
					WithWelcomeEmail(a.cfg.ResendAPIKey != "")
				workerStores = append(workerStores, visibilityStore)
				worker := &visibilityWorker{
					store:     visibilityStore,
					provider:  a.cloroProvider,
					syncSlots: make(chan struct{}, a.cfg.WorkerConcurrency),
				}
				if a.cfg.OpenRouterAPIKey != "" {
					worker.judge = openrouter.New(a.cfg.OpenRouterAPIKey, a.cfg.ProviderTimeout)
				}
				river.AddWorker(workers, worker)
			}
			if a.cfg.ResendAPIKey != "" {
				river.AddWorker(workers, &welcomeWorker{
					store:  database.NewStore(db),
					sender: resendprovider.New(a.cfg.ResendAPIKey, a.cfg.EmailFrom, a.cfg.PublicURL),
				})
				river.AddWorker(workers, &firstCheckEmailWorker{
					store:  database.NewStore(db),
					sender: resendprovider.New(a.cfg.ResendAPIKey, a.cfg.EmailFrom, a.cfg.PublicURL),
				})
				river.AddWorker(workers, &passwordResetWorker{
					issuer: domain.NewAuthCommander(database.NewStore(db), domain.AuthOptions{OpenSignup: a.cfg.AllowSignup}),
					sender: resendprovider.New(a.cfg.ResendAPIKey, a.cfg.EmailFrom, a.cfg.PublicURL),
				})
				river.AddWorker(workers, &passwordChangedWorker{
					store:  database.NewStore(db),
					sender: resendprovider.New(a.cfg.ResendAPIKey, a.cfg.EmailFrom, a.cfg.PublicURL),
				})
			}
		}
		maintenance := &maintenanceWorker{}
		river.AddWorker(workers, maintenance)
		queues["maintenance"] = river.QueueConfig{MaxWorkers: 1}
		if a.cfg.OpenRouterAPIKey != "" {
			queues["generation"] = river.QueueConfig{MaxWorkers: a.cfg.WorkerConcurrency}
		}
		if a.cfg.CloroAPIKey != "" {
			queues["visibility"] = river.QueueConfig{MaxWorkers: a.cfg.WorkerConcurrency}
		}
		if a.cfg.ResendAPIKey != "" {
			queues["email"] = river.QueueConfig{MaxWorkers: 1}
		}
		config := &river.Config{
			Queues:          queues,
			Workers:         workers,
			SoftStopTimeout: 60 * time.Second,
			PeriodicJobs: []*river.PeriodicJob{
				river.NewPeriodicJob(river.PeriodicInterval(time.Minute), func() (river.JobArgs, *river.InsertOpts) {
					return maintenanceArgs{}, &river.InsertOpts{Queue: "maintenance", MaxAttempts: 3}
				}, &river.PeriodicJobOpts{RunOnStart: true}),
			},
		}
		riverClient, err = river.NewClient(riverdatabasesql.New(sqlDB), config)
		if err != nil {
			return fmt.Errorf("create river client: %w", err)
		}
		for _, workerStore := range workerStores {
			workerStore.WithEnqueueClient(riverClient)
		}
		maintenance.store = database.NewStore(db).WithEnqueueClient(riverClient).
			WithLiveCollection(a.cfg.CloroAPIKey != "").WithCloudMode(a.cfg.DeploymentMode == "cloud").
			WithFirstLookDailyLimit(a.cfg.FirstLookDailyLimit)
		if a.cfg.DeploymentMode == "cloud" {
			maintenance.billing = domain.NewBillingCommander(maintenance.store, stripe.New(a.cfg.StripeSecretKey), a.cfg.StripePriceID, a.cfg.PublicURL)
		}
		if err := riverClient.Start(context.Background()); err != nil {
			return fmt.Errorf("start river client: %w", err)
		}
	}
	liveCollection := a.cfg.CloroAPIKey != "" && riverClient != nil
	store := database.NewStore(db).WithEnqueueClient(riverClient).
		WithWelcomeEmail(a.cfg.ResendAPIKey != "").
		WithPasswordResetEmail(a.cfg.ResendAPIKey != "" && a.cfg.PublicURL != "").
		WithLiveCollection(liveCollection).WithCloudMode(a.cfg.DeploymentMode == "cloud").
		WithFirstLookDailyLimit(a.cfg.FirstLookDailyLimit)
	options := api.ServerOptions{
		Searcher:       searcher,
		AIDrafting:     a.cfg.OpenRouterAPIKey != "" && riverClient != nil,
		LiveCollection: liveCollection,
		BrandLooker:    sitelookup.New(),
		OpenSignup:     a.cfg.AllowSignup,
		PublicURL:      a.cfg.PublicURL,
		ClientIPHeader: a.cfg.ClientIPHeader,
	}
	if a.cfg.DeploymentMode == "cloud" {
		stripeClient := stripe.New(a.cfg.StripeSecretKey)
		options.Billing = api.NewBillingHandler(
			domain.NewBillingQuerier(store, stripeClient, a.cfg.StripePriceID),
			domain.NewBillingCommander(store, stripeClient, a.cfg.StripePriceID, a.cfg.PublicURL),
			a.cfg.StripeWebhookSecret,
		)
	}

	handler := api.NewServer(store, sess, options)
	if a.cfg.DeploymentMode != "cloud" && a.cfg.PublicURL == "" {
		slog.Warn("KOI_PUBLIC_URL is not set: the session cookie is not marked Secure, so serve this instance on localhost only")
	}

	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      2 * time.Minute,
		IdleTimeout:       2 * time.Minute,
	}
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
		riverCtx, cancelRiver := context.WithTimeout(context.Background(), 65*time.Second)
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
