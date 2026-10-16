package koiapi

import (
	"net/http"
	"path"
	"strings"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"

	"github.com/koihq/koi/pkg/koiapi/api"
	"github.com/koihq/koi/pkg/koiapi/middlewares"
	"github.com/koihq/koi/pkg/koidomain"
	"github.com/koihq/koi/pkg/web"
)

func NewServer(store koidomain.Store, sess *scs.SessionManager, searcher koidomain.CompanySearcher, aiDrafting bool, collection ...bool) http.Handler {
	return buildHTTPServer(store, sess, searcher, aiDrafting, collection...)
}

func buildHTTPServer(store koidomain.Store, sess *scs.SessionManager, searcher koidomain.CompanySearcher, aiDrafting bool, collection ...bool) http.Handler {
	liveCollection := len(collection) > 0 && collection[0]
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
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
	r.Route("/api/v1", func(r chi.Router) {
		r.With(Authorizer(sess)).Get("/capabilities", func(w http.ResponseWriter, req *http.Request) {
			api.JSON(w, req, http.StatusOK, map[string]bool{"competitorSearch": searcher != nil, "aiDrafting": aiDrafting, "liveCollection": liveCollection})
		})
		r.Route("/auth", NewAuthHandler(
			koidomain.NewAuthQuerier(store),
			koidomain.NewAuthCommander(store),
			sess,
		).Routes())
		r.Route("/brands", func(r chi.Router) {
			NewBrandHandler(
				koidomain.NewBrandQuerier(store),
				koidomain.NewBrandCommander(store),
				sess,
			).Routes()(r)
			r.Route("/{brandID}/prompts", NewPromptHandler(
				koidomain.NewPromptQuerier(store),
				koidomain.NewPromptCommander(store),
				sess,
			).Routes())
			r.Route("/{brandID}/competitors", NewCompetitorHandler(
				koidomain.NewCompetitorQuerier(store, searcher),
				koidomain.NewCompetitorCommander(store),
				sess,
			).Routes())
			visibility := NewVisibilityHandler(koidomain.NewVisibilityQuerier(store), sess)
			r.With(Authorizer(sess)).Get("/{brandID}/tracking", visibility.Tracking)
			r.With(Authorizer(sess)).Get("/{brandID}/prompt-history", visibility.ListHistoricalPrompts)
			r.With(Authorizer(sess)).Get("/{brandID}/prompt-history/{promptID}/runs", visibility.ListPromptRuns)
			r.Route("/{brandID}/runs", visibility.Routes())
			r.Route("/{brandID}/generations", NewGenerationHandler(
				koidomain.NewGenerationQuerier(store),
				koidomain.NewGenerationCommander(store, aiDrafting),
				sess,
			).Routes())
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
