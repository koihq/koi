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

Browse `http://localhost:5173` during `make dev`. Vite proxies `/health` and `/api` to the Go API. `:8080` in dev serves the last embedded build — do not use it for UI work.

`GET /health` returns `{ "status": "ok", "db": "ok"|"down" }`.

## Agent rules

- Before Go API, domain, or GORM work, read `.cursor/skills/codebase-patterns` (`backend.md`).
- Before React routes, API wrappers, or UI features, read `.cursor/skills/codebase-patterns` (`frontend.md`).
- `ui/` uses **pnpm**. Do not add `package-lock.json` or `yarn.lock`. Imports use `@/` (not `#/`).
- Pin every dependency to an exact version; prefer LTS when the project publishes one. See `.cursor/rules/pin-dependencies.mdc`.
- No auth in this codebase yet. Dual-surface CLI/MCP comes after the HTTP + UI loop works.
