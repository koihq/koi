# Koi

Developer- and agent-first AEO tool: measure whether ChatGPT, Gemini, Google AI Mode, and Google AI Overviews mention or cite a brand, who is recommended instead, and hand a markdown brief to the founder’s existing agent.

## Layout

```
cmd/koi/            Go binary
pkg/app/            boot
pkg/api/            chi HTTP
pkg/domain/         entities, Store
pkg/database/       GORM
ui/                 React SPA (pnpm, Vite)
ui/site/            Public landing at getkoi.co: separate static build, not embedded in the binary
docs/               User docs (self-hosting, configuration, MCP, troubleshooting) + architecture, OpenAPI
install.sh          Self-host installer: downloads the release's Compose file and .env, starts Koi
skills/             Public agent skills (koi-setup, koi-brief), installed with `npx skills add koihq/koi`
```

README and docs are for users: keep them short and concrete. Internal plans and runbooks stay out
of the repository. Self-host variables live in `.env.selfhosted.example`,
`docker-compose.selfhosted.yml` and `docs/configuration.md`; change all three together.

## Commands

```bash
cp .env.example .env
make dev                # Postgres + Air API (:8080) + Vite HMR (:5173)
make compose-up          # Postgres
make ui-build            # Vite → pkg/web/dist (embedded by Go)
make run                 # Go API + embedded UI on :8080
make site-dev            # Landing with HMR on :5174
make site-build          # Prerendered landing → ui/site/dist
make test                # go test ./...
pnpm --dir ui test       # Vitest: unit (node) + browser (Chromium, MSW) projects
pnpm --dir ui test:e2e   # Playwright full-stack journeys: real binary, UI and Postgres on :8090
make format              # Biome safe fixes + gofmt
make go-format-check     # fail if Go files need formatting
```

`pnpm --dir ui install` registers Git hooks. `ui/.husky/pre-commit` runs `go-format-check`, then UI `check` (formatting, lint, imports), `check-types`, and unit tests (`vitest run --project unit`); browser and e2e tests run in CI. Checks fail without rewriting files. Type checking uses `tsc -p tsconfig.json --noEmit`; build-info caches are local artifacts and must not be committed.

Browse `http://localhost:5173` during `make dev`. Vite proxies `/health` and `/api` to the Go API. `:8080` in dev serves the last embedded build — do not use it for UI work.

`GET /health` returns `{ "status": "ok", "db": "ok"|"down" }`.

## Self-hosting rule

Self-hosted live visibility collection requires the instance owner's Cloro account and API key.
OpenRouter drafting and Logo.dev search/logos are optional. Keep built-in manual prompt and
competitor management, initial-letter icons, and cookie authentication usable without those
optional services. Do not rely on Koi Cloud to supply credentials or services to a self-hosted
instance. New integrations beyond Cloro must remain optional for the core experience.
Self-hosted sign-up admits only the first user unless `KOI_ALLOW_SIGNUP=true`, because every user
spends the owner's Cloro credits. `koi admin reset-password` recovers an account without email.

Koi applies goose migrations from `pkg/database/migrations` at startup, under an advisory lock.
`00001_init.sql` is the schema of the first release. Every schema change after it is a new
migration (`make migration name=...`) that runs on existing data; never edit a released one.
`TestMigrationsMatchModels` fails when the GORM structs and the migrations disagree. Do not add
compatibility code for historical job kinds.

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
- Cookie session auth (email/password, `koi_session`). Sessions live in Postgres, so the cookie holds only a random token and needs no secret. Browse `:5173` during `make dev`. Remote MCP and OAuth are available; CLI comes later.
- Before adding an endpoint, MCP tool, or UI API wrapper, find the closest existing implementation. Compare contracts, validation, errors, auth, and tests across affected surfaces; follow `codebase-patterns` for the specific checks.

## API and implementation conventions

