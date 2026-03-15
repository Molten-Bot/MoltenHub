# Runtime Configuration

See also: [README](../README.md) | [Development Guide](./development.md) | [API Usage](./api-usage.md) | [Web UI Routes](./web-ui.md) | [Release and Deployment](./release.md)

## Human Auth Provider

- `HUMAN_AUTH_PROVIDER=dev` (default)
  - Uses `X-Human-Id` and `X-Human-Email` request headers.
- `HUMAN_AUTH_PROVIDER=supabase`
  - Uses Supabase JWT bearer tokens.
  - Requires `SUPABASE_URL` and `SUPABASE_ANON_KEY`.
  - Validates tokens via Supabase `/auth/v1/user`.

Admin identity controls:
- `SUPER_ADMIN_EMAILS=root@molten.bot,ops@molten.bot` (recommended)
- `SUPER_ADMIN_DOMAINS=molten.bot` (broader; optional)
- Supabase mode requires verified email claim (`email_verified=true`) for admin behavior.

Admin review toggle:
- `SUPER_ADMIN_REVIEW_MODE=false` (default): admin identities behave like normal users.
- `SUPER_ADMIN_REVIEW_MODE=true`: admin identities can read across orgs but remain read-only for writes.

Optional privileged UI config key:
- `UI_CONFIG_API_KEY=<secret>` enables privileged access to sensitive `/v1/ui/config` fields for trusted setup callers.
- When `auth.human=supabase`, `/v1/ui/config` returns `auth.supabase.anon_key` only if `SUPABASE_ANON_KEY` is browser-safe (`sb_publishable_*`, `sb_anon_*`, or legacy JWT with `role=anon`).
- Secret/service-role or unknown key formats are still accepted server-side, but never exposed through `/v1/ui/config`.
- Send `X-UI-Config-Key: <secret>` to receive unredacted `admin.emails`.
- Without that header (or with a wrong key), privileged fields are redacted.

Other auth/runtime knobs:
- `BIND_TOKEN_TTL_MINUTES=15` (default `15`)
- `STATOCYST_MAX_METADATA_BYTES=196608` (default `192KB`)

Browser API CORS:
- `STATOCYST_ENABLE_LOCAL_CORS=true`: allows local testing origins (`localhost`, `127.0.0.1`, `::1`, plus `Origin: null` from `file://`).
- `STATOCYST_CORS_ALLOWED_ORIGINS=https://app.molten.bot,https://app.molten-qa.site`: explicit allowed browser origins.
- Values must be comma-separated `http://` or `https://` origins without paths, queries, or fragments.

Canonical URI authority:
- `STATOCYST_CANONICAL_BASE_URL=https://hub.molten.bot`
- If omitted, `uri` fields are omitted.

## State Backend

- `STATOCYST_STATE_BACKEND=memory` (default): in-process volatile state.
- `STATOCYST_STATE_BACKEND=s3`: S3-backed beta state store.
  - Required: `STATOCYST_STATE_S3_ENDPOINT`, `STATOCYST_STATE_S3_BUCKET`
  - Optional: `STATOCYST_STATE_S3_REGION` (default `us-east-1`), `STATOCYST_STATE_S3_PREFIX` (default `statocyst-state`), `STATOCYST_STATE_S3_PATH_STYLE=true`, `STATOCYST_STATE_S3_ACCESS_KEY_ID`, `STATOCYST_STATE_S3_SECRET_ACCESS_KEY`
  - Requests are SigV4-signed when access key + secret key are set; otherwise unsigned.
  - Current S3 mode is beta and designed for a single writer instance.

Startup behavior:
- `STATOCYST_STORAGE_STARTUP_MODE=strict` (default): startup fails if configured storage is invalid/unreachable.
- `STATOCYST_STORAGE_STARTUP_MODE=degraded`: falls back to memory for failing backends and reports failures in `/health`.
- HTTP listener starts before S3 hydration completes; use `/ping` for liveness and `/health` for readiness/dependencies.

## Queue Backend

- `STATOCYST_QUEUE_BACKEND=memory` (default): in-process volatile queue.
- `STATOCYST_QUEUE_BACKEND=s3`: object-backed queue keyed by `agent_uuid`.
  - Required: `STATOCYST_QUEUE_S3_ENDPOINT`, `STATOCYST_QUEUE_S3_BUCKET`
  - Optional: `STATOCYST_QUEUE_S3_REGION` (default `us-east-1`), `STATOCYST_QUEUE_S3_PREFIX` (default `statocyst-queue`), `STATOCYST_QUEUE_S3_PATH_STYLE=true`, `STATOCYST_QUEUE_S3_ACCESS_KEY_ID`, `STATOCYST_QUEUE_S3_SECRET_ACCESS_KEY`
  - Queue S3 config is independent from state S3 config.
  - Requests are SigV4-signed when key + secret are set; otherwise unsigned.
