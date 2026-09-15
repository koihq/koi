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

Cookie session auth exists (`scs`, `koi_session`, `Authorizer`). Keep constructors injectable. Do not add Keycloak or JWT.

Shared HTTP: `api.JSON`, `api.Error`, `api.WriteDomainError`, `middlewares.DecodeBody`. Do **not** add generic `api.List` / `api.Create` / `api.Get` or `middlewares.ID` until a `/{id}` resource exists. Auth handlers stay explicit (they also set cookies).

## Stack (from `go.mod`)

- HTTP router: `chi/v5` + `chi/cors`, `chi/render` (JSON).
- Sessions: `scs/v2` + `scs/postgresstore`.
- ORM: GORM (Postgres driver).
- Config: env vars with `KOI_*` prefix, `godotenv` for `.env`.
- Logging: stdlib `slog` (`KOI_LOG_LEVEL`, `KOI_LOG_FORMAT=text|json`).
- Testing: `stretchr/testify` + `vektra/mockery/v3` (`make mocks`).
- Misc: `google/uuid`, argon2id passwords.

## Handler pattern (`pkg/koiapi/`)

Handlers depend on **domain interfaces**, not unexported copies:

```go
type AuthHandler struct {
    querier   koidomain.AuthQuerier
    commander koidomain.AuthCommander
    sess      *scs.SessionManager
}

func (h *AuthHandler) Routes() func(r chi.Router) {
    return func(r chi.Router) {
        r.With(middlewares.DecodeBody[CredentialsReq]()).Post("/signup", h.Signup)
        r.With(middlewares.DecodeBody[CredentialsReq]()).Post("/login", h.Login)
        r.Post("/logout", h.Logout)
        r.With(Authorizer(h.sess)).Get("/me", h.Me)
    }
}
```

Health is an exception: no domain entity, handler pings `Store` directly. Use `api.JSON` for all JSON responses.

Public JSON lives under `/api/v1`. `GET /health` is unprefixed.

## Errors

Domain returns `*koidomain.Error` with a stable `Code` and optional `Fields`. UI owns user-facing copy; the API contract is the code (and field codes), not the message.

Sentinels (`ErrValidation`, `ErrEmailTaken`, …) stay for `errors.Is`. HTTP mapping lives only in `pkg/koiapi/api`:

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

GORM-backed. Name files `gorm_repo_<entity>.go`. Entity repos embed `*Repository[T]` (`Create` / `Get` by UUID). `mapDBError` turns `ErrRecordNotFound` into `ErrNotFound` and email unique-violations into `ErrEmailTaken`.

`pkg/koidatabase` implements `koidomain.Store`. `Atomic` opens a GORM transaction and passes a store bound to that tx.

## Tests

Generate doubles with `make mocks` (or `go generate ./pkg/koidomain`) into `*_test.go` in the consuming package (`pkg/koidomain/mocks_test.go`, `pkg/koiapi/mocks_test.go`). Same-package mocks avoid an import cycle with `koidomain`. Domain tests mock `Store` + repos (helper for `Atomic` to invoke the callback). Handler tests mock `AuthCommander` / `AuthQuerier`. Health mocks `Store.Ping` only. Commit generated mocks. No live Postgres in unit tests.

## Server wiring (`pkg/koiapi/server.go`)

`buildHTTPServer` instantiates commanders, then handlers, then mounts routes on a `chi.Router`. New handlers are registered here.

```go
r.Route("/api/v1/auth", NewAuthHandler(
    koidomain.NewAuthQuerier(store),
    koidomain.NewAuthCommander(store),
    sess,
).Routes())
```
