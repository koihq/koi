# Koi

Developer- and agent-first AEO tool. When buyers ask ChatGPT, Perplexity, Gemini, or Google AI Overviews about your category, it shows whether you are mentioned or cited, who is recommended instead, and a markdown brief your existing agent can execute.

See [ROADMAP.md](ROADMAP.md) for MVP slices.

## Run locally

```bash
cp .env.example .env
make dev
```

Set `KOI_SESSION_SECRET` in `.env` to a random value (`openssl rand -base64 32`).
Live visibility collection requires a Cloro account and `KOI_CLORO_API_KEY`, including
self-hosted installations. OpenRouter is optional: set `KOI_OPENROUTER_API_KEY` to enable
onboarding research and prompt/competitor drafting. Without it, curate these lists manually.

Logo.dev keys are optional: without them, add competitors by name and domain in the UI
and use initial-letter icons. Set `LOGODEV_SECRET_KEY` to enable company search;
set `VITE_LOGODEV_PUBLISHABLE_KEY` before building the UI to show logos.

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

## Workers and scheduling

River runs jobs from Postgres; queue discovery polls approximately every second with jitter.
`KOI_WORKER_CONCURRENCY` defaults to four workers per enabled generation/visibility queue.
A separate maintenance queue uses one worker and runs on startup, then every minute to
reconcile interrupted runs and dispatch visibility work.

Finishing both prompt and competitor reviews starts tracking. Each brand runs every 24 hours
from its persisted UTC anchor; missed slots are skipped rather than collected in a burst.
A visibility job processes cells for approximately 40 seconds, persists Cloro task IDs and
answers, then snoozes for ten seconds if work remains. Actual resumption also depends on
River's scheduler and available workers. This is a bounded loop inside a queued job.

Generation runs when a job is queued. `KOI_PROVIDER_TIMEOUT` defaults to three minutes per
provider call. Generation makes two sequential calls, so its job timeout is twice that value
plus 30 seconds (6m30s by default). Retry delays after failures are five and twenty seconds;
there are at most three attempts. Graceful shutdown allows jobs 60 seconds before cancellation.

The project currently initializes a fresh database through GORM entity definitions and River's
migrations. It does not provide upgrades from earlier development schemas.
