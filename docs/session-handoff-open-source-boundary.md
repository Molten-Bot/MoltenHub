# Session Handoff: Open-Source Boundary and Synthetic Checks

Date: 2026-04-03 (UTC)

## Goal

Keep this open-source repo (`moltenhub`) self-contained with no live-environment dependencies.

What should remain in this repo:
- Go unit/integration tests (`go test ./...`)
- Local container artifact smoke tests (local Docker/compose targets)

What should move to infra/deployment repo:
- Live QA/Prod synthetic checks using real agent tokens and live domains
- Environment secrets/variables for synthetic monitoring
- Post-deploy gating/alerting logic tied to deployment platform

## Current State (from this session)

### Added/modified in `moltenhub`

1. Expanded local smoke coverage for OpenClaw behavior in container smoke runner:
- `cmd/moltenhub-smoke/main.go`
  - Added steps for:
    - plugin registration
    - HTTP publish/pull/ack
    - websocket delivery/ack

2. Added live transport synthetic probe command:
- `cmd/moltenhub-openclaw-latency/main.go`
- `cmd/moltenhub-openclaw-latency/main_test.go`

3. Added live synthetic wrapper script:
- `scripts/release/run_openclaw_transport_synthetics.sh`

4. Added optional post-deploy live synthetic jobs in workflows:
- `.github/workflows/deploy-vnext.yml`
- `.github/workflows/deploy-prod.yml`

5. Updated docs with live synthetic usage:
- `docs/release.md`
- `docs/development.md`

6. Generated one live latency report document:
- `docs/openclaw-transport-latency.md`

## Recommended Boundary After Refactor

### Keep in `moltenhub` (OSS)

- `go test ./...`
- `scripts/release/run_container_smoke.sh`
- `scripts/release/run_federation_container_smoke.sh`
- `cmd/moltenhub-smoke`
- `cmd/moltenhub-federation-smoke`
- CI checks that validate code + built image locally

### Move out of `moltenhub` into infra/deployment repo

- `cmd/moltenhub-openclaw-latency` (or equivalent synthetic runner)
- `scripts/release/run_openclaw_transport_synthetics.sh` (or infra-native wrapper)
- Post-deploy `live-agent-synthetics` jobs currently in:
  - `.github/workflows/deploy-vnext.yml`
  - `.github/workflows/deploy-prod.yml`
- Live report generation/storage for deployed environments

## Next Session Plan (Two Repos)

### Repo A: `moltenhub` (this repo)

1. Remove live-dependency additions:
- delete `cmd/moltenhub-openclaw-latency/`
- delete `scripts/release/run_openclaw_transport_synthetics.sh`
- remove `live-agent-synthetics` jobs from:
  - `.github/workflows/deploy-vnext.yml`
  - `.github/workflows/deploy-prod.yml`
- remove live-synthetic sections from:
  - `docs/release.md`
  - `docs/development.md`
- decide whether to keep or remove `docs/openclaw-transport-latency.md` (likely remove from OSS repo)

2. Keep local smoke OpenClaw steps in `cmd/moltenhub-smoke/main.go` (no live dependency).

3. Verify:
- `go test ./...`
- container smoke workflow still green

### Repo B: infra/deployment repo

1. Add/port the live synthetic probe runner (or use curl/ws toolchain).
2. Add post-deploy QA and Prod synthetic gates.
3. Configure secrets and env vars in infra CI/CD:
- agent A token
- agent B token
- base URL per environment
- optional p95 threshold
- iterations/retries/timeouts
4. Publish synthetic outputs as CI artifacts and wire alerting.

## Known Runtime Observation (Useful for Infra)

During this session, QA briefly returned startup responses (`503` with `status: starting`) before becoming ready.
Recommendation for infra synthetics:
- include readiness wait/backoff before latency checks
- fail only after bounded retries, not on first startup response

## Validation Performed in This Session

- `go test ./...` passed after changes.
- Live synthetic matrix run succeeded against QA with 4 samples/scenario and report generated in:
  - `docs/openclaw-transport-latency.md`

## Working Tree Snapshot at Handoff

Modified:
- `.github/workflows/deploy-prod.yml`
- `.github/workflows/deploy-vnext.yml`
- `cmd/moltenhub-smoke/main.go`
- `docs/development.md`
- `docs/release.md`

Added:
- `cmd/moltenhub-openclaw-latency/main.go`
- `cmd/moltenhub-openclaw-latency/main_test.go`
- `scripts/release/run_openclaw_transport_synthetics.sh`
- `docs/openclaw-transport-latency.md`
- `docs/session-handoff-open-source-boundary.md`
