# Flux Board

Flux Board is a Go + PostgreSQL task board that is currently being upgraded from an MVP into an enterprise-grade, public-fork-ready project.

## Current Status
- Current runtime: Go backend with the React runtime on `/`, legacy rollback UI on `/legacy/`, and `/next/*` compatibility redirects.
- Current maturity: MVP that works, but is not yet production-hardened for public deployment.
- Active upgrade plan: [docs/MASTER_PLAN.md](docs/MASTER_PLAN.md).
- Read-first status handoff: [docs/STATUS_HANDOFF.md](docs/STATUS_HANDOFF.md).
- New frontend baseline: `web/` React + TypeScript + Vite scaffold now serving the canonical runtime on `/`, with React Router, TanStack Query, auth-aware routing, explicit sign-out handling, and the first non-drag board mutation path.

## Current Features
- Bootstrap-password-protected task board with DB-backed sessions
- Create, edit, archive, restore, and delete tasks
- Three board states: `queued`, `active`, `done`
- PostgreSQL persistence
- Unauthenticated `GET /healthz` and DB-backed `GET /readyz` probes
- API and probe request-id/access-log baseline for easier diagnosis
- React/Vite shell served by the Go app on `/`, with `/login`, guarded `/board`, auth-aware shell navigation, explicit sign-out handling, explicit create/move/archive/restore actions, and lane-local move up/down fallback
- Legacy embedded frontend preserved on `/legacy/` as the current rollback path

## Planned Direction
- Go API + PostgreSQL remains the backend foundation
- Frontend will be rebuilt as `React + TypeScript + Vite`
- RWD, accessibility, CI, migrations, and stronger security are part of the active roadmap

## Local Development
### Requirements
- Go `1.24+`
- PostgreSQL
- Node.js `20.19+` or `22.12+` if you want to run the `web/` scaffold or browser smoke tooling
- On Windows, local `go test -race` also needs a C toolchain such as `MSYS2 UCRT64 GCC`

### Environment
Create a local `.env` from `.env.example` and set:
- `DATABASE_URL`
- `APP_PASSWORD`
- `APP_ENV=development`
- `PORT`

### Run
```powershell
go run ./cmd/flux-board
```

Open `http://localhost:8080`.
If `web/dist` has been built, the Go server serves the React runtime from `http://localhost:8080/`, keeps the embedded rollback shell at `http://localhost:8080/legacy/`, and redirects old `/next/*` preview URLs to the canonical root routes.

Health probes for local or hosted verification:
- `GET /healthz` returns `200` once the HTTP process is running
- `GET /readyz` returns `200` only after the app can reach PostgreSQL

### Deployment Basics
Current deployment assumptions for the Go binary are:
- `DATABASE_URL` points to a reachable PostgreSQL instance
- `APP_PASSWORD` seeds the initial bootstrap admin on first startup
- `APP_ENV=development` only for local HTTP testing; hosted environments should use production defaults
- `PORT` is supplied by the platform or defaults to `8080`

Hosted deployments that need the canonical React runtime on `/` should use the repo-root `Dockerfile`, because it builds `web/dist` and ships it with the Go binary, migrations, and rollback assets in one image. Platform-native Go builds that skip the frontend step will leave the React runtime unavailable at `/`.

### Verify
```powershell
./scripts/verify-go.ps1
```

On macOS/Linux:
```sh
./scripts/verify-go.sh
```

Local race verification:
```powershell
./scripts/verify-go-race.ps1
```

On macOS/Linux:
```sh
./scripts/verify-go-race.sh
```

The Windows race script expects `C:\msys64\ucrt64\bin` and will temporarily wire `CGO_ENABLED=1`, `CC`, and `CXX` for the current shell before running `go test -race`.
The Go verification scripts now discover repo Go packages dynamically while excluding `web/` and other non-Go artifact directories, so local and CI coverage stay aligned as the codebase grows.

Verify the isolated `web/` scaffold:
```powershell
./scripts/verify-web.ps1
```

On macOS/Linux:
```sh
./scripts/verify-web.sh
```

These scripts now run install, typecheck, frontend unit tests, and production build for `web/`.

Browser smoke for the canonical React runtime:
```powershell
npm ci
$env:FLUX_PASSWORD="your-password"
$env:PLAYWRIGHT_BROWSER="chromium"
./scripts/verify-smoke.ps1
```

On macOS/Linux:
```sh
npm ci
export FLUX_PASSWORD="your-password"
export PLAYWRIGHT_BROWSER=chromium
./scripts/verify-smoke.sh
```

These smoke scripts now build the Go app, start it locally, wait for `/readyz`, run the repo-owned Playwright smoke flow, keep logs under `test-results/`, and clean up the app process automatically.
`PLAYWRIGHT_BROWSER` defaults to `chromium`; CI now also runs the same smoke path against `firefox` as the first non-Chromium browser gate.

