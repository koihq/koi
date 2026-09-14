# Koi

Developer- and agent-first AEO tool. When buyers ask ChatGPT, Perplexity, Gemini, or Google AI Overviews about your category, it shows whether you are mentioned or cited, who is recommended instead, and a markdown brief your existing agent can execute.

See [ROADMAP.md](ROADMAP.md) for MVP slices.

## Run locally

```bash
cp .env.example .env
make dev
```

Dev serves the UI with Vite HMR at `http://localhost:5173` and the API (Air) at `:8080`. Vite proxies `/health` and `/api` to the API. Check `GET http://localhost:5173/health` — `{ "status": "ok", "db": "ok"|"down" }`.

To run the production-shaped binary (embedded UI, no HMR):

```bash
make compose-up
make run
```

API listens on `:8080`.

```bash
make test
```

The React app lives in `ui/` (pnpm). `make ui-build` writes the Vite bundle to `pkg/web/dist` (gitignored); Go embeds it. `make dev`, `make run`, and `make test` build the UI first if that output is missing.
