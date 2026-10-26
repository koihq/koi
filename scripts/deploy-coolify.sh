#!/usr/bin/env bash
set -euo pipefail

: "${COOLIFY_URL:?Set COOLIFY_URL}"
: "${COOLIFY_APP_UUID:?Set COOLIFY_APP_UUID}"
: "${COOLIFY_API_TOKEN:?Set COOLIFY_API_TOKEN}"
: "${KOI_IMAGE_TAG:?Set KOI_IMAGE_TAG}"
: "${KOI_SMOKE_URL:?Set KOI_SMOKE_URL}"
: "${CF_ACCESS_CLIENT_ID:?Set CF_ACCESS_CLIENT_ID}"
: "${CF_ACCESS_CLIENT_SECRET:?Set CF_ACCESS_CLIENT_SECRET}"

if [[ ! "$KOI_IMAGE_TAG" =~ ^(sha-[0-9a-f]{40}|v[0-9]+\.[0-9]+\.[0-9]+)$ ]]; then
  echo "Invalid image tag: $KOI_IMAGE_TAG" >&2
  exit 1
fi

api="${COOLIFY_URL%/}/api/v1"
auth=(
  -H "Authorization: Bearer $COOLIFY_API_TOKEN"
  -H "CF-Access-Client-Id: $CF_ACCESS_CLIENT_ID"
  -H "CF-Access-Client-Secret: $CF_ACCESS_CLIENT_SECRET"
)

payload="$(jq -n --arg tag "$KOI_IMAGE_TAG" '{docker_registry_image_tag: $tag}')"
curl --fail-with-body --silent --show-error \
  "${auth[@]}" -H 'Content-Type: application/json' \
  -X PATCH --data "$payload" \
  "$api/applications/$COOLIFY_APP_UUID" >/dev/null

response="$(curl --fail-with-body --silent --show-error \
  "${auth[@]}" -X POST "$api/deploy?uuid=$COOLIFY_APP_UUID")"
deployment_uuid="$(jq -er '.deployments[0].deployment_uuid' <<<"$response")"
echo "Coolify deployment: $deployment_uuid"

for attempt in {1..120}; do
  response="$(curl --fail-with-body --silent --show-error \
    "${auth[@]}" "$api/deployments/$deployment_uuid")"
  status="$(jq -er '.status' <<<"$response")"

  case "$status" in
    finished)
      break
      ;;
    failed|cancelled|cancelled-by-user|error)
      echo "Coolify deployment ended with status: $status" >&2
      exit 1
      ;;
    queued|in_progress)
      sleep 5
      ;;
    *)
      echo "Unexpected Coolify deployment status: $status" >&2
      exit 1
      ;;
  esac
done

if [[ "$status" != finished ]]; then
  echo "Coolify deployment timed out" >&2
  exit 1
fi

for attempt in {1..12}; do
  if curl --fail --silent --show-error \
    -H "CF-Access-Client-Id: $CF_ACCESS_CLIENT_ID" \
    -H "CF-Access-Client-Secret: $CF_ACCESS_CLIENT_SECRET" \
    "${KOI_SMOKE_URL%/}/health" | jq -e '.status == "ok" and .db == "ok"' >/dev/null; then
    echo "Koi health check passed: $KOI_IMAGE_TAG"
    exit 0
  fi
  sleep 5
done

echo "Koi health check did not pass" >&2
exit 1
