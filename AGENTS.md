# Koi

Developer- and agent-first AEO tool: measure whether ChatGPT, Perplexity, Gemini, and Google AI Overviews mention or cite a brand, who is recommended instead, and hand a markdown brief to the founder’s existing agent.

## Layout

```
cmd/koi/            Go binary
pkg/koiapp/         boot
pkg/koiapi/         chi HTTP
pkg/koidomain/      entities, Store
pkg/koidatabase/    GORM
ui/                 React SPA (pnpm) — add with Vite
```

## Commands

```bash
cp .env.example .env
make dev                # Postgres + Air API (:8080) + Vite HMR (:5173)
make compose-up          # Postgres
make ui-build            # Vite → pkg/web/dist (embedded by Go)
make run                 # Go API + embedded UI on :8080
make test                # go test ./...
make format              # Biome safe fixes + gofmt
make go-format-check     # fail if Go files need formatting
```

`pnpm --dir ui install` registers Git hooks. `ui/.husky/pre-commit` runs `go-format-check`, then UI `check` (formatting, lint, imports), `check-types`, and unit tests (`vitest run`). Checks fail without rewriting files. Type checking uses `tsc -p tsconfig.json --noEmit`; build-info caches are local artifacts and must not be committed.

Browse `http://localhost:5173` during `make dev`. Vite proxies `/health` and `/api` to the Go API. `:8080` in dev serves the last embedded build — do not use it for UI work.

`GET /health` returns `{ "status": "ok", "db": "ok"|"down" }`.

## Self-hosting rule

Self-hosted live visibility collection requires the instance owner's Cloro account and API key.
OpenRouter drafting and Logo.dev search/logos are optional. Keep built-in manual prompt and
competitor management, initial-letter icons, and cookie authentication usable without those
optional services. Do not rely on Koi Cloud to supply credentials or services to a self-hosted
instance. New integrations beyond Cloro must remain optional for the core experience.

The project is not deployed yet. Initialize the current schema on a fresh database; do not add
compatibility migrations for earlier development schemas or historical job kinds.

## Agent rules

- Use `.agents/skills/ui-plan` when the user asks to explore a UI view or component, compare design directions, or explicitly invokes the skill. A page counts as a view. Routine UI plans and implementation tasks do not require it.
- Before Go API, domain, or GORM work, read `.agents/skills/codebase-patterns` (`backend.md`).
- Before visual or UI work, read `.agents/skills/product-design`, then `.agents/skills/codebase-patterns` (`frontend.md`).
- Before React routes, API wrappers, or UI features, read `.agents/skills/codebase-patterns` (`frontend.md`).
- Before changing primitive motion, also read `.agents/skills/emil-design-eng` and `.agents/skills/animate`.
- For a screen or flow review, load `.agents/skills/better-interface` (it already routes to the installed `better-accessibility`, `better-layout`, `better-writing`, `better-typography`, `better-colors`, `better-ui`).
- For a branch, PR, or uncommitted UI diff, use `.agents/skills/interface-review` instead.
- `ui/` uses **pnpm**. Do not add `package-lock.json` or `yarn.lock`. Imports use `@/` (not `#/`).
- Before an agent runs pnpm install, add, or update, check `pnpm --dir ui store path`. If it resolves inside this repo, request escalation so pnpm can use the normal user store; never set `store-dir` inside this repo.
- Cookie session auth (email/password, `koi_session`). `KOI_SESSION_SECRET` is required (`openssl rand -base64 32`). Browse `:5173` during `make dev`. Dual-surface CLI/MCP comes later.

## API and implementation conventions

- Persisted-resource `GET` lists accept `page` and `pageSize` and return `Page[T]`, even when empty: `items: []`, `totalItems: 0`, `totalPages: 0`, `currentPage: 1`, `hasNext: false`, `hasPrev: false`. Keep OpenAPI and UI wrappers aligned. Country options and competitor search are bounded combobox lookups, so they are exceptions.
- Put shared UI blocks in `ui/src/components/blocks/`, shared hooks in `ui/src/hooks/`, and shared query keys in `ui/src/lib/query-keys.ts`. Route-private code stays under a route's `-` folders.
- Live visibility runs collect engine answers and sources through Cloro. OpenRouter is for onboarding research and prompt/competitor drafting. Keep engine identity (ChatGPT, Perplexity, Gemini, Google AI Overviews) separate from the collection provider (stub or Cloro).
- For external APIs, inspect installed clients, read official vendor documentation, and verify the implementation against the pinned SDK version. Use the installed OpenRouter SDK for Responses API; use the repo's Resty pattern for direct JSON HTTP integrations without a suitable SDK.
- Nontrivial plans include concrete type or call examples, data flow, compatibility choices, failure cases, and acceptance tests.
- Break up nested Go literals for readability even when `gofmt` accepts a one-line form.
- Postgres integration tests run during `go test ./...`. `internal/testdb.URL(t)` reads `KOI_DATABASE_URL` from the environment or repo `.env`, isolates a schema in that database, and fails if Postgres is unavailable. Tests explicitly run their own migrations.

## Readability

Apply these conventions throughout handwritten source, tests, and configuration:

- Separate logical steps with one blank line: setup, queries, transformations, mutations, and return. Separate multiline hook calls from surrounding steps.
- Keep closely related declarations together, and keep an operation beside its error check.
- Expand nested Go literals and long independent assignments onto separate lines. Preserve simultaneous assignments, swaps, and idiomatic value/error pairs.
- Extract a function when it isolates a responsibility; use spacing to make ordinary sequencing readable.
- Use Biome's existing two-space indentation and 100-column width for UI code, and `gofmt` for Go. Logical grouping is a review convention; formatters enforce mechanical consistency.
- Exclude generated clients, route trees, mocks, vendored code, and build outputs from manual readability edits. Keep the existing generated-file formatter exclusions.

Run `make format` to apply safe formatting fixes. CI checks UI formatting, lint, and imports with `pnpm --dir ui check`, and Go formatting with `make go-format-check`, alongside type checking, `go vet`, and tests.

## Pin dependencies

When adding or bumping a dependency (pnpm, Go modules, Docker base images, GitHub Actions):

1. **Pin exact versions.** No `latest`, `*`, `^`, `~`, or floating dist-tags. Write `1.170.35`, not `^1.170.0`.
2. **Choose LTS when it exists.** Runtimes and images that publish LTS (Node, Postgres, Ubuntu, etc.) must use the current LTS line, not Current/nightly/rc.
3. **Libraries without LTS:** pin the stable version just resolved into the lockfile (`pnpm-lock.yaml` / `go.sum`). Do not leave the specifier as `latest`.
4. **Record the pin.** After install, the manifest (`package.json` / `go.mod`) and lockfile must both show the same exact version.

Go: `go get example.com/mod@v1.2.3` (never unversioned `go get example.com/mod`). Docker: `postgres:16`, not `postgres:latest`.
