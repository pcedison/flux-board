# Flux Board Backup And Restore Drill

## Purpose
- This is the W16 operator drill for proving that Flux Board can be backed up and restored without guesswork.
- Run it before risky imports, before major hosted changes, and on a regular rehearsal cadence for long-lived deployments.

## Expected Artifacts
- one JSON export from `/settings` or `GET /api/export`
- one PostgreSQL dump from the live database
- one scratch restore target
- one post-restore status verification record from `./scripts/verify-status-contract.sh`

## Preconditions
- You can sign in to `/settings`.
- You can reach the production database with PostgreSQL tooling.
- You have a scratch database or disposable restore target.
- You are not treating JSON import as a substitute for a full database backup.

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

## Drill B: Restore Into A Scratch Target
1. Create or choose an empty scratch database URL, referenced here as `RESTORE_DATABASE_URL`.
2. Restore the dump into that scratch target.

```sh
pg_restore \
  --clean \
  --if-exists \
  --no-owner \
  --dbname "$RESTORE_DATABASE_URL" \
  "flux-board-$timestamp.dump"
```

3. Start a scratch Flux Board instance against `RESTORE_DATABASE_URL`.
4. Verify the restored runtime:
  - `/healthz`
  - `/readyz`
  - `/api/status`
  - `/login`
  - `/board`
  - `/settings`

## Drill C: Validate Export And Import Safety
1. Open `/settings` on the scratch instance.
2. Export again and confirm the bundle is structurally valid:
  - `version` is present
  - `exportedAt` is present
  - `settings` is present
  - `tasks` is an array
  - `archived` is an array
3. If you need to rehearse JSON import, do it only on the scratch instance.
4. Confirm the imported board loads and that the retention policy matches the expected settings.

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
- Do not claim W16 progress until the drill can be repeated cleanly.

### Restored app is unhealthy
- Run `verify-status-contract` against the scratch target.
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
- status-contract artifact directory
- whether login, board, and settings all worked after restore
