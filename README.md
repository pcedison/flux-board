# Flux Board

Flux Board is a Go + PostgreSQL task board that is currently being upgraded from an MVP into an enterprise-grade, public-fork-ready project.

## Current Status
- Current runtime: Go backend with embedded static frontend.
- Current maturity: MVP that works, but is not yet production-hardened for public deployment.
- Active upgrade plan: [docs/MASTER_PLAN.md](docs/MASTER_PLAN.md).
- New frontend baseline: isolated `web/` React + TypeScript + Vite scaffold with React Router, TanStack Query, auth-aware routing, explicit sign-out handling, and the first non-drag board mutation path.

## Current Features
- Bootstrap-password-protected task board with DB-backed sessions
- Create, edit, archive, restore, and delete tasks
- Three board states: `queued`, `active`, `done`
- PostgreSQL persistence
- Unauthenticated `GET /healthz` and DB-backed `GET /readyz` probes
- API and probe request-id/access-log baseline for easier diagnosis
- Embedded frontend served by the Go app
- Isolated React/Vite shell with `/login`, guarded `/board`, auth-aware shell navigation, explicit sign-out handling, explicit create/move/archive/restore actions, and lane-local move up/down fallback for the next frontend

## Planned Direction
- Go API + PostgreSQL remains the backend foundation
- Frontend will be rebuilt as `React + TypeScript + Vite`
- RWD, accessibility, CI, migrations, and stronger security are part of the active roadmap

## Local Development
### Requirements
- Go `1.22+`
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
go run .
```

Open `http://localhost:8080`.
If `web/dist` has been built, the Go server also exposes the React preview shell on `http://localhost:8080/next/` while keeping the embedded frontend on `/`.

Health probes for local or hosted verification:
- `GET /healthz` returns `200` once the HTTP process is running
- `GET /readyz` returns `200` only after the app can reach PostgreSQL

### Deployment Basics
Current deployment assumptions for the Go binary are:
- `DATABASE_URL` points to a reachable PostgreSQL instance
- `APP_PASSWORD` seeds the initial bootstrap admin on first startup
- `APP_ENV=development` only for local HTTP testing; hosted environments should use production defaults
- `PORT` is supplied by the platform or defaults to `8080`

This is still a transition-stage deployment model. It is suitable for development or restricted environments, not yet for open public production use.

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

Browser smoke for the current embedded frontend:
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

Preview-route smoke for the Go-served React shell:
```powershell
./scripts/verify-next-preview.ps1
```

On macOS/Linux:
```sh
./scripts/verify-next-preview.sh
```

These scripts verify the `web/` scaffold, build `web/dist`, then run a dedicated Playwright smoke flow against `/next/login` and `/next/board` without replacing the legacy `/` runtime.

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
- [main.go](main.go): current backend entrypoint and API server
- [app_bootstrap.go](app_bootstrap.go): app construction and bootstrap-only admin seeding
- [app_runtime.go](app_runtime.go): background loops and shared runtime middleware helpers
- [app_state.go](app_state.go): shared app state and cross-cutting runtime types
- [health_http.go](health_http.go): unauthenticated liveness/readiness handlers
- [auth_http.go](auth_http.go): auth/session HTTP handlers and middleware
- [auth_cookie.go](auth_cookie.go): cookie helpers for login/logout transport logic
- [auth_context.go](auth_context.go): shared auth context and client-IP helpers
- [auth_runtime.go](auth_runtime.go): login throttling and token-generation runtime helpers
- [auth_service.go](auth_service.go): auth/session persistence and audit helpers behind the transport layer
- [web_preview.go](web_preview.go): Go-served `/next/` React preview route with SPA fallback for built `web/dist`
- [http_helpers.go](http_helpers.go): shared JSON request/response helpers
- [server_observability.go](server_observability.go): request-id and access-log middleware at the server boundary
- [tasks_http.go](tasks_http.go): task and archive HTTP handlers
- [task_service.go](task_service.go): task validation and service-layer orchestration for CRUD/reorder flows
- [task_validation.go](task_validation.go): pure task validation and ID-normalization rules shared by the service seam
- [static/index.html](static/index.html): current embedded frontend
- [web/](web): isolated React + TypeScript + Vite scaffold for the future frontend rebuild
- [web/src/lib/useBoardMutations.ts](web/src/lib/useBoardMutations.ts): React Query mutation layer for the isolated board shell
- [scripts/verify-smoke.ps1](scripts/verify-smoke.ps1) and [scripts/verify-smoke.sh](scripts/verify-smoke.sh): repo-owned app startup, readiness, Playwright smoke, and cleanup orchestration
- [docs/MASTER_PLAN.md](docs/MASTER_PLAN.md): master execution plan and progress log
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md): current and target architecture
- [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md): current deployment assumptions and post-deploy checks

## Known Current Limitations
- Current auth model is now a safer single-admin baseline with DB-backed sessions and audit logging, but it is not yet a multi-user or OIDC-backed auth model
- The current migration baseline and reorder integrity path are in place for the embedded single-board model, but broader schema/domain normalization remains for later waves
- Browser smoke coverage for the current embedded frontend is repo-owned and CI-backed, and the isolated `web/` scaffold now has build/typecheck/unit tests, scoped non-drag board mutations, basic focus continuity, and a Go-served `/next/` preview route, but the future React/Vite frontend is still not the production runtime owner
- Automated backend verification is still light and currently centered on Go checks plus focused unit tests
- The current default user-facing runtime still depends on the embedded HTML shell on `/`; the React app now has a Go-served preview route on `/next/`, but it is not yet the production runtime owner
- Health/readiness probes and auth audit paths now expose a minimal request-id/access-log correlation baseline, but richer observability such as metrics, tracing, and structured logs remains for later W9 slices
- Release governance now has a first dry-run and rollback baseline, but there is still no versioning/changelog policy or multi-platform release matrix

## Governance Docs
- [CONTRIBUTING.md](CONTRIBUTING.md)
- [SECURITY.md](SECURITY.md)
- [LICENSE](LICENSE)

## Public Fork Note
This repository now has the corrected `W1-W4` baseline in place, but it is still not at the final security and engineering target described in the master plan. Until later waves such as migrations, deeper modularization, and the React/Vite rebuild are complete, treat public deployment as development-only unless you apply your own hardening.
