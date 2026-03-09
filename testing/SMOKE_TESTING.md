# Smoke Testing

Run the launch smoke suite locally with:

```bash
go test -run TestLaunchSmoke -v ./internal/api
```

This suite is English-named and runs entirely in-process against the in-memory store.
It validates the launch-critical workflow without relying on `endpoints.http` or `testing/manual/index.htm`.

Current coverage:

- `Health endpoint responds and reports ok`
- `Profile lifecycle supports unique handles and metadata add change clear`
- `Organization lifecycle supports unique handles metadata add change clear and delete`
- `Two agent lifecycle supports bind handle finalize metadata list and revoke`

Notes:

- Metadata "delete" is implemented as metadata clear via `PATCH` with `{}`.
- Agent "delete" is currently agent revoke in the API.
