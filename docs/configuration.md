# Configuration

Koi reads its settings from environment variables. With the [installer](self-hosting.md), they
live in `koi/.env`; apply a change with `docker compose up -d`. Empty values count as unset.

## Required

| Variable | Default | Purpose |
| --- | --- | --- |
| `KOI_IMAGE_TAG` | set by the installer | Release image to run, such as `v0.2.0`. |
| `POSTGRES_PASSWORD` | set by the installer | Database password. Use hex: `openssl rand -hex 24`. |
| `KOI_CLORO_API_KEY` | none | Your [Cloro](https://cloro.dev) key. Without it, Koi runs but collects no live answers. |

## Public address

| Variable | Default | Purpose |
| --- | --- | --- |
| `KOI_PUBLIC_URL` | none | HTTPS origin of the instance, such as `https://koi.example.com`. Marks the session cookie `Secure`, adds links to emails, and lets remote agents complete OAuth. HTTP is accepted only for `localhost`. |
| `KOI_MCP_PUBLIC_URL` | `KOI_PUBLIC_URL` | Public origin of `/mcp`, when it differs from the UI. See [MCP](mcp.md#hosting-the-endpoint). |
| `KOI_CLIENT_IP_HEADER` | none | Header your proxy overwrites with the client address: `X-Real-IP` (Nginx) or `CF-Connecting-IP` (Cloudflare). Without it, all visitors share one rate limit. |
| `KOI_HOST_PORT` | `8080` | Host port. Koi binds to `127.0.0.1` only. |
| `KOI_ALLOW_SIGNUP` | `false` | Set `true` to let more people sign up. Only the first user can sign up by default. |

## Optional services

Each service is off until you set its key. The core workflow works without all of them.

| Variable | Service | Without it |
| --- | --- | --- |
| `KOI_OPENROUTER_API_KEY` | [OpenRouter](https://openrouter.ai): researches your brand, drafts prompts and competitors, and checks that an answer naming your brand is about your product. | You write prompts and competitors yourself. A brand named in an answer counts as a mention by name match. |
| `LOGODEV_SECRET_KEY` | [Logo.dev](https://logo.dev) company search, server side. | You add competitors by name and domain. |
| `VITE_LOGODEV_PUBLISHABLE_KEY` | Logo.dev logos. The `pk_` key is sent to the browser. | Competitors show initial-letter icons. |

### Email

Set both after verifying a sending domain in [Resend](https://resend.com).

| Variable | Purpose |
| --- | --- |
| `KOI_RESEND_API_KEY` | Resend API key. |
| `KOI_EMAIL_FROM` | Sender, such as `Koi <hello@mail.example.com>`. |

With email set, Koi sends a welcome email and password reset emails. Both need `KOI_PUBLIC_URL`
for their links. Without email, sign-up works and you
[reset passwords on the server](self-hosting.md#reset-a-password).

## Advanced

The self-hosted Compose file does not pass these. Add them to its `environment` block, or set
them when you run the binary.

| Variable | Default | Purpose |
| --- | --- | --- |
| `KOI_DATABASE_URL` | set by Compose | PostgreSQL URL. Required for the binary. |
| `KOI_PORT` | `:8080` in Compose | Listen address. Required for the binary. |
| `KOI_SESSION_LIFETIME` | `8h` | How long a login lasts. |
| `KOI_WORKER_CONCURRENCY` | `4` | Workers per job queue. |
| `KOI_PROVIDER_TIMEOUT` | `3m` | Timeout for each Cloro or OpenRouter call. |
| `KOI_DB_MAX_OPEN_CONNS` | `12` | Database connection pool size. |
| `KOI_LOG_LEVEL` | `info` | `debug`, `info`, `warn` or `error`. |
| `KOI_LOG_FORMAT` | `json` in Compose | `text` or `json`. |

`KOI_DEPLOYMENT_MODE` stays `selfhosted`. The `cloud` mode, Stripe and PostHog variables in
[`.env.example`](../.env.example) are for Koi Cloud only.
