# Koi

Developer- and agent-first AEO tool. When buyers ask ChatGPT, Perplexity, Gemini, or Google AI Overviews about your category, it shows whether you are mentioned or cited, who is recommended instead, and a markdown brief your existing agent can execute.

See [ROADMAP.md](ROADMAP.md) for MVP slices.

## Run locally

```bash
cp .env.example .env
make dev
```

Set `KOI_SESSION_SECRET` in `.env` to a random value (`openssl rand -base64 32`).
It is currently required by configuration but is not used by the session manager.
Live visibility collection requires a Cloro account and `KOI_CLORO_API_KEY`, including
self-hosted installations. OpenRouter is optional: set `KOI_OPENROUTER_API_KEY` to enable
onboarding research and prompt/competitor drafting. Without it, curate these lists manually.

Logo.dev keys are optional: without them, add competitors by name and domain in the UI
and use initial-letter icons. Set `LOGODEV_SECRET_KEY` to enable company search;
set `VITE_LOGODEV_PUBLISHABLE_KEY` in the running instance to show logos.
The publishable `pk_` key is sent to the browser through the capabilities API;
keep `LOGODEV_SECRET_KEY` on the server only. A missing publishable key keeps
initial-letter icons, including in self-hosted installations.

Welcome emails are optional. Verify a sending domain in Resend, then set both
`KOI_RESEND_API_KEY` and `KOI_EMAIL_FROM` (for example, `Koi <hello@mail.example.com>`).
Set `KOI_PUBLIC_URL` to the browser origin to include a “Set up your brand” link;
without it, the email has no link. Each self-hosted instance uses its own Resend
account and credentials. Without the Resend variables, signup works without email.

Dev serves the UI with Vite HMR at `http://localhost:5173` and the API (Air) at `:8080`. Vite proxies `/health` and `/api` to the API. Check `GET http://localhost:5173/health` — `{ "status": "ok", "db": "ok"|"down" }`.

To run the production-shaped binary (embedded UI, no HMR):

```bash
make compose-up
make run
```

API listens on `:8080`.

To build and run the Docker image locally with PostgreSQL, set
`KOI_SESSION_SECRET` in `.env`, then run `make compose-image`. The image profile
is separate from `make dev` so the Go API does not claim port 8080 twice.
Use `make compose-down` to stop it.

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

Visibility cells show matched competitor names and supporting evidence. Matching uses current
names, aliases, official domains, and GitHub repositories, so edits also update historical links
without changing saved answers or brand outcomes. Live Cloro entities and leading Markdown
links to product homepages or repositories provide conservative discovery: two distinct cells
create an unconfirmed observed suggestion; retries do not count twice. Names found only in
ordinary prose may be missed. Name-only suggestions need a primary domain or repository before
individual confirmation. Finish review excludes observed suggestions, and deleting one suppresses
its rediscovery for that brand without interrupting tracking.

Generation runs when a job is queued. `KOI_PROVIDER_TIMEOUT` defaults to three minutes per
provider call. Generation makes two sequential calls, so its job timeout is twice that value
plus 30 seconds (6m30s by default). Retry delays after failures are five and twenty seconds;
there are at most three attempts. Graceful shutdown allows jobs 60 seconds before cancellation.

The project currently initializes a fresh database through GORM entity definitions and River's
migrations. It does not provide upgrades from earlier development schemas.

Koi Cloud billing setup and sandbox checks are in [docs/billing.md](docs/billing.md).

## Self-host with Docker Compose

Use a released image after the first public release. Docker Compose and an existing
HTTPS reverse proxy are the only host requirements; Koi does not require Coolify or
Cloudflare. Live visibility collection requires your own Cloro account and API key.

```bash
cp .env.selfhosted.example .env.selfhosted
openssl rand -hex 24      # POSTGRES_PASSWORD
openssl rand -base64 32   # KOI_SESSION_SECRET
```

Put those generated values and a published `KOI_IMAGE_TAG` (for example `v0.1.0`)
in `.env.selfhosted`. `POSTGRES_PASSWORD` must be hex because Compose uses it in a
PostgreSQL URL. Add `KOI_CLORO_API_KEY` to collect live answers. OpenRouter and
Logo.dev are optional; manual prompt and competitor management work without them.

```bash
docker compose --env-file .env.selfhosted -f docker-compose.selfhosted.yml up -d --wait
```

Koi listens only on `127.0.0.1:8080` on the host. Point your existing HTTPS
reverse proxy at that address; do not publish PostgreSQL. To update, set
`KOI_IMAGE_TAG` to another published version, pull it, and restart the stack:

```bash
docker compose --env-file .env.selfhosted -f docker-compose.selfhosted.yml pull
docker compose --env-file .env.selfhosted -f docker-compose.selfhosted.yml up -d --wait
```

The `koi_pg` volume holds the database. Back it up before changing versions.
The current schema is intended for a fresh installation; upgrades from earlier
development schemas are unsupported.
