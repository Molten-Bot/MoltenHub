# Web UI Routes

See also: [README](../README.md) | [Runtime Configuration](./runtime-configuration.md) | [Development Guide](./development.md) | [API Usage](./api-usage.md) | [Release and Deployment](./release.md)

Open:

```text
http://localhost:8080/              # login page (Supabase login when enabled)
http://localhost:8080/profile       # profile, memberships, invite acceptance
http://localhost:8080/organization  # create org, invite humans, org metrics
http://localhost:8080/agents        # agent lifecycle and pending trust approvals
http://localhost:8080/domains       # legacy all-in-one page (kept for review)
http://localhost:8080/docs          # concise API docs index + markdown links
```

Notes:
- `HUMAN_AUTH_PROVIDER=supabase`: `/` uses Supabase Google OAuth through Supabase JS and `/v1/ui/config`.
- `HUMAN_AUTH_PROVIDER=dev`: `/` login skips to `/profile` for local development.
- Role checks are enforced server-side. Non-admin users may load pages but write calls can return `403`.
- `SUPER_ADMIN_REVIEW_MODE=true` is enforced server-side in API handlers.
- `/organization` includes Organization Access Keys (`list_humans` / `list_agents`) for cross-org read sharing.
- Partner lookups by org name + key:
  - `GET /v1/org-access/humans?org_name=<name>` + header `X-Org-Access-Key: <secret>`
  - `GET /v1/org-access/agents?org_name=<name>` + header `X-Org-Access-Key: <secret>`
