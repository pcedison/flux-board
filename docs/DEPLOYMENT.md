# Deployment Notes

## Current Scope
Flux Board is still in a transition phase between MVP and public-fork-ready baseline. These notes describe the current deployment shape only.

## Required Environment
- `DATABASE_URL`: PostgreSQL connection string
- `APP_PASSWORD`: first-run bootstrap admin seed
- `APP_ENV`: use `development` only for local HTTP testing
- `PORT`: bind port for the Go server

## Runtime Shape
- one Go binary serves both API routes and embedded static assets
- PostgreSQL is required at startup
- schema bootstrap still happens in-process
- auth currently centers on one bootstrap-created admin account with DB-backed sessions

## Current Hosting Expectations
- the app should run behind a trusted reverse proxy in hosted environments
- if proxy headers are passed through, the proxy must be the sole writer of `X-Forwarded-For` and `X-Real-IP`
- public internet deployment should still be treated as restricted or development-stage until later waves close

## Verification After Deploy
1. run backend verification with `./scripts/verify-go.ps1` or `./scripts/verify-go.sh`
2. set `FLUX_PASSWORD` or `APP_PASSWORD` for the current bootstrap admin, then run browser smoke with `npm run smoke:login`
3. confirm `/healthz` returns `200` once the process is serving HTTP
4. confirm `/readyz` returns `200` once the app can reach PostgreSQL
5. confirm `/api/auth/me` returns `401` before login and `200` after login

## Release Dry Run
Use the repo-owned release dry-run scripts before treating a build as publishable:

```powershell
./scripts/release-dry-run.ps1
```

On macOS/Linux:

```sh
./scripts/release-dry-run.sh
```

Current dry-run behavior:
- builds one versionable binary artifact
- emits a SHA-256 checksum beside that binary
- reuses the built artifact for browser smoke instead of rebuilding
- stores output and smoke diagnostics under `test-results/release/`

Required env for a realistic dry run:
- `DATABASE_URL`
- `APP_PASSWORD`
- `FLUX_PASSWORD` or `APP_PASSWORD`
- optional `PLAYWRIGHT_BROWSER` if you want a browser other than the default `chromium`

## Rollback Baseline
Current rollback is intentionally simple and binary-first:
1. stop the current app process
2. restore the previous known-good binary artifact and matching environment values
3. start the restored binary
4. verify `/healthz` returns `200`
5. verify `/readyz` returns `200`
6. run `./scripts/verify-smoke.ps1` or `./scripts/verify-smoke.sh` against the restored binary
7. confirm `/api/auth/me` is `401` before login and returns `200` after login

If rollback fails any readiness or smoke step, keep the app out of rotation and inspect the release diagnostics saved under `test-results/`.

## Current Non-Goals
- no migration framework yet
- no multi-user auth model yet
- no React/Vite frontend yet
