# Flux Board

Flux Board is a Go + PostgreSQL task board that is currently being upgraded from an MVP into an enterprise-grade, public-fork-ready project.

## Current Status
- Current runtime: Go backend with embedded static frontend.
- Current maturity: MVP that works, but is not yet production-hardened for public deployment.
- Active upgrade plan: [docs/MASTER_PLAN.md](docs/MASTER_PLAN.md).
- New frontend baseline: isolated `web/` React + TypeScript + Vite scaffold with React Router, TanStack Query, auth-aware routing, and read-only API integration.

## Current Features
- Bootstrap-password-protected task board with DB-backed sessions
- Create, edit, archive, restore, and delete tasks
- Three board states: `queued`, `active`, `done`
- PostgreSQL persistence
- Unauthenticated `GET /healthz` and DB-backed `GET /readyz` probes
- Embedded frontend served by the Go app
- Isolated React/Vite shell with `/login` and guarded `/board` routes for the next frontend

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

Optional browser smoke for the current embedded frontend:
```powershell
npm ci
$env:FLUX_PASSWORD="your-password"
npm run smoke:login
```

For probe-based startup checks, use `http://127.0.0.1:8080/readyz` instead of relying on auth endpoints.

The Vite dev server is configured to proxy `/api/*` to `http://127.0.0.1:8080` by default, so the read-only `web/` shell can talk to a local Go app without extra setup.

## Repository Layout
- [main.go](main.go): current backend entrypoint and API server
- [health_http.go](health_http.go): unauthenticated liveness/readiness handlers
- [auth_http.go](auth_http.go): auth/session HTTP handlers and middleware
- [http_helpers.go](http_helpers.go): shared JSON request/response helpers
- [tasks_http.go](tasks_http.go): task and archive HTTP handlers plus task payload validation
- [static/index.html](static/index.html): current embedded frontend
- [web/](web): isolated React + TypeScript + Vite scaffold for the future frontend rebuild
- [docs/MASTER_PLAN.md](docs/MASTER_PLAN.md): master execution plan and progress log
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md): current and target architecture
- [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md): current deployment assumptions and post-deploy checks

## Known Current Limitations
- Current auth model is now a safer single-admin baseline with DB-backed sessions and audit logging, but it is not yet a multi-user or OIDC-backed auth model
- The current migration baseline and reorder integrity path are in place for the embedded single-board model, but broader schema/domain normalization remains for later waves
- Browser smoke coverage for the current embedded frontend is repo-owned and CI-backed, and the isolated `web/` scaffold now has build/typecheck/unit tests plus auth-aware routing, but the future React/Vite frontend is still not deployed or board-mutation-capable
- Automated backend verification is still light and currently centered on Go checks plus focused unit tests
- The current user-facing runtime still depends on a single embedded HTML file until later W7/W8 integration waves
- Health/readiness probes are intentionally minimal today and do not yet expose richer observability signals
- Health/readiness probes are intentionally minimal today and do not yet expose richer observability signals

## Governance Docs
- [CONTRIBUTING.md](CONTRIBUTING.md)
- [SECURITY.md](SECURITY.md)
- [LICENSE](LICENSE)

## Public Fork Note
This repository now has the corrected `W1-W4` baseline in place, but it is still not at the final security and engineering target described in the master plan. Until later waves such as migrations, deeper modularization, and the React/Vite rebuild are complete, treat public deployment as development-only unless you apply your own hardening.
