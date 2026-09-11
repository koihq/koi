# Koi

Developer- and agent-first AEO tool. When buyers ask ChatGPT, Perplexity, Gemini, or Google AI Overviews about your category, it shows whether you are mentioned or cited, who is recommended instead, and a markdown brief your existing agent can execute.

See [ROADMAP.md](ROADMAP.md) for MVP slices.

## Run locally

```bash
cp .env.example .env
make compose-up
make run
```

API listens on `:8080`. Check `GET /health` — `{ "status": "ok", "db": "ok"|"down" }`.

```bash
make test
```

The React app lives in `ui/` (pnpm). Scaffold it with Vite; Go embeds a built bundle from `pkg/web/dist` when present.
