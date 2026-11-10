# Koi

Developer- and agent-first AEO tool. When buyers ask ChatGPT, Gemini, Google AI Mode, or Google AI Overviews about your category, it shows whether you are mentioned or cited, who is recommended instead, and a markdown brief your existing agent can execute.

See [ROADMAP.md](ROADMAP.md) for MVP slices.

## Run locally

```bash
cp .env.example .env
make dev
```

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

Password recovery uses the same email setup and also requires `KOI_PUBLIC_URL`. Without
them, the recovery page says it is not set up on that instance.

Dev serves the UI with Vite HMR at `http://localhost:5173` and the API (Air) at `:8080`. Vite proxies `/health` and `/api` to the API. Check `GET http://localhost:5173/health` — `{ "status": "ok", "db": "ok"|"down" }`.

To run the production-shaped binary (embedded UI, no HMR):

```bash
make compose-up
make run
```

API listens on `:8080`.

To build and run the Docker image locally with PostgreSQL, run
`make compose-image`. The image profile
is separate from `make dev` so the Go API does not claim port 8080 twice.
Use `make compose-down` to stop it.

```bash
make test
```

The React app lives in `ui/` (pnpm). `make ui-build` writes the Vite bundle to `pkg/web/dist` (gitignored); Go embeds it. `make dev`, `make run`, and `make test` build the UI first if that output is missing.

## Demo data

With local Postgres running (`make compose-up`), seed a dedicated demo account:

```bash
make seed-demo
```

Log in at `http://localhost:5173` with **demo@koi.local** / **KoiDemo2026!**.
Use the existing brand selector to switch between these US-market examples:

| Brand | Dataset |
| --- | --- |
| Koi · History | 180 daily collections, 32 current prompts, 2 historical-only prompts, 6 competitors |
| Koi · Empty | Brand profile only; no prompts, competitors, or collections |
| Koi · First check | 5 confirmed prompts, 4 competitors, one completed collection today (20 answers) |

The populated example includes 26 active confirmed prompts, 3 inactive prompts,
and 3 unconfirmed drafts. Inactive and deleted prompts have answers only in the
first 90 days. Four engines, citations, mentions, misses, simulated provider
errors, and varying trends exercise pagination, filters, prompt history, and
7/30/90-day comparisons. Days without observations have no rate, rather than 0%.
The demo account represents all three brand states; the empty brand starts after
brand creation, and the first-collection brand starts after configuration.

Repeated seeding preserves existing scenarios and creates missing ones. To restore
all three demo brands and move their dates to today:

```bash
make seed-demo-reset
```

Reset replaces only those three brands and their contents, including edits made
inside them. It preserves the account, login, sessions, other brands, and other
users. An occupied demo email or unexpected organization aborts without changes.
Keep this account for local development; its password is public.

Both run `koi admin seed-demo`, which loads `.env`, requires only `KOI_DATABASE_URL`, and
applies pending migrations. It works in `selfhosted` mode (the default). In `cloud` mode it
runs only with `KOI_LIVE_COLLECTION=false`, and it also gives the demo organization an active
50-prompt plan that Stripe does not know about, so the brands show instead of the paywall;
billing actions fail on that account. The Koi Cloud staging instance seeds itself this way after
each deploy.
Start the app normally with `make dev`. Seeding needs no running API,
Cloro, OpenRouter, Logo.dev, or other service credentials. All results use the
`stub` collector and disclose that their answers are simulated. No jobs, welcome
emails, provider requests, or future collections are scheduled, even if Cloro is
configured. The full and first-collection brands deliberately have no next run;
normal collection-availability messages still reflect the instance configuration.

### Dataset sources

Product identities and broad capabilities were checked on **2026-11-02** against
the [Koi repository](https://github.com/koihq/koi),
[Profound](https://www.tryprofound.com/), [Peec AI](https://peec.ai/),
[OtterlyAI](https://otterly.ai/), [Scrunch](https://scrunch.com/),
[AthenaHQ](https://athenahq.ai/), and [Rankscale](https://rankscale.ai/).
The first-collection brand uses the first four competitors. Koi uses its GitHub
repository identity without an invented website domain. Brand snapshots retain
the name `Koi`; selector names add scenario labels.

The dataset in `internal/demoseed/dataset.go` contains original illustrative
answers, not copied vendor text or engine transcripts. Dates, recommendations,
citations, rankings, trends, and metrics are simulated, not evidence of actual
engine behavior. Generation is deterministic relative to the seed date, and the
existing visibility scorer derives outcomes from the saved text and source URLs.

## Workers and scheduling

River runs jobs from Postgres; queue discovery polls approximately every second with jitter.
`KOI_WORKER_CONCURRENCY` defaults to four workers per enabled generation/visibility queue.
A separate maintenance queue uses one worker and runs on startup, then every minute to
reconcile interrupted runs and dispatch visibility work.

Each brand gets one AI draft: at most 5 prompts and every competitor it can verify. After
that, add entries by hand or through your agent over MCP. "Start tracking" confirms the lists
once and starts collection; later changes never pause it and apply from the next check. Each
brand runs every 24 hours
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
individual confirmation. Starting tracking excludes observed suggestions, and deleting one suppresses
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
```

Put that generated value and a published `KOI_IMAGE_TAG` (for example `v0.1.0`)
in `.env.selfhosted`. Set `KOI_PUBLIC_URL` to your HTTPS address, because without it the session
cookie is not marked `Secure`. Set `KOI_CLIENT_IP_HEADER` to the header your proxy overwrites with
the client's address, or all visitors share one rate limit. `POSTGRES_PASSWORD` must be hex because Compose uses it in a
PostgreSQL URL. Add `KOI_CLORO_API_KEY` to collect live answers. OpenRouter and
Logo.dev are optional; manual prompt and competitor management work without them.

Only the first user can sign up, so nobody else can spend your Cloro credits. To let someone in,
set `KOI_ALLOW_SIGNUP=true`, restart, and turn it off again when they have signed up.

Without Resend there is no reset email. To recover a password, run this on the server. It also
signs out every session and connected agent:

```bash
printf '%s\n' 'the-new-password' | docker compose --env-file .env.selfhosted \
  -f docker-compose.selfhosted.yml exec -T koi koi admin reset-password person@example.com
```

[docs/data-requests.md](docs/data-requests.md) covers requests to export or delete personal data.

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
Koi applies pending schema migrations when it starts, so an upgrade needs no extra step.
Going back to an older image is safe only if that version supports the current schema;
otherwise restore the backup you took before upgrading.

Each [release](https://github.com/koihq/koi/releases) also ships `koi` binaries for Linux and
macOS (amd64 and arm64), with the UI embedded, and a checksums file. They take the same
environment variables as the image.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) and the [code of conduct](CODE_OF_CONDUCT.md). Report
security problems privately, as described in [SECURITY.md](SECURITY.md).

## License

[MIT](LICENSE)
