# Statocyst V2 Plan

## Summary

Statocyst V2 should become an agent-first control plane and messaging API. The core goal is to make agent bootstrap, discovery, and API calling succeed with minimal prompt glue and minimal out-of-band documentation.

V2 should stay JSON-first for runtime correctness, add markdown as a first-class negotiated discovery format, and make the authenticated agent surface self-describing.

## Vision

An agent that receives a valid Statocyst bearer token should be able to:

- discover what it can do from the API itself
- understand auth, route, and payload rules without external docs
- call runtime routes successfully with stable machine-readable contracts
- fetch markdown guidance when operating in LLM-oriented environments

## Core Principles

- JSON is the source of truth for runtime APIs and all mutating requests.
- Markdown is a negotiated read format for discovery, help, and agent guidance.
- Discovery must be built into the authenticated agent surface.
- Error handling must be machine-usable, stable, and consistent across agent routes.
- Public liveness and readiness routes must remain minimal and stable.
- Human and agent credential classes must remain strictly separated.

## V2 API Direction

### Canonical discovery

Add `GET /v1/agents/me/manifest` as the canonical agent discovery route.

It should return JSON by default and markdown when callers send `Accept: text/markdown` or `?format=markdown`.

The manifest should include:

- capability ids
- allowed routes
- methods
- auth requirements
- request and response content types
- endpoint map
- schema links or compact schema summaries
- concise success examples
- common failure examples
- retry guidance

### Capabilities

Expand `GET /v1/agents/me/capabilities` into a richer machine-readable contract rather than a short list.

It should expose:

- capability metadata
- route mapping
- operational constraints
- communication affordances
- whether each action is read-only, mutating, retryable, or gated by trust state

### Skill document

Keep `GET /v1/agents/me/skill`, but treat it as a compatibility and guidance route instead of the primary source of truth.

The skill document should be rendered from the same typed manifest data used for JSON discovery responses.

### Runtime contract normalization

Refactor the existing agent runtime surface to make calling safer for generic agents:

- normalize success and error envelope shapes
- include stable `error.code` values
- include `retryable` hints where meaningful
- include `next_action` guidance where meaningful
- include request correlation ids in headers and error bodies
- reject unsupported request or response media types with precise `415` and `406` responses

## Markdown Strategy

Markdown support should exist for discovery and guidance surfaces only:

- `/v1/agents/me/manifest`
- `/v1/agents/me/skill`
- an optional OpenAPI companion summary route such as `/openapi.md`

Mutating routes should remain JSON request-body based:

- `/v1/agents/bind`
- `/v1/messages/publish`
- profile and metadata updates

If Statocyst is deployed behind Cloudflare, docs and UI pages should also be compatible with Cloudflare Markdown for Agents. That should improve doc consumption, but it should not replace the JSON agent contract.

## Agent Success Improvements

The authenticated agent surface should make these failure modes easier to avoid:

- using the wrong credential class
- sending malformed JSON
- missing required fields
- misunderstanding endpoint purpose
- retrying non-retryable operations
- publishing or pulling without required trust relationships

Every agent-facing route should provide errors that are useful to both code and LLM-driven clients.

## Reliability Requirements

V2 should preserve and protect:

- `/ping` as a lightweight liveness route
- `/health` as the dependency and readiness route
- strict separation between human control-plane routes and agent runtime routes
- deterministic content negotiation on discovery routes

Markdown or doc rendering changes must never interfere with `/ping`, `/health`, or authenticated API behavior.

## Initial Milestones

### Milestone 1: Discovery foundation

- add `/v1/agents/me/manifest`
- derive markdown and JSON discovery output from shared typed data
- expand `/v1/agents/me/capabilities`
- document the canonical agent route contract in OpenAPI

### Milestone 2: Contract normalization

- standardize success and error envelopes across agent routes
- add stable error codes and retryability hints
- improve validation and auth failure precision
- add correlation ids consistently

### Milestone 3: Markdown and docs polish

- improve markdown output for manifest and skill routes
- add an OpenAPI markdown companion or summary route
- make HTML docs pages agent-readable and Cloudflare markdown friendly

## Non-Goals

- markdown request bodies for mutating runtime routes
- blending human and agent auth models
- moving liveness or readiness behavior into documentation logic
- relying on external docs as the only source of agent discovery

## Acceptance Criteria

V2 is successful when a newly bound agent can:

- fetch a canonical manifest in JSON
- fetch equivalent guidance in markdown
- determine which routes it may call and how
- publish and pull messages without out-of-band docs
- receive structured failures that identify what went wrong and what to do next

## Test Plan

Add or extend coverage for:

- manifest JSON responses
- manifest markdown responses
- skill markdown regression behavior
- capabilities schema stability
- normalized agent-route error envelopes
- auth-class mismatch failures
- validation failures
- publish and pull success paths
- content negotiation failures
- `/ping` and `/health` regressions

## Defaults

- V2 may refactor the existing agent runtime routes directly rather than only adding parallel alternatives.
- JSON remains the canonical runtime protocol.
- Markdown is additive for discovery and help surfaces.
- Cloudflare Markdown for Agents is an optional distribution improvement, not the primary API contract.
