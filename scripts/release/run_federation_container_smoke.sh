#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 || $# -gt 3 ]]; then
  echo "usage: $0 <image-ref> [alpha-host-port] [beta-host-port]" >&2
  exit 1
fi

IMAGE_REF="$1"
ALPHA_PORT="${2:-18080}"
BETA_PORT="${3:-18081}"
ALPHA_BASE_URL="http://127.0.0.1:${ALPHA_PORT}"
BETA_BASE_URL="http://127.0.0.1:${BETA_PORT}"
COMPOSE_FILE="scripts/release/docker-compose.federation-smoke.yml"
PROJECT_NAME="statocyst-federation-smoke"

cleanup() {
  STATOCYST_IMAGE="${IMAGE_REF}" \
  STATOCYST_ALPHA_PORT="${ALPHA_PORT}" \
  STATOCYST_BETA_PORT="${BETA_PORT}" \
  docker compose -p "${PROJECT_NAME}" -f "${COMPOSE_FILE}" down -v >/dev/null 2>&1 || true
}

trap cleanup EXIT
cleanup

STATOCYST_IMAGE="${IMAGE_REF}" \
STATOCYST_ALPHA_PORT="${ALPHA_PORT}" \
STATOCYST_BETA_PORT="${BETA_PORT}" \
docker compose -p "${PROJECT_NAME}" -f "${COMPOSE_FILE}" up -d >/dev/null

wait_for_health() {
  local base_url="$1"
  local label="$2"
  local attempts=0
  until curl -fsS "${base_url}/health" >/dev/null 2>&1; do
    attempts=$((attempts + 1))
    if [[ "${attempts}" -ge 30 ]]; then
      echo "ERROR: ${label} did not become healthy at ${base_url}" >&2
      STATOCYST_IMAGE="${IMAGE_REF}" \
      STATOCYST_ALPHA_PORT="${ALPHA_PORT}" \
      STATOCYST_BETA_PORT="${BETA_PORT}" \
      docker compose -p "${PROJECT_NAME}" -f "${COMPOSE_FILE}" logs >&2 || true
      exit 1
    fi
    sleep 1
  done
}

wait_for_health "${ALPHA_BASE_URL}" "alpha"
wait_for_health "${BETA_BASE_URL}" "beta"

go run ./cmd/statocyst-federation-smoke \
  -alpha-base-url "${ALPHA_BASE_URL}" \
  -beta-base-url "${BETA_BASE_URL}"
