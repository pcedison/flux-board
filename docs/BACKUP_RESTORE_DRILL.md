# Flux Board Backup And Restore Drill

## Purpose
- This is the W16 operator drill for proving that Flux Board can be backed up and restored without guesswork.
- Run it before risky imports, before major hosted changes, and on a regular rehearsal cadence for long-lived deployments.

## Expected Artifacts
- one JSON export from `/settings` or `GET /api/export`
- one PostgreSQL dump from the live database
- one scratch restore target
- one automation results directory from `./scripts/verify-restore-drill.sh` or `./scripts/verify-restore-drill.ps1`

## Preconditions
- You can sign in to `/settings`.
- You can reach the production database with PostgreSQL tooling.
- You have a scratch database or disposable restore target.
- You are not treating JSON import as a substitute for a full database backup.

## Repo-Owned Automation
The repo-owned scratch rehearsal now lives in:
- `./scripts/verify-restore-drill.sh`
- `./scripts/verify-restore-drill.ps1`

The script takes an existing custom-format PostgreSQL dump, restores it into `RESTORE_DATABASE_URL`, starts a scratch Flux Board process against that restored database, runs `verify-status-contract`, signs in with `FLUX_PASSWORD`, opens `/board` and `/settings`, and saves the post-restore `/api/export` bundle plus screenshots under `test-results/restore-drill/`.

Required env:
- `RESTORE_DRILL_DUMP_PATH`: path to the `pg_dump --format=custom` artifact you want to rehearse
- `RESTORE_DATABASE_URL`: empty or disposable scratch database URL
- `FLUX_PASSWORD` or `APP_PASSWORD`: current sign-in password stored in the dump

Common optional env:
- `BASE_URL`: scratch app URL to bind and verify, default `http://127.0.0.1:8080`
- `APP_BINARY`: output path for the scratch binary, default repo-root `./flux-board`
- `RESTORE_DRILL_PG_RESTORE_BIN`: alternate `pg_restore` path or wrapper when it is not on `PATH`
- `VERIFY_RESTORE_DRILL_BUILD=0`: reuse an existing app build
- `VERIFY_RESTORE_DRILL_WEB_BUILD=0`: reuse an existing `web/dist`

Example on macOS or Linux:

```sh
npm ci
export FLUX_PASSWORD="your-current-password"
export RESTORE_DRILL_DUMP_PATH="$PWD/backups/flux-board-20260419-103000.dump"
export RESTORE_DATABASE_URL='postgres://flux:flux@127.0.0.1:5432/flux_restore?sslmode=disable'
./scripts/verify-restore-drill.sh
```

Example on PowerShell:

```powershell
npm ci
$env:FLUX_PASSWORD = "your-current-password"
$env:RESTORE_DRILL_DUMP_PATH = "$PWD\backups\flux-board-20260419-103000.dump"
$env:RESTORE_DATABASE_URL = "postgres://flux:flux@127.0.0.1:5432/flux_restore?sslmode=disable"
./scripts/verify-restore-drill.ps1
```

The results directory includes:
- `dump.sha256.txt`
- `pg-restore.stdout.log` and `pg-restore.stderr.log`
- `server.stdout.log` and `server.stderr.log`
- `status-contract/` artifacts
- `browser/` screenshots, `/api/settings`, `/api/tasks`, `/api/export`, and `summary.json`

## Drill A: Capture The Current Backup Set
1. Sign in to `/settings`.
2. Export the current board data and save the JSON bundle with a timestamped filename.
3. Capture a PostgreSQL backup from the live database.

```sh
timestamp=$(date +"%Y%m%d-%H%M%S")
pg_dump --format=custom --file "flux-board-$timestamp.dump" "$DATABASE_URL"
shasum -a 256 "flux-board-$timestamp.dump"
```

4. Record the current deployment status:

```sh
BASE_URL=https://your-host.example \
EXPECT_NEEDS_SETUP=false \
EXPECT_ENVIRONMENT=production \
./scripts/verify-status-contract.sh
```

## Drill B: Rehearse The Scratch Restore
1. Create or choose an empty scratch database URL, referenced here as `RESTORE_DATABASE_URL`.
2. Run the repo-owned restore drill script with the current dump and password.
3. Confirm the script finishes with `Restore drill completed successfully`.
4. Save the results directory path beside the dump filename and checksum.

## Drill C: Optional Import Rehearsal
1. If you need to rehearse JSON import, do it only on the scratch instance from Drill B.
2. Confirm the imported board loads and that the retention policy matches the expected settings.
3. Re-run the restore drill afterward if you want a fresh post-restore verification record.

## Success Criteria
- The dump restores without manual table surgery.
- The scratch instance reaches `ready`.
- The scratch instance can sign in and open `/board` and `/settings`.
- The exported JSON bundle remains structurally valid after restore.
- The operator can describe which artifact is the fastest rollback path:
  - JSON import for board-content recovery
  - PostgreSQL restore for full-instance recovery

## Failure Handling
### Import rejected
- Stop and keep the live deployment unchanged.
- Fix the JSON bundle in scratch first.
- Do not bypass validation by editing production data directly.

### Restore fails
- Treat it as a release blocker for backup confidence.
- Capture the failing `pg_restore` output and database version details.
- Keep `pg-restore.stderr.log` from the restore drill results directory.
- Do not claim W16 progress until the drill can be repeated cleanly.

### Restored app is unhealthy
- Review `status-contract/summary.json`, `server.stderr.log`, and the browser `summary.json` from the restore drill results directory.
- Check `database`, `bootstrap`, and `archive-retention` in `/api/status`.
- Fix the environment or schema mismatch before repeating the drill.

## Recommended Cadence
- Before destructive imports.
- Before major hosted platform changes.
- After schema or backup-contract changes.
- At a regular rehearsal interval chosen by the operator.

## What To Record
- dump filename and checksum
- export filename and checksum
- scratch restore target used
- restore-drill artifact directory
- whether login, board, and settings all worked after restore
