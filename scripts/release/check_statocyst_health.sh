#!/usr/bin/env bash
set -euo pipefail

url="${1:?healthcheck url is required}"
attempt_limit="${2:-30}"
sleep_seconds="${3:-2}"

liveness_url=""
health_url="$url"

case "$url" in
  */ping)
    liveness_url="$url"
    health_url="${url%/ping}/health"
    ;;
  */health)
    ;;
esac

tmp_body="$(mktemp)"
trap 'rm -f "$tmp_body"' EXIT

health_ready() {
  local body_path="$1"
  python3 - "$body_path" <<'PY'
import json
import sys

path = sys.argv[1]
with open(path, "r", encoding="utf-8") as handle:
    payload = json.load(handle)

status = str(payload.get("status", "")).strip().lower()
boot_status = str(payload.get("boot_status", "")).strip().lower()

if boot_status == "starting":
    print(f"boot_status={boot_status} status={status}")
    sys.exit(2)

if status != "ok":
    print(f"status={status}")
    sys.exit(3)

print(f"status={status}")
PY
}

attempt=0
while [ "$attempt" -lt "$attempt_limit" ]; do
  attempt=$((attempt + 1))

  if [ -n "$liveness_url" ]; then
    code="$(curl --connect-timeout 3 --max-time 10 -sS -o "$tmp_body" -w "%{http_code}" "$liveness_url" || true)"
    if [ "$code" != "200" ] && [ "$code" != "204" ]; then
      echo "liveness attempt $attempt/$attempt_limit for $liveness_url -> HTTP $code"
      sleep "$sleep_seconds"
      continue
    fi
  fi

  code="$(curl --connect-timeout 3 --max-time 10 -sS -o "$tmp_body" -w "%{http_code}" "$health_url" || true)"
  if [ "$code" = "200" ]; then
    if readiness="$(health_ready "$tmp_body" 2>&1)"; then
      echo "health check passed for $health_url on attempt $attempt/$attempt_limit ($readiness)"
      exit 0
    fi
    echo "health attempt $attempt/$attempt_limit for $health_url not ready: $readiness"
  else
    echo "health attempt $attempt/$attempt_limit for $health_url -> HTTP $code"
  fi

  if [ "$attempt" -eq 1 ] || [ "$attempt" -eq "$attempt_limit" ]; then
    if [ -s "$tmp_body" ]; then
      echo "response body:"
      head -c 512 "$tmp_body"
      printf "\n"
    fi
  fi

  sleep "$sleep_seconds"
done

echo "ERROR: health check failed for $health_url after $attempt_limit attempts" >&2
exit 1
