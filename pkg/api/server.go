package api

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/oauthex"

	"github.com/koihq/koi/pkg/api/middlewares"
	"github.com/koihq/koi/pkg/api/response"
	"github.com/koihq/koi/pkg/auth"
	"github.com/koihq/koi/pkg/domain"
	"github.com/koihq/koi/pkg/web"
)

// ServerOptions lists the optional capabilities of an instance. Billing is set only on Koi Cloud.
type ServerOptions struct {
	Searcher       domain.CompanySearcher
	AIDrafting     bool
	LiveCollection bool
	Billing        *BillingHandler
	BrandLooker    domain.BrandLooker
	// OpenSignup false admits only the first user.
	OpenSignup bool
	// PublicURL enables HSTS when HTTPS and turns off the local dev origin when not localhost.
	PublicURL string
	// ClientIPHeader names the header a trusted proxy overwrites with the client's address.
	ClientIPHeader string
}

type OnboardingEventReq struct {
	Step string `json:"step"`
}

// services is the single set of domain services shared by the HTTP routes and MCP tools.
type services struct {
	auth               domain.AuthQuerier
	brands             domain.BrandQuerier
	brandCommands      domain.BrandCommander
	prompts            domain.PromptQuerier
	promptCommands     domain.PromptCommander
	competitors        domain.CompetitorQuerier
	competitorCommands domain.CompetitorCommander
	generations        domain.GenerationQuerier
	generationCommands domain.GenerationCommander
	visibility         domain.VisibilityQuerier
	tracking           domain.TrackingCommander
	analytics          *domain.VisibilityAnalyticsQuerier
}

func newServices(store domain.Store, options ServerOptions) services {
	return services{
		auth:               domain.NewAuthQuerier(store),
		brands:             domain.NewBrandQuerier(store),
		brandCommands:      domain.NewBrandCommander(store),
		prompts:            domain.NewPromptQuerier(store),
		promptCommands:     domain.NewPromptCommander(store),
		competitors:        domain.NewCompetitorQuerier(store, options.Searcher),
		competitorCommands: domain.NewCompetitorCommander(store),
		generations:        domain.NewGenerationQuerier(store),
		generationCommands: domain.NewGenerationCommander(store, options.AIDrafting),
		visibility:         domain.NewVisibilityQuerier(store),
		tracking:           domain.NewTrackingCommander(store),
		analytics:          domain.NewVisibilityAnalyticsQuerier(store),
	}
}

const (
	maxRequestBytes = 1 << 20
	devUIOrigin     = "http://localhost:5173"
)

// localPublicURL reports an instance that may be browsed through the Vite dev server.
func localPublicURL(publicURL string) bool {
	if publicURL == "" {
		return true
	}

	parsed, err := url.Parse(publicURL)
	if err != nil {
		return false
	}

	host := parsed.Hostname()

	return host == "localhost" || host == "127.0.0.1"
}

func passThrough(next http.Handler) http.Handler {
	return next
}

