# Flux Board Operations Runbook

## Purpose
- Use this runbook when a local or hosted Flux Board instance is unhealthy, newly deployed, or being checked for release evidence.
- The goal is to diagnose operator-facing failures from repo-owned signals before reaching for ad hoc database surgery.

## Core Signals
### `GET /healthz`
- Expected: `200` with `{"status":"ok"}`.
- Meaning: the Go process is up and serving HTTP.
- If it fails: treat it as a process/runtime startup issue first.

### `GET /readyz`
- Expected: `200` with `{"status":"ready"}`.
- Meaning: the app can reach PostgreSQL and finish its readiness check.
- If it returns `503`: the runtime is up, but the database path is not healthy enough for real traffic.

### `GET /api/status`
- Expected: `200` for `status=ready`, or `503` for `status=degraded`.
- This is the main operator contract for W14-W15. Record it during deploys and incident review.
- Stable fields to expect:
  - `version`
  - `environment`
  - `runtimeArtifact`
  - `runtimeOwnershipPath`
  - `legacyRollbackPath`
  - `needsSetup`
  - `archiveRetentionDays`
  - `archiveCleanupEvery`
  - `sessionCleanupEvery`
  - `generatedAt`
  - `checks`
- Stable check names:
  - `database`
  - `bootstrap`
  - `archive-retention`

### `GET /metrics`
- Expected: `200` with Prometheus text output.
- Current bounded HTTP metrics to verify:
  - `flux_board_http_requests_total`
  - `flux_board_http_request_duration_seconds`

### `GET /status`
- Expected: `200` with the React runtime HTML.
- Use it as the operator-facing route for the deployment overview UI.

## Fast Evidence Capture
- Preferred repo-owned check:

```sh
BASE_URL=https://your-host.example \
EXPECT_NEEDS_SETUP=false \
EXPECT_ENVIRONMENT=production \
./scripts/verify-status-contract.sh
```

- Windows:

```powershell
$env:BASE_URL="https://your-host.example"
$env:EXPECT_NEEDS_SETUP="false"
$env:EXPECT_ENVIRONMENT="production"
./scripts/verify-status-contract.ps1
```

- The script writes artifact files under `test-results/status-contract/...`, including:
  - `healthz.json`
  - `readyz.json`
  - `status.json`
  - `status-page.html`
  - `metrics.txt`
  - `summary.json`

## Common Failures
### Bad `DATABASE_URL`
- Symptoms:
  - `/healthz` may still return `200`
  - `/readyz` returns `503`
  - `/api/status` returns `503` with a failed `database` check
  - startup or request logs mention DB connect or ping failure
- Actions:
  - confirm the hostname, port, database name, user, password, and SSL mode
  - verify the database accepts network traffic from the app host
  - rerun `verify-status-contract` after fixing the env var

### Browser setup still required
- Symptoms:
  - `/api/status` shows `needsSetup=true`
  - bootstrap check message says browser setup is still required
  - `/setup` is the correct next operator step
- Actions:
  - finish `/setup` once
  - rerun the status verification with `EXPECT_NEEDS_SETUP=false`

### Stale cookies or revoked sessions
- Symptoms:
  - `/api/auth/me` returns `401`
  - `/settings` shows a missing or revoked session
  - sign-in succeeds again after clearing cookies or logging in fresh
- Actions:
  - sign out and sign in again
  - if the current browser was revoked intentionally, verify the active session list in `/settings`
  - do not treat this as a deploy outage unless fresh sign-in also fails

### Failed import payloads
- Symptoms:
  - `POST /api/import` returns `400`
  - `/settings` shows import failure feedback
  - the existing board remains unchanged
- Actions:
  - stop and keep the current live state untouched
  - validate the JSON bundle structure before retrying
  - if the operator is attempting a large restore, take a PostgreSQL backup first and follow [docs/BACKUP_RESTORE_DRILL.md](BACKUP_RESTORE_DRILL.md)

### Cleanup expectations and recovery
- Current contract:
  - if `archiveRetentionDays` is `null`, archived cards stay until manually deleted
  - if retention is set, cleanup runs on the repo-owned cleanup interval reported by `/api/status`
- Actions:
  - confirm the retention value in `/settings` and `/api/status`
  - if archived data disappeared unexpectedly, stop destructive imports and restore from the latest export or PostgreSQL backup
  - document the observed retention policy in the incident notes

## Hosted Release Checklist
- Record the exact Git commit and release tag or image tag being deployed.
- Run `verify-status-contract` against the real host and save the artifact directory.
- Confirm these routes manually or through existing smoke lanes:
  - `/healthz`
  - `/readyz`
  - `/api/status`
  - `/status`
  - `/setup` or `/login`
  - `/board`
  - `/settings`
  - `/legacy/`
- Save one deployment note that includes:
  - deployed version
  - database host class or provider
  - result of the status-contract run
  - rollback target image or binary version

## Recovery Baseline
- If the new deploy is unhealthy but the previous release was healthy:
  1. stop routing traffic to the unhealthy release
  2. redeploy the previous Docker image or root binary artifact
  3. verify `/healthz`, `/readyz`, and `/api/status`
  4. verify `/login` or `/setup`, `/board`, `/settings`, and `/legacy/`
  5. only then resume normal operator actions