- Architecture and examples: `docs/architecture.md`. Product APIs are flat: collections require `brandId` in query, creates and collection commands in the body; detail and mutation routes use the resource UUID alone. `active_brand_id` is only a session UI preference.
- HTTP DTOs use camelCase, including error field names. Keep persisted snapshot JSON and vendor payloads unchanged. Shared responses live in `pkg/api/response`; generic HTTP adapters stay in `api` and carry no business policy.
- Authentication resolves membership once per request into `auth.Identity` with user, organization and role. Scoped repositories reject a missing organization. One organization per user is enforced in the DB; an organization can have multiple members and brands.
- Product rules and transactions live in domain commanders and entity methods; repositories stay thin and declare their scope, filter and writable columns. A change to what a brand tracks runs under `withBrandLock` or `withLockedChild`. Worker state machines keep their transactions in `database`.
- Keep the intentional absence of a live-prompt FK on historical results. Model live parent relations with GORM associations and explicit delete policy.
- OpenAPI generates request objects (`useSingleRequestParameter=true`). Share `queryOptions` between loaders and hooks, normalize HTTP errors in the client middleware, and clear queries on account changes.

- Persisted-resource `GET` lists accept `page` and `pageSize` and return `Page[T]`, even when empty: `items: []`, `totalItems: 0`, `totalPages: 0`, `currentPage: 1`, `hasNext: false`, `hasPrev: false`. Keep OpenAPI and UI wrappers aligned. Country options and competitor search are bounded combobox lookups, so they are exceptions.
- Put shared UI blocks in `ui/src/components/blocks/`, shared hooks in `ui/src/hooks/`, and shared query keys in `ui/src/lib/query-keys.ts`. Route-private code stays under a route's `-` folders.
- Live visibility runs collect engine answers and sources through Cloro. OpenRouter is for onboarding research, prompt/competitor drafting, and judging whether an answer that names the brand is really about it; without a key, a named brand stays a mention by name match. Keep engine identity (ChatGPT, Gemini, Google AI Mode, Google AI Overviews) separate from the collection provider (stub or Cloro).
- For external APIs, inspect installed clients, read official vendor documentation, and verify the implementation against the pinned SDK version. Use the installed OpenRouter SDK for Responses API; use the repo's Resty pattern for direct JSON HTTP integrations without a suitable SDK.
- Nontrivial plans include concrete type or call examples, data flow, compatibility choices, failure cases, and acceptance tests.
- Break up nested Go literals for readability even when `gofmt` accepts a one-line form.
- Postgres integration tests run during `go test ./...`. `internal/testdb.URL(t)` reads `KOI_DATABASE_URL` from the environment or repo `.env`, isolates a schema in that database, and fails if Postgres is unavailable. Tests explicitly run their own migrations.

## Tests

- Before adding or changing a test, find the existing tests for that behavior (same package or
  route, same handler, entity, hook, or e2e journey) and read them.
- Prefer strengthening an existing test: add the missing assertion, case, or table row to the
  test that already sets up that behavior. Add a new test only when no existing test covers the
  scenario, and say which tests you checked.
- Test each behavior at the lowest layer that catches the bug with real collaborators: unit for
  pure logic, Postgres/HTTP integration for Go, Vitest Browser Mode with MSW for UI routes, and
  Playwright only for full-stack journeys. Do not duplicate the same assertion across layers.
- Refactor existing tests when it makes them clearer: merge near-duplicates into table-driven
  tests, move repeated setup into shared helpers and fixture factories, and delete tests made
  redundant by a stronger one. Keep behavior coverage equal or better, and name the test that now
  covers each deleted case.
- Reuse the shared helpers. Go: `internal/testdb`, `newIntegrationServer` and `signup` in
  `pkg/api/integration_helpers_test.go`, and `newCookieClient`, whose transport fails a test when
  a documented response violates `docs/openapi.yaml` (`pkg/api/contract_test.go`). UI:
  `renderRoute` mounts the real router, `serveWorkspace` in `ui/src/test/worker.ts` is the fake
  API with mutable state and named failures, and `ui/src/test/fixtures.ts` builds typed responses
  from the generated client. A browser test fails on any `/api` request without an MSW handler.
- UI tests are `*.test.ts` (logic, node) or `*.test.tsx` (components and routes, Chromium). Route
  tests live beside the route with a `-` prefix so the router ignores them. Playwright in
  `ui/e2e/` covers only journeys that need the real server; `internal/e2e/server` runs it on an
  isolated schema with optional services off.

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
