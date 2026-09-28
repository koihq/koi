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
```

`pnpm --dir ui install` registers Git hooks. `ui/.husky/pre-commit` runs UI `lint`, `check-types`, and unit tests (`vitest run`).

Browse `http://localhost:5173` during `make dev`. Vite proxies `/health` and `/api` to the Go API. `:8080` in dev serves the last embedded build — do not use it for UI work.

`GET /health` returns `{ "status": "ok", "db": "ok"|"down" }`.

## Agent rules

- Before Go API, domain, or GORM work, read `.agents/skills/codebase-patterns` (`backend.md`).
- Before visual or UI work, read `.agents/skills/product-design`, then `.agents/skills/codebase-patterns` (`frontend.md`).
- Before React routes, API wrappers, or UI features, read `.agents/skills/codebase-patterns` (`frontend.md`).
- Before changing primitive motion, also read `.agents/skills/emil-design-eng` and `.agents/skills/animate`.
- For a screen or flow review, load `.agents/skills/better-interface` (it already routes to the installed `better-accessibility`, `better-layout`, `better-writing`, `better-typography`, `better-colors`, `better-ui`).
- For a branch, PR, or uncommitted UI diff, use `.agents/skills/interface-review` instead.
- `ui/` uses **pnpm**. Do not add `package-lock.json` or `yarn.lock`. Imports use `@/` (not `#/`).
- Before an agent runs pnpm install, add, or update, check `pnpm --dir ui store path`. If it resolves inside this repo, request escalation so pnpm can use the normal user store; never set `store-dir` inside this repo.
- Cookie session auth (email/password, `koi_session`). `KOI_SESSION_SECRET` is required (`openssl rand -base64 32`). Browse `:5173` during `make dev`. Dual-surface CLI/MCP comes later.

## Pin dependencies

When adding or bumping a dependency (pnpm, Go modules, Docker base images, GitHub Actions):

1. **Pin exact versions.** No `latest`, `*`, `^`, `~`, or floating dist-tags. Write `1.170.35`, not `^1.170.0`.
2. **Choose LTS when it exists.** Runtimes and images that publish LTS (Node, Postgres, Ubuntu, etc.) must use the current LTS line, not Current/nightly/rc.
3. **Libraries without LTS:** pin the stable version just resolved into the lockfile (`pnpm-lock.yaml` / `go.sum`). Do not leave the specifier as `latest`.
4. **Record the pin.** After install, the manifest (`package.json` / `go.mod`) and lockfile must both show the same exact version.

Go: `go get example.com/mod@v1.2.3` (never unversioned `go get example.com/mod`). Docker: `postgres:16`, not `postgres:latest`.
