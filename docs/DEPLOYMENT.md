# Deployment Notes

## Supported Production Contracts
Flux Board supports two official runtime contracts:

1. Docker image built from the repo-root [Dockerfile](../Dockerfile)
2. Self-contained root binary built from `go build .`

Both contracts assume:
- PostgreSQL is reachable at startup
- the root build embeds migrations, the legacy rollback shell, and the React runtime
- `APP_PASSWORD` is optional and bootstrap-only

## Required Environment
- `DATABASE_URL`
- `PORT`
- `APP_ENV`

Optional:
- `APP_PASSWORD`
- `APP_VERSION`
- `BUILD_VERSION`
- `OTEL_EXPORTER_OTLP_ENDPOINT`

### Env behavior
- `APP_ENV=development` is only for local plain-HTTP testing.
- Hosted deployments should use `APP_ENV=production`.
- `APP_PASSWORD` can be left empty.
  - If empty and no admin exists yet, finish setup in the browser at `/setup`.
  - If set and no admin exists yet, Flux Board seeds the first admin password automatically.
  - After bootstrap, changing `APP_PASSWORD` does not rotate the live password.
- tagged release artifacts and Docker images already embed the tracked `VERSION`.
- source-built hosted Docker paths should set `BUILD_VERSION` to the same release
  label so `/api/status` and tracing stay aligned with the deployed artifact.
- `APP_VERSION` is an override knob for operators who intentionally want runtime status or tracing to announce a different version label.

## Local Docker Path
Use [docker-compose.yml](../docker-compose.yml):

```sh
docker compose up --build
```

Open [http://localhost:8080](http://localhost:8080).

## Local Binary Path
Build the embedded runtime:

```sh
./scripts/verify-web.sh
go build -o flux-board .
./flux-board
```

This produces the same runtime contract that release artifacts now use.

## Hosted Docker-First Guidance
### Render
- A starter template is tracked at [deploy/render.yaml](../deploy/render.yaml).
- Required env vars:
  - `DATABASE_URL`
  - `APP_ENV=production`
  - optional `APP_PASSWORD`

### Railway
- Use the repo root so Railway builds the tracked Dockerfile.
- Required env vars:
  - `DATABASE_URL`
  - `APP_ENV=production`
  - optional `APP_PASSWORD`
  - optional `PORT=8080` if the platform does not inject it automatically

### Zeabur
- Use the repo root Dockerfile as the deployment source.
- Required env vars:
  - `DATABASE_URL`
  - `APP_ENV=production`
  - optional `APP_PASSWORD`

## Release Dry Run
Before cutting a release, run:

```sh
RELEASE_RUN_SMOKE=0 ./scripts/release-dry-run.sh
```

On Windows:

```powershell
$env:RELEASE_RUN_SMOKE="0"
./scripts/release-dry-run.ps1
```

This verifies:
- version metadata
- root self-contained binary build
- release checksum generation

If `web/dist` is missing, the dry-run script now builds it first.

## Post-Deploy Checks
After any deploy, verify:
1. `GET /healthz` returns `200`
2. `GET /readyz` returns `200`
3. first-run instance opens `/setup` or seeded instance opens `/login`
4. successful sign-in reaches `/board`
5. `/settings` loads password, session, retention, and backup controls
6. `/legacy/` remains reachable as the rollback shell

For a repo-owned status artifact set, run:

```sh
BASE_URL=https://your-host.example \
EXPECT_NEEDS_SETUP=false \
EXPECT_ENVIRONMENT=production \
./scripts/verify-status-contract.sh
```

Windows:

```powershell
$env:BASE_URL="https://your-host.example"
$env:EXPECT_NEEDS_SETUP="false"
$env:EXPECT_ENVIRONMENT="production"
./scripts/verify-status-contract.ps1
```

For a repo-owned hosted deployment verification artifact, prefer an explicit host target:

```sh
BASE_URL=https://your-host.example \
EXPECT_NEEDS_SETUP=false \
EXPECT_ENVIRONMENT=production \
./scripts/verify-hosted-deploy.sh
```

If you intentionally want the script to discover the live deployment from GitHub deployment metadata, opt in explicitly:

```sh
ALLOW_LIVE_DEPLOYMENT_DISCOVERY=1 \
HOSTED_ENVIRONMENT=production \
./scripts/verify-hosted-deploy.sh
```

Without `BASE_URL` or `ALLOW_LIVE_DEPLOYMENT_DISCOVERY=1`, `verify-hosted-deploy.sh` now exits immediately instead of probing production by default.

For a repo-owned hosted auth artifact on macOS with an already signed-in Chrome session, run:

```sh
BASE_URL=https://your-host.example \
./scripts/verify-hosted-auth-browser.sh
```

The script fails if Chrome lands anywhere other than `/board` and `/settings`, and it writes the captured URL/title evidence under `test-results/hosted-auth/...`.

## Backup And Restore Notes
- Use `/settings -> Export board data` before manual imports, host moves, or risky maintenance that could change the live board.
- Flux Board rejects malformed import bundles before replacing the current board snapshot. If an import fails, the current board and archive-retention policy stay in place; fix the JSON or re-export from the source instance before retrying.
- JSON export/import covers active tasks, archived tasks, and archive-retention settings. It is the right tool for logical board migration, but it is not a full-instance disaster-recovery substitute for auth/session/audit tables.
- For full-instance recovery, take a PostgreSQL backup before upgrades or destructive imports. On self-managed Postgres, `pg_dump "$DATABASE_URL" > flux-board-$(date +%F).sql` and `psql "$DATABASE_URL" < flux-board-YYYY-MM-DD.sql` remain the baseline workflow; managed snapshot equivalents are fine when the host provides them.
- The full operator drill now lives in [docs/BACKUP_RESTORE_DRILL.md](BACKUP_RESTORE_DRILL.md), and the incident/deploy troubleshooting flow lives in [docs/OPERATIONS_RUNBOOK.md](OPERATIONS_RUNBOOK.md).

## Rollback Baseline
Current rollback is intentionally simple:
1. stop the new app instance
2. redeploy the previous Docker image or previous root binary artifact
3. verify `/healthz`
4. verify `/readyz`
5. verify login and board access
6. keep `/legacy/` available as the HTML rollback shell during frontend incidents

## Current Gaps
- local `go build .` still assumes `web/dist` has already been built unless you use the Docker path or the release dry-run path
- provider-specific operational details like managed Postgres backups and TLS termination remain operator-owned
