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
3. confirm `/api/auth/me` returns `401` before login and `200` after login

## Current Non-Goals
- no migration framework yet
- no multi-user auth model yet
- no React/Vite frontend yet
