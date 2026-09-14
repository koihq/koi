# Backend architecture

Single Go binary built from `cmd/koi/main.go`. The binary boots the app (`koiapp.NewKoiApp`) and serves the HTTP API on port 8080. The UI bundle is produced by `make ui-build` into `pkg/web/dist` (gitignored) and embedded at compile time.

`make dev` runs Air against `./cmd/koi` (see `.air.toml`) plus Vite. Do not serve dev UI from `:8080`; that is the embedded bundle.

## Layering

```
HTTP                pkg/koiapi/*_handler.go      (chi routes, DTO decode, render)
  ↓
Domain              pkg/koidomain/*.go           (entities, *Commander, *Querier, Store interface)
  ↓
Persistence         pkg/koidatabase/gorm_repo_*.go  (GORM implementations of the Store)
```

Handlers receive dependencies via constructor (querier, commander). No global state.

Authz is a later slot: keep constructors injectable so an authorizer can be added without rewriting routes. Do not add auth middleware now.

## Stack (from `go.mod`)

- HTTP router: `chi/v5` + `chi/cors`, `chi/render` (JSON).
- ORM: GORM (Postgres driver).
- Config: env vars with `KOI_*` prefix, `godotenv` for `.env`.
- Logging: `slog`.
- Testing: `stretchr/testify`.
- Misc: `google/uuid`.

## Handler pattern (`pkg/koiapi/`)

Handler struct wires querier and commander:

```go
type BrandHandler struct {
    querier   koidomain.BrandQuerier
    commander koidomain.BrandCommander
}

func NewBrandHandler(q koidomain.BrandQuerier, c koidomain.BrandCommander) *BrandHandler {
    return &BrandHandler{querier: q, commander: c}
}

func (h *BrandHandler) Routes() func(r chi.Router) {
    return func(r chi.Router) {
        r.Get("/", api.List(h.querier, BrandToRes))
        r.With(middlewares.DecodeBody[CreateBrandReq]()).Post("/", api.Create(h.Create, BrandToRes))
        r.Group(func(r chi.Router) {
            r.Use(middlewares.ID)
            r.Get("/{id}", api.Get(h.querier.Get, BrandToRes))
            // update, delete, ...
        })
    }
}
```

Health is an exception: no domain entity, handler pings `Store` directly.

## Domain (`pkg/koidomain/`)

Each entity has an `entity.go` file declaring:

- The GORM-mapped struct.
- `TableName()`, `Validate()`, `Update(params)` methods.
- `EventType*` constants for domain events (when events exist).
- `Create*Params` / `Update*Params` value types.
- A `New<Entity>Commander(store)` constructor.

The `Store` interface (`pkg/koidomain/store.go`) aggregates repos and exposes `Atomic(ctx, fn)` for transactions:

```go
type Store interface {
    Atomic(context.Context, func(Store) error) error
    Ping(context.Context) error
    // BrandRepo() BrandRepository  — add getters as entities land
}
```

Slice 0 has no domain entities. Add the first repo getter when Brand (or equivalent) is introduced.

## Repositories (`pkg/koidatabase/`)

GORM-backed. Filter and sort mappers are declared at package scope. Name files `gorm_repo_<entity>.go`.

```go
var applyBrandFilter = map[string]func(*gorm.DB, []string) *gorm.DB{
    "name": func(db *gorm.DB, values []string) *gorm.DB {
        return db.Where("name ILIKE ?", "%"+values[0]+"%")
    },
}

func NewGormBrandRepository(db *gorm.DB) *GormBrandRepository {
    return &GormBrandRepository{db: db}
}
```

`pkg/koidatabase` implements `koidomain.Store`. `Atomic` opens a GORM transaction and passes a store bound to that tx.

## Server wiring (`pkg/koiapi/server.go`)

`buildHTTPServer` instantiates commanders, then handlers, then mounts routes on a `chi.Router`. New handlers are registered here.

```go
r.Route("/api/brands", NewBrandHandler(brandQuerier, brandCommander).Routes())
```

Public routes stay under `/api`. `GET /health` is unprefixed.
