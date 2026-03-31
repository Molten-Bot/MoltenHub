#!/usr/bin/env bash
set -euo pipefail

if ! command -v jq >/dev/null 2>&1; then
  echo "ERROR: jq is required" >&2
  exit 1
fi

NA_BASE_URL="${NA_BASE_URL:-https://na.hive.molten-qa.site/v1}"
EU_BASE_URL="${EU_BASE_URL:-https://eu.hive.molten-qa.site/v1}"

ITERATIONS="${ITERATIONS:-10}"
PULL_TIMEOUT_MS="${PULL_TIMEOUT_MS:-5000}"
HTTP_TIMEOUT_SEC="${HTTP_TIMEOUT_SEC:-20}"
SLO_MS="${SLO_MS:-10000}"
VERBOSE="${VERBOSE:-false}"

NA_SESSION_FILE="${NA_SESSION_FILE:-$HOME/.codex/memories/moltenbot_na_hive_bind_session_codex-synth-a.json}"
EU_SESSION_FILE="${EU_SESSION_FILE:-$HOME/.codex/memories/moltenbot_eu_hive_bind_session_codex-synth-b.json}"

NA_TOKEN="${NA_TOKEN:-}"
EU_TOKEN="${EU_TOKEN:-}"
NA_URI="${NA_URI:-}"
EU_URI="${EU_URI:-}"

if [[ -z "${NA_TOKEN}" && -f "${NA_SESSION_FILE}" ]]; then
  NA_TOKEN="$(jq -r '.bind_response.token // empty' "${NA_SESSION_FILE}")"
fi
if [[ -z "${EU_TOKEN}" && -f "${EU_SESSION_FILE}" ]]; then
  EU_TOKEN="$(jq -r '.bind_response.token // empty' "${EU_SESSION_FILE}")"
fi
if [[ -z "${NA_URI}" && -f "${NA_SESSION_FILE}" ]]; then
  NA_URI="$(jq -r '.bind_response.agent.uri // empty' "${NA_SESSION_FILE}")"
fi
if [[ -z "${EU_URI}" && -f "${EU_SESSION_FILE}" ]]; then
  EU_URI="$(jq -r '.bind_response.agent.uri // empty' "${EU_SESSION_FILE}")"
fi

if [[ -z "${NA_TOKEN}" || -z "${EU_TOKEN}" || -z "${NA_URI}" || -z "${EU_URI}" ]]; then
  cat >&2 <<EOF2
ERROR: missing required inputs.
Provide NA/EU token + URI either via env vars:
  NA_TOKEN EU_TOKEN NA_URI EU_URI
or session files:
  ${NA_SESSION_FILE}
  ${EU_SESSION_FILE}
EOF2
  exit 2
fi

args=(
  -na-base-url "${NA_BASE_URL}"
  -eu-base-url "${EU_BASE_URL}"
  -na-token "${NA_TOKEN}"
  -eu-token "${EU_TOKEN}"
  -na-uri "${NA_URI}"
  -eu-uri "${EU_URI}"
  -iterations "${ITERATIONS}"
  -pull-timeout-ms "${PULL_TIMEOUT_MS}"
  -http-timeout-sec "${HTTP_TIMEOUT_SEC}"
  -slo-ms "${SLO_MS}"
)
if [[ "${VERBOSE}" == "true" ]]; then
  args+=(-verbose)
fi

go run ./cmd/moltenhub-federation-latency "${args[@]}"
