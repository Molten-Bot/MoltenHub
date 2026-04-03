#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:?BASE_URL is required}"
AGENT_A_TOKEN="${AGENT_A_TOKEN:?AGENT_A_TOKEN is required}"
AGENT_B_TOKEN="${AGENT_B_TOKEN:?AGENT_B_TOKEN is required}"

ITERATIONS="${ITERATIONS:-6}"
SAMPLE_RETRIES="${SAMPLE_RETRIES:-1}"
PULL_TIMEOUT_MS="${PULL_TIMEOUT_MS:-5000}"
HTTP_TIMEOUT_SEC="${HTTP_TIMEOUT_SEC:-45}"
WS_TIMEOUT_SEC="${WS_TIMEOUT_SEC:-10}"
MAX_P95_MS="${MAX_P95_MS:-0}"
VERBOSE="${VERBOSE:-false}"
REPORT_PATH="${REPORT_PATH:-}"

args=(
  -base-url "${BASE_URL}"
  -agent-a-token "${AGENT_A_TOKEN}"
  -agent-b-token "${AGENT_B_TOKEN}"
  -iterations "${ITERATIONS}"
  -sample-retries "${SAMPLE_RETRIES}"
  -pull-timeout-ms "${PULL_TIMEOUT_MS}"
  -http-timeout-sec "${HTTP_TIMEOUT_SEC}"
  -ws-timeout-sec "${WS_TIMEOUT_SEC}"
  -max-p95-ms "${MAX_P95_MS}"
)

if [[ "${VERBOSE}" == "true" ]]; then
  args+=( -verbose )
fi
if [[ -n "${REPORT_PATH}" ]]; then
  args+=( -report-path "${REPORT_PATH}" )
fi

go run ./cmd/moltenhub-openclaw-latency "${args[@]}"
