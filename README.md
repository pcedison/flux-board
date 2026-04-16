# Flux Board

Flux Board is a Go + PostgreSQL task board that is currently being upgraded from an MVP into an enterprise-grade, public-fork-ready project.

## Current Status
- Current runtime: Go backend with embedded static frontend.
- Current maturity: MVP that works, but is not yet production-hardened for public deployment.
- Active upgrade plan: [docs/MASTER_PLAN.md](docs/MASTER_PLAN.md).

## Current Features
- Bootstrap-password-protected task board with DB-backed sessions
- Create, edit, archive, restore, and delete tasks
- Three board states: `queued`, `active`, `done`
- PostgreSQL persistence
- Embedded frontend served by the Go app

## Planned Direction
- Go API + PostgreSQL remains the backend foundation
- Frontend will be rebuilt as `React + TypeScript + Vite`
- RWD, accessibility, CI, migrations, and stronger security are part of the active roadmap

## Local Development
### Requirements
- Go `1.22+`
- PostgreSQL
- Node.js `18+` only if you want to run the optional browser smoke tooling

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

Optional browser smoke for the current embedded frontend:
```powershell
npm ci
$env:FLUX_PASSWORD="your-password"
npm run smoke:login
```

## Repository Layout
- [main.go](main.go): current backend entrypoint and API server
- [static/index.html](static/index.html): current embedded frontend
- [docs/MASTER_PLAN.md](docs/MASTER_PLAN.md): master execution plan and progress log
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md): current and target architecture
- [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md): current deployment assumptions and post-deploy checks

## Known Current Limitations
- Current auth model still centers on a single bootstrap-created admin account; it is safer than the original MVP, but not yet a multi-user or public-fork-final auth model
- No formal migration system yet
- Browser smoke coverage now has a repo-owned local script, but it is not yet wired into a full CI-grade frontend pipeline
- Automated backend verification is still light and currently centered on Go checks plus focused unit tests
- Current frontend still depends on a single embedded HTML file

## Governance Docs
- [CONTRIBUTING.md](CONTRIBUTING.md)
- [SECURITY.md](SECURITY.md)
- [LICENSE](LICENSE)

## Public Fork Note
This repository is being prepared for public-fork use, but it is not yet at the final security and engineering baseline described in the master plan. Until the corrected `W1-W4` gates are closed, treat public deployment as development-only unless you apply your own hardening.