func NewServer(store domain.Store, sess *scs.SessionManager, options ServerOptions) http.Handler {
	billing := options.Billing
	shared := newServices(store, options)

	logoKey := strings.TrimSpace(os.Getenv("VITE_LOGODEV_PUBLISHABLE_KEY"))
	if !strings.HasPrefix(logoKey, "pk_") {
		logoKey = ""
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middlewares.RequestIDHeader)
	r.Use(middlewares.RequestLogger)
	r.Use(middlewares.Recoverer)
	r.Use(middleware.RequestSize(maxRequestBytes))
	r.Use(middlewares.SecurityHeaders(options.PublicURL))
	r.Use(render.SetContentType(render.ContentTypeJSON))

	var trustedOrigins []string
	if localPublicURL(options.PublicURL) {
		trustedOrigins = []string{devUIOrigin}

		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   trustedOrigins,
			AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "Idempotency-Key"},
			AllowCredentials: true,
		}))
	}

	r.Use(middlewares.CrossOriginProtection(trustedOrigins, []string{
		"/mcp",
		"/oauth/token",
		"/oauth/register",
		"/oauth/revoke",
		"/api/v1/webhooks/stripe",
	}))
	r.Use(sess.LoadAndSave)

	limiter := middlewares.NewRateLimiter(options.ClientIPHeader)

	r.Get("/health", NewHealthHandler(store).Get)
	var credentials domain.CredentialService
	if store.CredentialRepo() != nil {
		credentials = domain.NewCredentialService(store)
		r.Handle("/mcp", requireConfiguredRemoteMCP(bearerIdentity(store, credentials, newMCPHandler(shared, billing))))
		oauth := &oauthHandler{store: credentials, sess: sess, origin: oauthOrigin()}
		metadata := mcpauth.ProtectedResourceMetadataHandler(&oauthex.ProtectedResourceMetadata{Resource: oauth.resourceURL(), AuthorizationServers: []string{oauth.origin}, ScopesSupported: []string{"koi:read", "koi:write"}, BearerMethodsSupported: []string{"header"}})
		r.Handle("/.well-known/oauth-protected-resource", requireConfiguredRemoteMCP(metadata))
		r.Handle("/.well-known/oauth-protected-resource/mcp", requireConfiguredRemoteMCP(metadata))
		r.With(requireConfiguredRemoteMCP).Get("/.well-known/oauth-authorization-server", oauth.serverMetadata)
		r.With(requireConfiguredRemoteMCP).Get("/.well-known/oauth-authorization-server/oauth", oauth.serverMetadata)
		r.With(limiter.PerIP(20, time.Hour)).Post("/oauth/register", oauth.register)
		r.With(limiter.PerIP(60, time.Minute)).Get("/oauth/authorize", oauth.authorize)
		r.Get("/api/v1/oauth/consent", oauth.consent)
		r.Post("/api/v1/oauth/consent", oauth.authorizePost)
		r.With(limiter.PerIP(60, time.Minute)).Post("/oauth/token", oauth.token)
		r.Post("/oauth/revoke", oauth.revoke)
	}
	authenticate := middlewares.Authenticate(sess, func(ctx context.Context, userID uuid.UUID) (auth.Identity, error) {
		return domain.ResolveIdentity(ctx, store, userID)
	})
	r.Route("/api/v1", func(r chi.Router) {
		authCommander := domain.NewAuthCommander(store, domain.AuthOptions{OpenSignup: options.OpenSignup})
		authHandler := NewAuthHandler(domain.NewAuthQuerier(store), authCommander, sess, authenticate)
		authHandler.limiter = limiter
		r.Route("/auth", authHandler.Routes())
		if billing != nil {
			r.Post("/webhooks/stripe", billing.Webhook)
		}
		r.Group(func(r chi.Router) {
			r.Use(authenticate)
			if credentials != nil {
				r.Mount("/credentials", credentialRoutes(credentials))
			}
			if billing != nil {
				r.Route("/billing", billing.Routes())
			}
			r.Get("/capabilities", func(w http.ResponseWriter, req *http.Request) {
				response.JSON(w, req, http.StatusOK, map[string]any{
					"competitorSearch":      options.Searcher != nil,
					"aiDrafting":            options.AIDrafting,
					"liveCollection":        options.LiveCollection,
					"cloudBilling":          billing != nil,
					"mcpUrl":                publicMCPURL(req),
					"logoDevPublishableKey": logoKey,
				})
			})
			billingGuard := passThrough
			if billing != nil {
				billingGuard = middlewares.RequireBilling(billing.querier)
			}

			brands := NewBrandHandler(shared.brands, shared.brandCommands, sess, billingGuard)
			brands.looker = options.BrandLooker
			brands.lookupLimit = limiter.PerIP(30, time.Minute)
			r.Route("/brands", brands.Routes())
			if billing != nil {
				r.With(middlewares.DecodeBody[OnboardingEventReq]()).Post("/onboarding/events", func(w http.ResponseWriter, req *http.Request) {
					input := middlewares.Body[OnboardingEventReq](req)
					if err := domain.RecordOnboardingStep(req.Context(), store, input.Step); err != nil {
						response.WriteDomainError(w, req, err)

						return
					}

					w.WriteHeader(http.StatusNoContent)
				})
			}
			r.Group(func(r chi.Router) {
				r.Use(billingGuard)

				visibility := NewVisibilityHandler(shared.visibility, shared.tracking, shared.analytics)

				r.Route("/prompts", NewPromptHandler(shared.prompts, shared.promptCommands).Routes())
				r.Route("/competitors", NewCompetitorHandler(shared.competitors, shared.competitorCommands).Routes())
				r.Get("/tracking", visibility.Tracking())
				r.With(middlewares.DecodeBody[BrandRequestScope]()).Post("/tracking/start", visibility.StartTracking())
				r.Route("/prompt-history", visibility.HistoryRoutes())
				r.Get("/visibility/summary", visibility.Summary())
				r.Route("/runs", visibility.Routes())
				r.Route("/generations", NewGenerationHandler(shared.generations, shared.generationCommands).Routes())
			})
		})
	})
	r.With(middlewares.ContentSecurityPolicy).Handle("/*", spaHandler(web.FileSystem()))

	return r
}

func spaHandler(files http.FileSystem) http.Handler {
	fileServer := http.FileServer(files)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead ||
			r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/api/") ||
			r.URL.Path == "/health" || strings.HasPrefix(r.URL.Path, "/health/") {
			http.NotFound(w, r)
			return
		}

		file, err := files.Open(path.Clean("/" + r.URL.Path))
		if err == nil {
			_ = file.Close()
		} else if path.Ext(r.URL.Path) == "" && !strings.HasPrefix(r.URL.Path, "/assets/") {
			r = r.Clone(r.Context())
			r.URL.Path = "/"
		}

		fileServer.ServeHTTP(w, r)
	})
}
