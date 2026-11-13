# Self-hosting

Koi runs as one container next to PostgreSQL. You need a machine with Docker and Docker Compose
v2. For remote access you also need an HTTPS reverse proxy, such as Caddy, Nginx or Traefik.

Live answers come from [Cloro](https://cloro.dev), so live collection needs your own Cloro API
key. Everything else is optional: without other keys you add prompts and competitors by hand and
competitors get initial-letter icons. Koi Cloud never supplies credentials to a self-hosted
instance.

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/koihq/koi/main/install.sh | sh
```

The [installer](../install.sh):

1. Checks that Docker and Docker Compose are available.
2. Creates `./koi` with `docker-compose.yml` and `.env` from the latest
   [release](https://github.com/koihq/koi/releases).
3. Generates a database password and sets the image tag in `.env`.
4. Starts PostgreSQL and Koi, then waits until both are healthy.

Open `http://localhost:8080` and create your account. The first account owns the instance.

Pass settings as environment variables to skip editing `.env` afterwards:

```sh
curl -fsSL https://raw.githubusercontent.com/koihq/koi/main/install.sh |
  KOI_CLORO_API_KEY=your-key KOI_PUBLIC_URL=https://koi.example.com sh
```

`KOI_DIR` changes the install directory and `KOI_VERSION` installs a specific tag, such as
`v0.2.0`. Running the installer again in the same directory changes nothing.

To change a setting later, edit `koi/.env` and apply it:

```sh
cd koi
docker compose up -d
```

## Install manually

Use this path to review each file or to keep Koi inside an existing Compose project.

1. Download `docker-compose.selfhosted.yml` and `.env.selfhosted.example` from the release you
   want, and save them as `docker-compose.yml` and `.env`.
2. In `.env`, set `KOI_IMAGE_TAG` to that release tag, such as `v0.2.0`.
3. Set `POSTGRES_PASSWORD` to the output of `openssl rand -hex 24`. It must be hex, because
   Compose places it in a PostgreSQL URL.
4. Add `KOI_CLORO_API_KEY` and any [optional settings](configuration.md).
5. Start Koi:

   ```sh
   docker compose up -d --wait
   ```

## Put Koi behind HTTPS

Koi listens on `127.0.0.1:8080` only. Point your reverse proxy at that address and do not
publish PostgreSQL. `KOI_HOST_PORT` changes the host port.

Then set these in `.env`:

- `KOI_PUBLIC_URL`: your HTTPS address, such as `https://koi.example.com`. Without it, the
  session cookie is not marked `Secure`, emails have no links, and remote agents cannot complete
  OAuth.
- `KOI_CLIENT_IP_HEADER`: the header your proxy overwrites with the client address, such as
  `X-Real-IP` for Nginx or `CF-Connecting-IP` behind Cloudflare. Without it, every visitor
  shares one rate limit. Set it only when the proxy overwrites the header.

A minimal Caddy site:

```caddy
koi.example.com {
  reverse_proxy 127.0.0.1:8080
}
```

## Users

Only the first user can sign up, because every user spends your Cloro credits. To let someone
in, set `KOI_ALLOW_SIGNUP=true`, run `docker compose up -d`, and remove it again after they have
signed up.

## Reset a password

Password reset emails need Resend (see [configuration](configuration.md#email)). Without it,
reset a password on the server. This also signs out every session and connected agent:

```sh
printf '%s\n' 'the-new-password' |
  docker compose exec -T koi koi admin reset-password person@example.com
```

## Upgrade

Back up the database first. Then set `KOI_IMAGE_TAG` in `.env` to the new
[release](https://github.com/koihq/koi/releases) and restart:

```sh
docker compose pull
docker compose up -d --wait
```

Koi applies database migrations when it starts, so an upgrade needs no other step. Going back to
an older image is safe only if that version supports the current schema. Otherwise, restore the
backup you took before upgrading.

## Back up and restore

The `koi_pg` volume holds all data. Dump it while Koi runs:

```sh
docker compose exec -T postgres pg_dump -U koi -d koi -Fc > koi-$(date +%F).dump
```

Restore into a running stack:

```sh
docker compose stop koi
docker compose exec -T postgres pg_restore -U koi -d koi --clean --if-exists < koi-2026-11-13.dump
docker compose start koi
```

Keep a copy off the machine. A backup on the same disk does not survive the loss of that disk.

## Run without Docker

Each [release](https://github.com/koihq/koi/releases) ships `koi` binaries for Linux and macOS
(amd64 and arm64) with the UI embedded, plus a checksums file. Run the binary with
`KOI_DATABASE_URL` pointing at PostgreSQL 16 or later. It reads the same
[environment variables](configuration.md) as the image.

## Personal data requests

[data-requests.md](data-requests.md) explains how to export or delete a user's data.
