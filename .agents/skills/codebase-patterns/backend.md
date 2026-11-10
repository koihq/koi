# Backend architecture

Single Go binary built from `cmd/koi/main.go`. The binary boots the app (`app.New`) and serves the HTTP API on port 8080. The UI bundle is produced by `make ui-build` into `pkg/web/dist` (gitignored) and embedded at compile time.

`make dev` runs Air against `./cmd/koi` (see `.air.toml`) plus Vite. Do not serve dev UI from `:8080`; that is the embedded bundle.

## Layering

```
HTTP                pkg/api/*_handler.go      (chi routes, DTO decode, JSON)
  ↓
Domain              pkg/domain/*.go           (entities, Commander/Querier interfaces, Store)
  ↓
Persistence         pkg/database/gorm_repo_*.go  (GORM implementations of Store)
```

Handlers receive querier and commander via constructor. No global state.

Cookie session auth exists (`scs`, `koi_session`, `middlewares.Authenticate`). Keep constructors injectable. Do not add Keycloak or JWT.

Shared HTTP responses live in `api/response`. Small `api.Get`, `List`, `Create`, `Update`, `Delete`, `Action`, `Command` and `Read` adapters share only transport handling. Cookie, OAuth, billing and generation creation handlers stay explicit. See `docs/architecture.md` for current contracts.

## Stack (from `go.mod`)

- HTTP router: `chi/v5` + `chi/cors`, `chi/render` (JSON).
- Sessions: `scs/v2` + `scs/postgresstore`.
- ORM: GORM (Postgres driver).
- Config: env vars with `KOI_*` prefix, `godotenv` for `.env`.
- Logging: stdlib `slog` (`KOI_LOG_LEVEL`, `KOI_LOG_FORMAT=text|json`).
- Testing: `stretchr/testify` + `vektra/mockery/v3` (`make mocks`).
- Misc: `google/uuid`, argon2id passwords.

## Handler pattern (`pkg/api/`)

Before adding a route or MCP tool, find the nearest existing operation and trace its callers. When an operation exists on both HTTP and MCP, reuse value validation and domain error codes, then test the same defaults, invalid values, and results on both surfaces. For example, `analysis_summary` and `prompt_visibility` must accept the same visibility `days` and `engines` as the HTTP routes. For OAuth changes, test the complete browser flow with one hostname, distinct public hostnames, and localhost on different ports.

Handlers receive domain interfaces. `server.go` applies authentication once to the
protected route group and injects the identity resolver. Resource handlers do not
receive the session manager; auth and brand preference handlers still manage cookies
or session state. Billing guards consume the verified organization identity.

A CRUD handler is its routes plus small functions that turn a request into params.
Keep the request, response and mapper in the handler file, and name them
`CreatePromptReq`, `UpdatePromptReq`, `PromptRes` and `PromptToRes`. Resource routes
use `{id}`. List parsers return only the resource filter, because `api.List` reads the
page parameters itself:

```go
r.Get("/", List(competitorFilter, h.querier.List, CompetitorToRes))
r.Post("/{id}/confirm", Action(h.commander.Confirm, CompetitorToRes))
r.With(middlewares.DecodeBody[BrandRequestScope]()).Post("/tracking/start", Command(h.startTracking))
```

MCP tools use the same services through `mcpResult` and `mcpPageResult`:

```go
return mcpResult(PromptToRes)(shared.prompts.Get(ctx, in.ID.UUID()))
```

HTTP DTOs explicitly map public camelCase fields without changing persistence
serialization. Collections require `brandId`; creates and collection commands carry
it in the body. Details and mutations take their UUID alone. Foreign resource IDs
return 404 through repository scope. `active_brand_id` is only a UI preference.

Health is an exception: no domain entity, handler pings Store directly.
Public JSON lives under `/api/v1`; `GET /health` is unprefixed.

## Errors

Domain returns `*domain.Error` with a stable `Code` and optional `Fields`. UI owns user-facing copy; the API contract is the code (and field codes), not the message.

Sentinels (`ErrValidation`, `ErrEmailTaken`, …) stay for `errors.Is`. HTTP mapping lives only in `pkg/api/response`:

```json
{
  "error": {
    "code": "validation",
    "message": "validation",
    "fields": [{ "field": "email", "code": "invalid" }]
  }
}
```

| code | HTTP |
| --- | --- |
| `validation`, `invalid_json` | 400 |
| `email_taken`, `first_look_unavailable` | 409 |
| `invalid_credentials`, `unauthorized` | 401 |
| `signup_closed`, `cross_origin_request` | 403 |
| `not_found` | 404 |
| `rate_limited` | 429, with `Retry-After` |
| anything else | 500 `{ code: "internal", message: "internal error" }` (log the real error) |

`invalid_credentials` has no fields (do not leak which of email/password). Unique-violation on `users.email` maps to `email_taken` in `database`.

## Logging

`app` sets `slog` from env. Chi `RequestID` stays. `middlewares.RequestLogger` logs `method`, `path`, `status`, `duration_ms`, `request_id` (no query, no body). `middlewares.Recoverer` logs panics. 5xx from `WriteDomainError` are logged; 4xx are not. GORM stays at `Warn` with `IgnoreRecordNotFoundError` and `ParameterizedQueries` (not-found is control flow; never interpolate bound values such as emails).

## Domain (`pkg/domain/`)

Entities embed `domain.BaseEntity`. Postgres assigns `id` (`gen_random_uuid()`). GORM fills `CreatedAt` / `UpdatedAt` (`autoCreateTime` / `autoUpdateTime`). Timestamps are `json:"-"`; commanders do not set ID or timestamps. Goose migrations create the tables for these same structs (no parallel row type); a struct change needs a new migration.

