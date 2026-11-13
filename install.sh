#!/bin/sh
# Install a self-hosted Koi with Docker Compose.
#
#   curl -fsSL https://raw.githubusercontent.com/koihq/koi/main/install.sh | sh
#
# The script creates ./koi with docker-compose.yml and .env from the latest release, generates a
# database password, and starts Koi on http://localhost:8080. It never overwrites an existing
# .env. Options are environment variables:
#
#   KOI_DIR       install directory (default: ./koi)
#   KOI_VERSION   release tag to install, such as v0.2.0 (default: latest release)
#   KOI_CLORO_API_KEY, KOI_PUBLIC_URL, KOI_HOST_PORT and the other settings in
#                 .env.selfhosted.example are copied into .env when set
#
# Read docs/self-hosting.md for the manual steps and for HTTPS, upgrades and backups.

set -eu

REPO="koihq/koi"
KOI_DIR="${KOI_DIR:-./koi}"

say() { printf '%s\n' "$*"; }
fail() {
  printf 'koi: %s\n' "$*" >&2
  exit 1
}

# Replace KEY=... in .env, or append it when the line is missing. awk reads the value from the
# environment, so URLs and other special characters need no escaping.
set_env() {
  KEY="$1" VALUE="$2" awk '
    BEGIN { key = ENVIRON["KEY"]; line = key "=" ENVIRON["VALUE"] }
    index($0, key "=") == 1 { print line; found = 1; next }
    { print }
    END { if (!found) print line }
  ' .env >.env.tmp
  cat .env.tmp >.env # keeps the 600 mode of .env
  rm -f .env.tmp
}

random_hex() {
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -hex 24
  else
    od -An -tx1 -N24 /dev/urandom | tr -d ' \n'
  fi
}

command -v curl >/dev/null 2>&1 || fail "curl is required."
command -v docker >/dev/null 2>&1 || fail "Docker is required: https://docs.docker.com/get-docker/"
docker compose version >/dev/null 2>&1 || fail "Docker Compose v2 is required: https://docs.docker.com/compose/install/"
docker info >/dev/null 2>&1 || fail "Docker is installed but not running. Start it and run this again."

if [ -f "$KOI_DIR/.env" ]; then
  say "Koi is already installed in $KOI_DIR. Nothing was changed."
  say "Start it with: cd $KOI_DIR && docker compose up -d"
  exit 0
fi

VERSION="${KOI_VERSION:-}"
if [ -z "$VERSION" ]; then
  VERSION="$(
    curl -fsSL --retry 2 --connect-timeout 10 --max-time 60 "https://api.github.com/repos/$REPO/releases/latest" |
      sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' |
      head -n 1
  )" || true
  [ -n "$VERSION" ] || fail "could not find the latest release. Set KOI_VERSION, for example KOI_VERSION=v0.2.0."
fi

say "Installing Koi $VERSION in $KOI_DIR"

mkdir -p "$KOI_DIR"
cd "$KOI_DIR"

RAW="https://raw.githubusercontent.com/$REPO/$VERSION"
curl -fsSL --retry 2 --connect-timeout 10 --max-time 60 "$RAW/docker-compose.selfhosted.yml" -o docker-compose.yml ||
  fail "could not download docker-compose.yml for $VERSION."
curl -fsSL --retry 2 --connect-timeout 10 --max-time 60 "$RAW/.env.selfhosted.example" -o .env.tmp ||
  fail "could not download .env for $VERSION."
mv .env.tmp .env
chmod 600 .env

set_env KOI_IMAGE_TAG "$VERSION"
set_env POSTGRES_PASSWORD "$(random_hex)"

# Copy settings the caller passed in the environment.
for key in KOI_CLORO_API_KEY KOI_OPENROUTER_API_KEY KOI_RESEND_API_KEY KOI_EMAIL_FROM \
  KOI_PUBLIC_URL KOI_MCP_PUBLIC_URL LOGODEV_SECRET_KEY VITE_LOGODEV_PUBLISHABLE_KEY \
  KOI_ALLOW_SIGNUP KOI_CLIENT_IP_HEADER KOI_HOST_PORT; do
  value="$(printenv "$key" || true)"
  [ -z "$value" ] || set_env "$key" "$value"
done

if ! docker compose up -d --wait; then
  fail "Koi did not start. See the logs with: cd $KOI_DIR && docker compose logs koi"
fi

PORT="$(sed -n 's/^KOI_HOST_PORT=//p' .env)"

say ""
say "Koi is running on http://localhost:${PORT:-8080}"
say "The first account you create owns this instance."
if ! grep -q '^KOI_CLORO_API_KEY=.' .env; then
  say ""
  say "To collect live answers, add your Cloro key to $KOI_DIR/.env:"
  say "  KOI_CLORO_API_KEY=..."
  say "then run: cd $KOI_DIR && docker compose up -d"
fi
if ! grep -q '^KOI_PUBLIC_URL=.' .env; then
  say ""
  say "For remote access, put an HTTPS proxy in front and set KOI_PUBLIC_URL."
fi
say ""
say "Guide: https://github.com/$REPO/blob/main/docs/self-hosting.md"
