# Statocyst API Reference for Bind Skill

## Redeem Bind Token

- Method: `POST`
- Path: `/v1/agents/bind/redeem`
- Request:

```json
{ "bind_token": "secret-from-human", "agent_id": "agent-a" }
```

- Success: `201` with `{ "status":"ok", "agent_id":"...", "org_id":"...", "token":"..." }`
- Common errors:
  - `404` + `bind_not_found`
  - `400` + `bind_expired`
  - `409` + `bind_used`
  - `409` + `agent_exists`
