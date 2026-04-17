# Deployment Notes

## Current Scope
Flux Board is still in a transition phase between MVP and public-fork-ready baseline. These notes describe the current deployment shape only.

## Required Environment
- `DATABASE_URL`: PostgreSQL connection string
- `APP_PASSWORD`: first-run bootstrap admin seed
- `APP_ENV`: use `development` only for local HTTP testing
- `PORT`: bind port for the Go server
- `APP_VERSION`: optional runtime version label for logs/traces; release automation can set this to the matching `VERSION`

## Runtime Shape
- one Go binary serves API routes, the canonical React runtime on `/`, and the legacy rollback shell on `/legacy/`
- preferred local/manual entrypoint is `go run ./cmd/flux-board`
- if `web/dist` has been built, the same Go binary serves the React runtime from `/` and redirects old `/next/*` preview URLs into the canonical root routes
- PostgreSQL is required at startup
- schema bootstrap still happens in-process
- auth currently centers on one bootstrap-created admin account with DB-backed sessions

## Current Hosting Expectations
- the app should run behind a trusted reverse proxy in hosted environments
- if proxy headers are passed through, the proxy must be the sole writer of `X-Forwarded-For` and `X-Real-IP`
- public internet deployment should still be treated as restricted or development-stage until later waves close
- hosted platforms that auto-detect a plain Go build should prefer the repo-root `Dockerfile`, because the canonical `/` runtime depends on `web/dist` being built and shipped with the binary
- for Zeabur specifically, point the service at the repository root so it uses the `Dockerfile`, then provide `DATABASE_URL`, `APP_PASSWORD`, and any desired `APP_VERSION`/`OTEL_EXPORTER_OTLP_ENDPOINT` values through the platform env UI

## Enterprise Deployment Seams
- These seams are documentation only for `W9/5-B`; the current runtime does not yet enable RBAC, SSO, SCIM, or multi-workspace routing.
- Keep the local bootstrap-admin login path available as the break-glass path until any future SSO rollout is independently verified and remotely closed.
- Future SSO work should trust only explicit app-owned callback flows; do not introduce ambient identity headers from the reverse proxy as an unofficial auth path.
- Future workspace rollout should happen after a default-workspace backfill migration, so existing single-board rows can be promoted safely before any workspace-specific routing or access policy is enabled.
- If later enterprise slices add IdP or workspace env vars, deployment docs should treat them as opt-in and keep the current `DATABASE_URL` plus bootstrap-admin startup path as the fallback until the new path is proven.

## Verification After Deploy
1. run backend verification with `./scripts/verify-go.ps1` or `./scripts/verify-go.sh`
2. set `FLUX_PASSWORD` or `APP_PASSWORD` for the current bootstrap admin, then run browser smoke with `npm run smoke:login`
3. confirm `/healthz` returns `200` once the process is serving HTTP
4. confirm `/readyz` returns `200` once the app can reach PostgreSQL
5. confirm `/api/auth/me` returns `401` before login and `200` after login
6. confirm `/metrics` returns Prometheus text output when observability scraping is expected

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
- no final multi-board/domain migration strategy yet beyond the current baseline
- no multi-user auth model yet
- no production SSO, SCIM, or workspace-aware routing yet
- no operator-owned dashboard/alert bundle is shipped in-repo; hosted monitoring remains deployment-specific
