package koiapi

import (
	"context"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/oauthex"

	"github.com/koihq/koi/pkg/koiapi/middlewares"
	apiresponse "github.com/koihq/koi/pkg/koiapi/response"
	"github.com/koihq/koi/pkg/koiauth"
	"github.com/koihq/koi/pkg/koidomain"
	"github.com/koihq/koi/pkg/web"
)

func NewServer(store koidomain.Store, sess *scs.SessionManager, searcher koidomain.CompanySearcher, aiDrafting bool, collection ...bool) http.Handler {
	return buildHTTPServer(store, sess, searcher, aiDrafting, collection...)
}

func buildHTTPServer(store koidomain.Store, sess *scs.SessionManager, searcher koidomain.CompanySearcher, aiDrafting bool, collection ...bool) http.Handler {
	liveCollection := len(collection) > 0 && collection[0]
	return buildHTTPServerWithBilling(store, sess, searcher, aiDrafting, liveCollection, nil)
}

func NewCloudServer(store koidomain.Store, sess *scs.SessionManager, searcher koidomain.CompanySearcher, aiDrafting, liveCollection bool, billing *BillingHandler) http.Handler {
	return buildHTTPServerWithBilling(store, sess, searcher, aiDrafting, liveCollection, billing)
}

func buildHTTPServerWithBilling(store koidomain.Store, sess *scs.SessionManager, searcher koidomain.CompanySearcher, aiDrafting, liveCollection bool, billing *BillingHandler) http.Handler {
	logoKey := strings.TrimSpace(os.Getenv("VITE_LOGODEV_PUBLISHABLE_KEY"))
	if !strings.HasPrefix(logoKey, "pk_") {
		logoKey = ""
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middlewares.RequestIDHeader)
	r.Use(middlewares.RequestLogger)
	r.Use(middlewares.Recoverer)
	r.Use(render.SetContentType(render.ContentTypeJSON))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "Idempotency-Key"},
		AllowCredentials: true,
	}))
	r.Use(sess.LoadAndSave)

	r.Get("/health", NewHealthHandler(store).Get)
	var credentials koidomain.CredentialService
	if store.CredentialRepo() != nil {
		credentials = koidomain.NewCredentialService(store)
		r.Handle("/mcp", bearerIdentity(store, credentials, newMCPHandler(store, searcher, aiDrafting, billing)))
		oauth := &oauthHandler{store: credentials, sess: sess, origin: oauthOrigin()}
		metadata := auth.ProtectedResourceMetadataHandler(&oauthex.ProtectedResourceMetadata{Resource: oauth.resourceURL(), AuthorizationServers: []string{oauth.origin}, ScopesSupported: []string{"koi:read", "koi:write"}, BearerMethodsSupported: []string{"header"}})
		r.Handle("/.well-known/oauth-protected-resource", metadata)
		r.Handle("/.well-known/oauth-protected-resource/mcp", metadata)
		r.Get("/.well-known/oauth-authorization-server", oauth.serverMetadata)
		r.Get("/.well-known/oauth-authorization-server/oauth", oauth.serverMetadata)
		r.Post("/oauth/register", oauth.register)
		r.Get("/oauth/authorize", oauth.authorize)
		r.Get("/api/v1/oauth/consent", oauth.consent)
		r.Post("/api/v1/oauth/consent", oauth.authorizePost)
		r.Post("/oauth/token", oauth.token)
		r.Post("/oauth/revoke", oauth.revoke)
	}
	authenticate := middlewares.Authenticate(sess, func(ctx context.Context, userID uuid.UUID) (koiauth.Identity, error) {
		return koidomain.ResolveIdentity(ctx, store, userID)
	})
	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", NewAuthHandler(koidomain.NewAuthQuerier(store), koidomain.NewAuthCommander(store), sess, authenticate).Routes())
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
				apiresponse.JSON(w, req, http.StatusOK, map[string]any{
					"competitorSearch":      searcher != nil,
					"aiDrafting":            aiDrafting,
					"liveCollection":        liveCollection,
					"cloudBilling":          billing != nil,
					"mcpUrl":                oauthOrigin() + "/mcp",
					"logoDevPublishableKey": logoKey,
				})
			})
			var billingGuard func(http.Handler) http.Handler
			if billing != nil {
				billingGuard = middlewares.RequireBilling(billing.querier)
			}
			r.Route("/brands", NewBrandHandler(koidomain.NewBrandQuerier(store), koidomain.NewBrandCommander(store), sess, billingGuard).Routes())
			r.Group(func(r chi.Router) {
				if billingGuard != nil {
					r.Use(billingGuard)
				}
				r.Route("/prompts", NewPromptHandler(koidomain.NewPromptQuerier(store), koidomain.NewPromptCommander(store), koidomain.NewVisibilityAnalyticsQuerier(store)).Routes())
				r.Route("/competitors", NewCompetitorHandler(koidomain.NewCompetitorQuerier(store, searcher), koidomain.NewCompetitorCommander(store)).Routes())
				visibility := NewVisibilityHandler(koidomain.NewVisibilityQuerier(store), koidomain.NewVisibilityAnalyticsQuerier(store))
				r.Get("/tracking", visibility.Tracking)
				r.Get("/prompt-history", visibility.ListHistoricalPrompts)
				r.Get("/prompt-history/{promptID}/runs", visibility.ListPromptRuns)
				r.Get("/prompt-history/{promptID}/visibility", visibility.PromptVisibility)
				r.Get("/visibility/summary", visibility.Summary)
				r.Route("/runs", visibility.Routes())
				r.Route("/generations", NewGenerationHandler(koidomain.NewGenerationQuerier(store), koidomain.NewGenerationCommander(store, aiDrafting)).Routes())
			})
		})
	})
	r.Handle("/*", spaHandler(web.FileSystem()))

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
