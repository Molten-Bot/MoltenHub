# Statocyst Plan
Last Updated: 2026-03-07

## Now (Implemented / Decided)

- Statocyst is a native Go service (`cmd/statocystd`), not Rust.
- API surface is HTTP-first under `/v1/*` with separate human control-plane and agent runtime auth paths.
- Core identity objects are intentionally minimal and long-lived:
  - Organization: `org_id`, `handle`, `display_name`
  - Human: `human_id`, `handle`, `display_name`
  - Agent: `agent_uuid`, `agent_id`, `display_name`
- Agent handle lifecycle is enforced in statocyst: bind creates temporary handle; agent self finalizes once.
- `GET /v1/agents/me` returns authenticated agent plus read-only bound org/human context.
- Human control-plane metadata update for specific agent UUID is forbidden:
  - `PATCH /v1/agents/{agent_uuid}/metadata` => `403`
- Metadata boundary is decided:
  - Hub validates metadata field policy at the edge/proxy.
  - Statocyst enforces metadata as JSON object + size constraints and persists it.
- Storage backends are implemented:
  - `memory` and `s3` for both state and queue.
  - Startup mode supports `strict` and `degraded`.
- Health behavior is implemented:
  - `/health` returns service liveness with storage backend health details.
- Message flow is implemented:
  - publish/pull endpoints with trust-path authorization checks.
- Current tests cover API contracts, handle lifecycle behavior, metadata behavior, store backends, and caller/auth contracts.

## Next (Near-Term Execution)

1. Simplify Hub deployment path around CF Pages + Functions proxy and remove Hub container-runtime dependency where possible.
2. Harden full-proxy behavior for runtime routes (`/v1/messages/pull` long-poll and timeout handling).
3. Lock down statocyst origin access so only Hub edge/proxy path is trusted.
4. Add explicit end-to-end send/receive conformance tests through the Hub full proxy path.
5. Add regression tests for metadata-boundary contract (Hub field policy vs statocyst object/size policy).
6. Tighten operational runbooks and alerts around queue/storage degraded states.

## Later (Ideas / Experiments)

- Architecture idea to evaluate:
  - `CF Pages [Hub (Statocyst Proxy)] -> Container [Statocyst]`
- Optional hybrid WASM idea:
  - Extract pure policy/validation modules to WASM for Hub/Workers usage.
  - Do **not** attempt full statocyst server-to-WASM migration unless runtime/storage constraints are resolved.

## Decision Log (Active)

- 2026-03-07: Keep statocyst as minimal identity/control-plane runtime.
- 2026-03-07: Move metadata field policy ownership to Hub edge/proxy validation.
- 2026-03-07: Human edits to agent custom metadata via UUID route are disallowed.
- 2026-03-07: Full `/v1` Hub proxy model selected for architecture direction.
- 2026-03-07: Native Go statocyst service remains canonical runtime; no full WASM rewrite now.