Compatibility-alias and rollback smoke for `/next/*` plus `/legacy/`:
```powershell
./scripts/verify-next-preview.ps1
```

On macOS/Linux:
```sh
./scripts/verify-next-preview.sh
```

These scripts verify the `web/` scaffold, build `web/dist`, then run a dedicated Playwright flow that starts from `/next/login`, confirms the compatibility redirect into the canonical runtime, and verifies that `/legacy/` still exposes the embedded rollback shell.

For probe-based startup checks, use `http://127.0.0.1:8080/readyz` instead of relying on auth endpoints.
Observed API and probe responses now include `X-Request-Id`, and the Go server logs matching request IDs with client, method, path, status, bytes, and duration for `/api/*`, `/healthz`, and `/readyz`.

The Vite dev server is configured to proxy `/api/*` to `http://127.0.0.1:8080` by default, so the isolated `web/` shell can talk to a local Go app without extra setup.

For release-style verification, run:
```powershell
./scripts/release-dry-run.ps1
```

On macOS/Linux:
```sh
./scripts/release-dry-run.sh
```

These scripts build a versionable binary artifact, emit a SHA-256 checksum, and then reuse that binary for the same smoke flow instead of rebuilding again. See [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) for the current release dry-run and rollback baseline.

## Repository Layout
- [cmd/flux-board/main.go](cmd/flux-board/main.go): canonical backend entrypoint for local runs and future packaging
- [main.go](main.go): transitional root shim that still assembles the same app for repo-local scripts
- [internal/domain/](internal/domain): task and auth domain types, task validation rules, and repository contracts
- [internal/store/postgres/](internal/store/postgres): PostgreSQL repositories, migrations, and maintenance helpers
- [internal/service/task/service.go](internal/service/task/service.go) and [internal/service/auth/service.go](internal/service/auth/service.go): task/auth service orchestration and validation seams
- [internal/transport/http/](internal/transport/http): HTTP handlers, mux/server assembly, cookies/context helpers, request decoding, root-runtime/rollback serving, and observability middleware
- [app_bootstrap.go](app_bootstrap.go): root-level DB/app wiring helpers
- [app_runtime.go](app_runtime.go): root-level background loop wiring and security-header middleware wrapper
- [app_state.go](app_state.go): shared app state and dependency container used by the root shim/tests
- [static/index.html](static/index.html): legacy embedded frontend kept as the rollback shell on `/legacy/`
- [web/](web): isolated React + TypeScript + Vite scaffold for the future frontend rebuild
- [web/src/lib/useBoardMutations.ts](web/src/lib/useBoardMutations.ts): React Query mutation layer for the isolated board shell
- [scripts/verify-smoke.ps1](scripts/verify-smoke.ps1) and [scripts/verify-smoke.sh](scripts/verify-smoke.sh): repo-owned app startup, readiness, Playwright smoke, and cleanup orchestration
- [docs/MASTER_PLAN.md](docs/MASTER_PLAN.md): master execution plan and progress log
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md): current and target architecture
- [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md): current deployment assumptions and post-deploy checks

## Known Current Limitations
- Current auth model is now a safer single-admin baseline with DB-backed sessions and audit logging, but it is not yet a multi-user or OIDC-backed auth model
- The current migration baseline and reorder integrity path are in place for the embedded single-board model, but broader schema/domain normalization remains for later waves
- Browser smoke coverage for the canonical React runtime is repo-owned and CI-backed across `chromium`, `firefox`, and `webkit`, and the `web/` scaffold has build/typecheck/unit tests plus dedicated drag and keyboard smoke lanes
- Automated backend verification is still light and currently centered on Go checks plus focused unit tests
- The current default user-facing runtime is now the React app on `/`; the old embedded HTML shell remains available on `/legacy/` for rollback while `/next/*` redirects into the canonical routes
- Observability now includes structured `slog` request logging, request-id correlation, Prometheus metrics on `/metrics`, and an optional OTLP tracing seam, but production dashboards/alerts are still deployment-specific work
- Release governance now has `VERSION`, `CHANGELOG.md`, cross-platform GitHub Releases, and checksum publication, but public internet production hardening still needs operator-owned secrets, reverse-proxy policy, and service-level monitoring

## Governance Docs
- [CONTRIBUTING.md](CONTRIBUTING.md)
- [SECURITY.md](SECURITY.md)
- [LICENSE](LICENSE)

## Public Fork Note
This repository now has the corrected `W1-W4` baseline in place, but it is still not at the final security and engineering target described in the master plan. Until later waves such as migrations, deeper modularization, and the React/Vite rebuild are complete, treat public deployment as development-only unless you apply your own hardening.
