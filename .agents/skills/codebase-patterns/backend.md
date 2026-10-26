# Backend architecture

Single Go binary built from `cmd/koi/main.go`. The binary boots the app (`koiapp.NewKoiApp`) and serves the HTTP API on port 8080. The UI bundle is produced by `make ui-build` into `pkg/web/dist` (gitignored) and embedded at compile time.

`make dev` runs Air against `./cmd/koi` (see `.air.toml`) plus Vite. Do not serve dev UI from `:8080`; that is the embedded bundle.

## Layering

```
HTTP                pkg/koiapi/*_handler.go      (chi routes, DTO decode, JSON)
  ↓
Domain              pkg/koidomain/*.go           (entities, Commander/Querier interfaces, Store)
  ↓
Persistence         pkg/koidatabase/gorm_repo_*.go  (GORM implementations of Store)
```

Handlers receive querier and commander via constructor. No global state.

Cookie session auth exists (`scs`, `koi_session`, `middlewares.Authenticate`). Keep constructors injectable. Do not add Keycloak or JWT.

Shared HTTP responses live in `koiapi/response` (import alias `apiresponse`). Small `koiapi.Get`, `List`, `Create`, `Update`, `Delete` adapters share only transport handling. Domain commands and cookie handlers stay explicit. See `docs/architecture.md` for current contracts.

## Stack (from `go.mod`)

- HTTP router: `chi/v5` + `chi/cors`, `chi/render` (JSON).
- Sessions: `scs/v2` + `scs/postgresstore`.
- ORM: GORM (Postgres driver).
- Config: env vars with `KOI_*` prefix, `godotenv` for `.env`.
- Logging: stdlib `slog` (`KOI_LOG_LEVEL`, `KOI_LOG_FORMAT=text|json`).
- Testing: `stretchr/testify` + `vektra/mockery/v3` (`make mocks`).
- Misc: `google/uuid`, argon2id passwords.

## Handler pattern (`pkg/koiapi/`)

Handlers receive domain interfaces. `server.go` applies authentication once to the
protected route group and injects the identity resolver. Resource handlers do not
receive the session manager; auth and brand preference handlers still manage cookies
or session state. Billing guards consume the verified organization identity.

HTTP DTOs explicitly map public camelCase fields without changing persistence
serialization. Collections require `brandId`; creates and collection commands carry
it in the body. Details and mutations take their UUID alone. Foreign resource IDs
return 404 through repository scope. `active_brand_id` is only a UI preference.

Health is an exception: no domain entity, handler pings Store directly.
Public JSON lives under `/api/v1`; `GET /health` is unprefixed.

## Errors

Domain returns `*koidomain.Error` with a stable `Code` and optional `Fields`. UI owns user-facing copy; the API contract is the code (and field codes), not the message.

Sentinels (`ErrValidation`, `ErrEmailTaken`, …) stay for `errors.Is`. HTTP mapping lives only in `pkg/koiapi/response`:

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
| `email_taken` | 409 |
| `invalid_credentials`, `unauthorized` | 401 |
| `not_found` | 404 |
| anything else | 500 `{ code: "internal", message: "internal error" }` (log the real error) |

`invalid_credentials` has no fields (do not leak which of email/password). Unique-violation on `users.email` maps to `email_taken` in `koidatabase`.

## Logging

`koiapp` sets `slog` from env. Chi `RequestID` stays. `middlewares.RequestLogger` logs `method`, `path`, `status`, `duration_ms`, `request_id` (no query, no body). `middlewares.Recoverer` logs panics. 5xx from `WriteDomainError` are logged; 4xx are not. GORM stays at `Warn` with `IgnoreRecordNotFoundError` and `ParameterizedQueries` (not-found is control flow; never interpolate bound values such as emails).

## Domain (`pkg/koidomain/`)

Entities embed `koidomain.BaseEntity`. Postgres assigns `id` (`gen_random_uuid()`). GORM fills `CreatedAt` / `UpdatedAt` (`autoCreateTime` / `autoUpdateTime`). Timestamps are `json:"-"`; commanders do not set ID or timestamps. GORM migrates these same structs (no parallel row type).

```go
type BaseEntity struct {
    ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    CreatedAt time.Time `json:"-" gorm:"not null;autoCreateTime"`
    UpdatedAt time.Time `json:"-" gorm:"not null;autoUpdateTime"`
}
```

Each entity file declares the GORM-mapped struct, `TableName()`, `Validate()`, and a repository interface. Commanders/queriers are interfaces with unexported implementations (`NewAuthCommander(store) AuthCommander`).

`Store` (`pkg/koidomain/store.go`) aggregates repos and exposes `Atomic(ctx, fn)` for transactions:

```go
type Store interface {
    Atomic(context.Context, func(Store) error) error
    Ping(context.Context) error
    UserRepo() UserRepository
    OrganizationRepo() OrganizationRepository
    MembershipRepo() MembershipRepository
}
```

## Repositories (`pkg/koidatabase/`)

GORM-backed. Name files `gorm_repo_<entity>.go`. Entity repos embed `*Repository[T]` (`Create` / `Get` by UUID). Domain interfaces embed `BaseEntityRepository[T]`; list filters are resource-specific. `mapDBError` turns `ErrRecordNotFound` into `ErrNotFound` and email unique-violations into `ErrEmailTaken`.

`pkg/koidatabase` implements `koidomain.Store`. `Atomic` opens a GORM transaction and passes a store bound to that tx.

## Tests

Generate doubles with `make mocks` (or `go generate ./pkg/koidomain`) into `*_test.go` in the consuming package (`pkg/koidomain/mocks_test.go`, `pkg/koiapi/mocks_test.go`). Same-package mocks avoid an import cycle with `koidomain`. Domain tests mock `Store` + repos (helper for `Atomic` to invoke the callback). Handler tests mock `AuthCommander` / `AuthQuerier`. Health mocks `Store.Ping` only. Commit generated mocks. No live Postgres in unit tests.

## Server wiring (`pkg/koiapi/server.go`)

Compose domain services, resource handlers and middleware here. Authentication
resolves membership into `koiauth.Identity{UserID, OrganizationID, Role}` once per
request. Do not put resource authorization in generic HTTP adapters; scoped
repositories enforce tenant access on every read and mutation.

## Lists and integration tests

Persisted-resource `GET` lists accept `page` and `pageSize` and return `Page[T]`, including empty results with `items: []`, `totalItems: 0`, `totalPages: 0`, `currentPage: 1`, `hasNext: false`, and `hasPrev: false`. Keep OpenAPI and UI wrappers aligned. Country options and competitor search are bounded combobox lookups.

Postgres integration tests call `koidatabase.Open(testdb.URL(t))`, then run only the migrations that test needs. `internal/testdb` reads `KOI_DATABASE_URL` from the environment or repo `.env`, creates and cleans up a schema in that database, and fails if Postgres is unavailable. It does not migrate, because some tests verify behavior before River tables exist.

Break up nested Go literals even when `gofmt` accepts one line. Keep resource handlers and domain operations explicit. For external APIs, inspect installed clients, read official vendor docs, and verify code against the pinned SDK version. Use the installed OpenRouter SDK for Responses API and Resty for direct JSON HTTP integrations without a suitable SDK. Nontrivial plans include concrete type or call examples, data flow, compatibility choices, failure cases, and acceptance tests.
