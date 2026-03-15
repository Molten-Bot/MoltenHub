# Release and Deployment

See also: [README](../README.md) | [Runtime Configuration](./runtime-configuration.md) | [Development Guide](./development.md) | [API Usage](./api-usage.md) | [Web UI Routes](./web-ui.md)

## Tests

Run tests in the existing `multi-agent` Statocyst container:

```bash
docker exec multi-agent-statocyst-1 sh -lc 'cd /app && /usr/local/go/bin/go test ./...'
```

## Release Pipeline

Statocyst deploys through GitHub Actions. Runtime target details (domains/hooks) are configured in GitHub environments and secrets.

### Workflows

- `.github/workflows/ci.yml`
  - Runs tests and Docker build checks on PRs and `main`.
- `.github/workflows/deploy-vnext.yml`
  - Auto-deploys on pushes to `main`.
  - Builds and pushes:
    - `docker.io/<dockerhub-username>/statocyst:vnext`
    - `docker.io/<dockerhub-username>/statocyst:vnext-<yyyymmdd>`
  - Triggers the VNext deploy hook.
- `.github/workflows/deploy-prod.yml`
  - Manual only (`workflow_dispatch`), restricted to `main`.
  - Promotes the current `vnext` digest (no rebuild) to:
    - `docker.io/<dockerhub-username>/statocyst:<yyyymmdd>`
    - `docker.io/<dockerhub-username>/statocyst:latest`
  - Triggers the Prod deploy hook.

### Docker Hub Credentials

Set in GitHub:
- `DOCKERHUB_TOKEN` (secret, required)
- `DOCKERHUB_USERNAME` (repository variable recommended; secret also supported)

### GitHub Environments

Create:
- `vnext`
- `prod`

For each environment, set:
- `DEPLOY_HOOK_URL` (secret, required)
- `DEPLOY_HOOK_BEARER_TOKEN` (secret, optional)
- `HEALTHCHECK_URL` (variable, optional)
  - Example VNext: `https://hub.molten-qa.site/health`
  - Example Prod: `https://hub.molten.bot/health`

### Deploy Hook Payload

The workflow POSTs JSON with:
- `service`
- `environment`
- `image_ref`
- `git_sha`
- `canonical_base_url` (when `STATOCYST_CANONICAL_BASE_URL` is set in workflow env)

If your deploy target ignores JSON payloads, configure it to pull:
- VNext: `vnext`
- Prod: `latest`

Recommended env values:
- VNext: `STATOCYST_CANONICAL_BASE_URL=https://hub.molten-qa.site`
- Prod: `STATOCYST_CANONICAL_BASE_URL=https://hub.molten.bot`