```go
type BaseEntity struct {
    ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    CreatedAt time.Time `json:"-" gorm:"not null;autoCreateTime"`
    UpdatedAt time.Time `json:"-" gorm:"not null;autoUpdateTime"`
}
```

Each entity file declares the GORM-mapped struct, `TableName()`, `Validate()`, and a repository interface. Commanders/queriers are interfaces with unexported implementations (`NewPromptCommander(store) PromptCommander`). Input types are `CreatePromptParams`, `UpdatePromptParams` and `PromptFilter`.

Business rules live here, not in `database`. Put what an entity can decide alone on the entity (`NewPrompt`, `Prompt.Update`, `Prompt.Confirm`, `Competitor.ConflictsWith`), and put the sequence in the commander. A change to what a brand tracks runs under its lock:

```go
func (c *promptCommander) Delete(ctx context.Context, id uuid.UUID) error {
    return withLockedChild(ctx, c.store, id, promptChildren, func(tx Store, _ *Brand, prompt *Prompt) error {
        if err := markPromptResults(ctx, tx, prompt.BrandID, prompt); err != nil {
            return err
        }

        if prompt.HasResults {
            return ErrResourceConflict
        }

        return tx.PromptRepo().Delete(ctx, id)
    })
}
```

Use `withBrandLock` when the child does not exist yet, as in `Create`. Check the brand before validating input, so another organization's brand is 404 rather than 400. `brandChildQuerier[T, F]` serves `Get` and `List` for anything owned by a brand; `List` checks the brand first.

Compose repository interfaces from `BaseEntityRepository[T]`, `MutableRepository[T]` and `Lister[T, F]`, then declare only what differs. Filters for a brand's rows embed `BrandFilter`. Entities owned by a brand implement `BrandChild`.

`Store` (`pkg/domain/store.go`) aggregates repos and exposes `Atomic(ctx, fn)` for transactions:

```go
type Store interface {
    Atomic(context.Context, func(Store) error) error
    Ping(context.Context) error
    UserRepo() UserRepository
    OrganizationRepo() OrganizationRepository
    MembershipRepo() MembershipRepository
}
```

## Repositories (`pkg/database/`)

GORM-backed. Name files `gorm_repo_<entity>.go`, types `GormPromptRepository` and constructors `NewPromptRepository`. `mapDBError` turns `ErrRecordNotFound` into `ErrNotFound` and email unique-violations into `ErrEmailTaken`.

`Repository[T]` provides `Create`, `Get`, `Update` and `Delete`; `ListRepository[T, F]` adds `List`. A concrete repository is configuration plus its own queries:

```go
func NewPromptRepository(db *gorm.DB) *GormPromptRepository {
    base := newBrandChildRepository[domain.Prompt](db, "text", "confirmed", "inactive")

    return &GormPromptRepository{
        ListRepository: newListRepository(base, applyPromptFilter, promptOrder),
    }
}
```

The columns passed to the constructor are the only ones `Update` writes. Use `newRepository` for entities without a tenant (users, organizations, memberships), `newBrandChildRepository` for anything owned by a brand, and `newScopedRepository` with a `tenantScope` for other scopes. A scoped repository without `authorizeCreate` rejects every `Create`. Use `getBy` only for queries with their own meaning, such as `LockForUpdate` or `Latest`. Do not add product rules or transactions to these repositories.

Worker state machines (`visibility_attempts`, `generation_attempts`, `visibility_schedule`), analytics, prompt history, billing and credentials keep specialized queries. Anything that updates existing prompts or competitors must hold the brand lock, because `Update` writes every declared column.

`pkg/database` implements `domain.Store`. `Atomic` opens a GORM transaction and passes a copy of the store bound to that tx. A repository transaction inside it becomes a savepoint.

## Tests

Generate doubles with `make mocks` (or `go generate ./pkg/domain`) into `*_test.go` in the consuming package (`pkg/domain/mocks_test.go`, `pkg/api/mocks_test.go`). Same-package mocks avoid an import cycle with `domain`. Domain tests mock `Store` + repos (helper for `Atomic` to invoke the callback) and check the order of lock, write and review invalidation. Rules that depend on the database, such as rollback and tenant scope, have integration tests in `pkg/database/commander_integration_test.go`. Handler tests mock the commander and querier interfaces. Health mocks `Store.Ping` only. Commit generated mocks. No live Postgres in unit tests.

## Server wiring (`pkg/api/server.go`)

Compose domain services, resource handlers and middleware here. `newServices` builds
the single set shared with MCP, and `ServerOptions` lists optional capabilities. Authentication
resolves membership into `auth.Identity{UserID, OrganizationID, Role}` once per
request. Do not put resource authorization in generic HTTP adapters; scoped
repositories enforce tenant access on every read and mutation.

## Lists and integration tests

Persisted-resource `GET` lists accept `page` and `pageSize` and return `Page[T]`, including empty results with `items: []`, `totalItems: 0`, `totalPages: 0`, `currentPage: 1`, `hasNext: false`, and `hasPrev: false`. Keep OpenAPI and UI wrappers aligned. Country options and competitor search are bounded combobox lookups.

Postgres integration tests call `database.Open(testdb.URL(t))`, then run only the migrations that test needs. `internal/testdb` reads `KOI_DATABASE_URL` from the environment or repo `.env`, creates and cleans up a schema in that database, and fails if Postgres is unavailable. It does not migrate, because some tests verify behavior before River tables exist.

Break up nested Go literals even when `gofmt` accepts one line. For external APIs, inspect installed clients, read official vendor docs, and verify code against the pinned SDK version. Use the installed OpenRouter SDK for Responses API and Resty for direct JSON HTTP integrations without a suitable SDK. Nontrivial plans include concrete type or call examples, data flow, compatibility choices, failure cases, and acceptance tests.
