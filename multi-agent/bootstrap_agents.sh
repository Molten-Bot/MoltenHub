#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-$SCRIPT_DIR/docker-compose.yml}"
BASE_URL="${STATOCYST_BASE_URL:-http://statocyst:8080}"

ALICE_ID="${ALICE_ID:-crab-admin}"
ALICE_EMAIL="${ALICE_EMAIL:-crab@local.dev}"
BOB_ID="${BOB_ID:-shrimp-admin}"
BOB_EMAIL="${BOB_EMAIL:-shrimp@local.dev}"
CRAB_AGENT_ID="${CRAB_AGENT_ID:-crab}"
SHRIMP_AGENT_ID="${SHRIMP_AGENT_ID:-shrimp}"

wait_for_http() {
  for _ in $(seq 1 40); do
    if docker exec multi-agent-crab-1 bash -lc "curl -sS $BASE_URL/health >/dev/null"; then
      return 0
    fi
    sleep 1
  done
  echo "ERROR: statocyst health endpoint not ready" >&2
  exit 1
}

echo "[bootstrap] starting stack"
docker compose -f "$COMPOSE_FILE" up -d statocyst shrimp crab

echo "[bootstrap] recreating statocyst for clean in-memory state"
docker compose -f "$COMPOSE_FILE" up -d --force-recreate --no-deps statocyst
wait_for_http

echo "[bootstrap] creating orgs"
ORG_A="$(docker exec multi-agent-crab-1 bash -lc "curl -sS -X POST $BASE_URL/v1/orgs -H 'Content-Type: application/json' -H 'X-Human-Id: $ALICE_ID' -H 'X-Human-Email: $ALICE_EMAIL' -d '{\"name\":\"Crab Org\"}'" | python3 -c 'import json,sys; print(json.load(sys.stdin)["organization"]["org_id"])')"
ORG_B="$(docker exec multi-agent-crab-1 bash -lc "curl -sS -X POST $BASE_URL/v1/orgs -H 'Content-Type: application/json' -H 'X-Human-Id: $BOB_ID' -H 'X-Human-Email: $BOB_EMAIL' -d '{\"name\":\"Shrimp Org\"}'" | python3 -c 'import json,sys; print(json.load(sys.stdin)["organization"]["org_id"])')"

ALICE_HID="$(docker exec multi-agent-crab-1 bash -lc "curl -sS $BASE_URL/v1/me -H 'X-Human-Id: $ALICE_ID' -H 'X-Human-Email: $ALICE_EMAIL'" | python3 -c 'import json,sys; print(json.load(sys.stdin)["human"]["human_id"])')"
BOB_HID="$(docker exec multi-agent-crab-1 bash -lc "curl -sS $BASE_URL/v1/me -H 'X-Human-Id: $BOB_ID' -H 'X-Human-Email: $BOB_EMAIL'" | python3 -c 'import json,sys; print(json.load(sys.stdin)["human"]["human_id"])')"

echo "[bootstrap] registering agents"
CRAB_TOKEN="$(docker exec multi-agent-crab-1 bash -lc "curl -sS -X POST $BASE_URL/v1/agents/register -H 'Content-Type: application/json' -H 'X-Human-Id: $ALICE_ID' -H 'X-Human-Email: $ALICE_EMAIL' -d '{\"org_id\":\"$ORG_A\",\"agent_id\":\"$CRAB_AGENT_ID\",\"owner_human_id\":\"$ALICE_HID\"}'" | python3 -c 'import json,sys; print(json.load(sys.stdin)["token"])')"
SHRIMP_TOKEN="$(docker exec multi-agent-crab-1 bash -lc "curl -sS -X POST $BASE_URL/v1/agents/register -H 'Content-Type: application/json' -H 'X-Human-Id: $BOB_ID' -H 'X-Human-Email: $BOB_EMAIL' -d '{\"org_id\":\"$ORG_B\",\"agent_id\":\"$SHRIMP_AGENT_ID\",\"owner_human_id\":\"$BOB_HID\"}'" | python3 -c 'import json,sys; print(json.load(sys.stdin)["token"])')"

docker exec multi-agent-crab-1 bash -lc "umask 077; printf '%s\n' '$CRAB_TOKEN' > /tmp/${CRAB_AGENT_ID}.token"
docker exec multi-agent-shrimp-1 bash -lc "umask 077; printf '%s\n' '$SHRIMP_TOKEN' > /tmp/${SHRIMP_AGENT_ID}.token"

echo "[bootstrap] activating org trust"
ORG_TRUST_ID="$(docker exec multi-agent-crab-1 bash -lc "curl -sS -X POST $BASE_URL/v1/org-trusts -H 'Content-Type: application/json' -H 'X-Human-Id: $ALICE_ID' -H 'X-Human-Email: $ALICE_EMAIL' -d '{\"org_id\":\"$ORG_A\",\"peer_org_id\":\"$ORG_B\"}'" | python3 -c 'import json,sys; print(json.load(sys.stdin)["trust"]["edge_id"])')"
docker exec multi-agent-crab-1 bash -lc "curl -sS -X POST $BASE_URL/v1/org-trusts/$ORG_TRUST_ID/approve -H 'X-Human-Id: $BOB_ID' -H 'X-Human-Email: $BOB_EMAIL' >/tmp/org_trust_approve.json"

echo "[bootstrap] activating agent trust"
AGENT_TRUST_ID="$(docker exec multi-agent-crab-1 bash -lc "curl -sS -X POST $BASE_URL/v1/agent-trusts -H 'Content-Type: application/json' -H 'X-Human-Id: $ALICE_ID' -H 'X-Human-Email: $ALICE_EMAIL' -d '{\"org_id\":\"$ORG_A\",\"agent_id\":\"$CRAB_AGENT_ID\",\"peer_agent_id\":\"$SHRIMP_AGENT_ID\"}'" | python3 -c 'import json,sys; print(json.load(sys.stdin)["trust"]["edge_id"])')"
docker exec multi-agent-crab-1 bash -lc "curl -sS -X POST $BASE_URL/v1/agent-trusts/$AGENT_TRUST_ID/approve -H 'X-Human-Id: $BOB_ID' -H 'X-Human-Email: $BOB_EMAIL' >/tmp/agent_trust_approve.json"

echo "[bootstrap] running exchange smoke test"
EXCHANGE_JSON="$(docker exec multi-agent-crab-1 bash -lc "/mnt/skills/openclaw-exchange-messages/scripts/exchange_roundtrip.sh '$BASE_URL' '$CRAB_AGENT_ID' '$CRAB_TOKEN' '$SHRIMP_AGENT_ID' '$SHRIMP_TOKEN' 'bootstrap-ping' 'bootstrap-pong' 5000")"

python3 - <<PY
import json
print(json.dumps({
  "status": "ok",
  "org_a": "$ORG_A",
  "org_b": "$ORG_B",
  "org_trust_id": "$ORG_TRUST_ID",
  "agent_trust_id": "$AGENT_TRUST_ID",
  "crab_token_file": "/tmp/$CRAB_AGENT_ID.token",
  "shrimp_token_file": "/tmp/$SHRIMP_AGENT_ID.token",
  "exchange": json.loads('''$EXCHANGE_JSON'''),
}, indent=2))
PY
