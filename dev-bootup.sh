#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

if [[ -n "${MOLTENHUB_ADDR:-}" ]]; then
  if [[ "$MOLTENHUB_ADDR" =~ :([0-9]+)$ ]]; then
    PORT="${BASH_REMATCH[1]}"
  else
    PORT="${MOLTENHUB_PORT:-8080}"
  fi
else
  PORT="${MOLTENHUB_PORT:-8080}"
  export MOLTENHUB_ADDR=":${PORT}"
fi

export HUMAN_AUTH_PROVIDER="${HUMAN_AUTH_PROVIDER:-dev}"
export MOLTENHUB_UI_DEV_MODE="${MOLTENHUB_UI_DEV_MODE:-true}"
export MOLTENHUB_ENABLE_LOCAL_CORS="${MOLTENHUB_ENABLE_LOCAL_CORS:-true}"
export GOCACHE="${GOCACHE:-/tmp/moltenhub-gocache}"

mkdir -p "${GOCACHE}"

list_port_pids() {
  lsof -tiTCP:"${PORT}" -sTCP:LISTEN 2>/dev/null || true
}

stop_existing_moltenhub_on_port() {
  local pid cmd stopped=0
  while IFS= read -r pid; do
    [[ -z "$pid" ]] && continue
    cmd="$(ps -p "$pid" -o command= 2>/dev/null || true)"
    if [[ "$cmd" == *moltenhub* || "$cmd" == *cmd/moltenhubd* ]]; then
      echo "Stopping existing moltenhub process on :${PORT} (pid ${pid})"
      kill "$pid" 2>/dev/null || true
      stopped=1
    fi
  done < <(list_port_pids)

  if [[ "$stopped" -eq 1 ]]; then
    local _i
    for _i in $(seq 1 25); do
      [[ -z "$(list_port_pids)" ]] && break
      sleep 0.2
    done
  fi
}

ensure_port_free() {
  local pids
  pids="$(list_port_pids | tr '\n' ' ' | sed 's/[[:space:]]*$//')"
  if [[ -n "$pids" ]]; then
    echo "Port :${PORT} is in use by pid(s): ${pids}."
    echo "Stop the other process or set MOLTENHUB_PORT (or MOLTENHUB_ADDR) to another port."
    exit 1
  fi
}

stop_existing_moltenhub_on_port
ensure_port_free

echo "Starting moltenhub (native) at http://localhost:${PORT}"
echo "MOLTENHUB_ADDR=${MOLTENHUB_ADDR} HUMAN_AUTH_PROVIDER=${HUMAN_AUTH_PROVIDER} MOLTENHUB_UI_DEV_MODE=${MOLTENHUB_UI_DEV_MODE} MOLTENHUB_ENABLE_LOCAL_CORS=${MOLTENHUB_ENABLE_LOCAL_CORS}"
echo "GOCACHE=${GOCACHE}"
echo "Press Ctrl+C to stop."

exec go run ./cmd/moltenhubd
