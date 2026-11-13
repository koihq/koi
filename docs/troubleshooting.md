# Troubleshooting

Run these commands from the install directory (`./koi` by default).

## Check that Koi is running

```sh
docker compose ps
curl -s http://localhost:8080/health
```

A healthy instance returns `{"status":"ok","db":"ok"}`. `"db":"down"` means Koi cannot reach
PostgreSQL. Read the logs:

```sh
docker compose logs --tail 100 koi
docker compose logs --tail 100 postgres
```

Koi exits at startup when a setting is invalid, and the log names the variable. For example,
`KOI_RESEND_API_KEY` and `KOI_EMAIL_FROM` must be set together, and `KOI_PUBLIC_URL` must use
HTTPS unless the host is `localhost`.

## The installer stops

- **Docker is not running.** Start Docker Desktop or the Docker service, then run the installer
  again.
- **Could not find the latest release.** The GitHub API may be rate-limited. Pick a tag from the
  [releases page](https://github.com/koihq/koi/releases) and pass it:
  `KOI_VERSION=v0.2.0`.
- **Port 8080 is in use.** Pass another port, such as `KOI_HOST_PORT=8081`. After an install,
  change `KOI_HOST_PORT` in `.env` and run `docker compose up -d`.
- **Already installed.** The installer never overwrites `.env`. Start the existing stack with
  `docker compose up -d`, or remove the directory to start over. Removing it does not delete the
  database: `docker compose down -v` does.

## No live answers

Live collection needs `KOI_CLORO_API_KEY`. Add it to `.env`, run `docker compose up -d`, and
start tracking again from the brand page. Each brand runs once every 24 hours. Missed slots are
skipped, not collected in a burst.

## Cannot sign up

Only the first user can sign up. Set `KOI_ALLOW_SIGNUP=true` to let someone else in, then remove
it. See [Users](self-hosting.md#users).

## "KOI_PUBLIC_URL is not set" in the logs

Without `KOI_PUBLIC_URL`, the session cookie is not marked `Secure`. That is fine on `localhost`.
For any other address, serve Koi over HTTPS and set `KOI_PUBLIC_URL` to it.

## An agent cannot connect over MCP

The agent needs a public HTTPS URL and `KOI_PUBLIC_URL` set to it. Without a public origin, Koi
advertises its MCP endpoint only to local requests. See [MCP](mcp.md#hosting-the-endpoint).

## Forgot a password

[Reset it on the server](self-hosting.md#reset-a-password).
