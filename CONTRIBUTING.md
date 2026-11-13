# Contributing to Koi

Thanks for helping. Koi is a Go API with an embedded React UI, backed by PostgreSQL. Bug reports,
fixes, and small focused improvements are the easiest to review. Open an issue before starting a
large change, so we can agree on the approach first.

Report security problems privately, as described in [SECURITY.md](SECURITY.md), not in an issue.

## Set up

You need Go (the version in `go.mod`), Node 24 with pnpm (the version in `ui/package.json`), and
Docker for PostgreSQL.

```bash
cp .env.example .env
pnpm --dir ui install    # also installs the pre-commit hook
make dev                 # PostgreSQL, API on :8080, UI with HMR on http://localhost:5173
make seed-demo           # optional: a demo account with sample data
```

Browse `http://localhost:5173`. Vite proxies `/api` and `/health` to the API; `:8080` serves the
last embedded build, so do not use it for UI work. `GET /health` returns
`{ "status": "ok", "db": "ok"|"down" }`.

No external service is required for development. Live collection needs a
[Cloro](https://cloro.dev) key; OpenRouter, Logo.dev, Resend, Stripe and PostHog are optional.
For MCP with Vite, set `KOI_PUBLIC_URL=http://localhost:5173` and
`KOI_MCP_PUBLIC_URL=http://localhost:8080`.

Other targets:

```bash
make run                 # production-shaped binary with the embedded UI on :8080
make compose-image       # build the Docker image and run it with PostgreSQL
make site-dev            # public landing (ui/site) with HMR on :5174
pnpm --dir ui test:e2e   # Playwright journeys against the real binary on :8090
```

## Before you open a pull request

```bash
make format              # Biome and gofmt fixes
make test                # Go tests, including PostgreSQL integration tests
pnpm --dir ui check && pnpm --dir ui check-types && pnpm --dir ui test
```

The pre-commit hook runs the same checks without rewriting files. CI runs them again, with
`go vet`, `govulncheck` and a Docker image build.

- Schema changes need a new migration: `make migration name=add_something`. Never edit a migration
  that has been released.
- Add an endpoint, MCP tool or UI API wrapper next to the closest existing one, and keep the
  OpenAPI spec (`docs/openapi.yaml`) and generated client (`make openapi`) in sync.
- Pin exact dependency versions. The UI uses pnpm only.
- [AGENTS.md](AGENTS.md) and [docs/architecture.md](docs/architecture.md) describe the layering,
  API conventions and readability rules. Coding agents read the same file.

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

## Pull request titles

PRs are squash-merged, and the title becomes the commit message on `main`. Release Please reads
those messages to choose the next version and write the changelog, so titles follow
[Conventional Commits](https://www.conventionalcommits.org/):

```
feat: show citation gaps per engine
fix(billing): keep the trial end after a plan change
docs: explain self-hosted backups
```

`feat` is a minor release and `fix` is a patch. Add `!` after the type for a breaking change. A
check fails when the title does not match.

## License

By contributing, you agree that your contribution is licensed under the [MIT License](LICENSE).
