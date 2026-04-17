# Architecture

## Current Architecture
Flux Board currently runs as a single Go application that:
- reads environment variables
- opens a PostgreSQL connection pool
- initializes schema in-process
- serves API routes and embedded static assets from the same binary

## Current Runtime Flow
1. `main.go` loads `DATABASE_URL`, `APP_PASSWORD`, and runtime settings
2. the app creates a `pgxpool` connection
3. schema bootstrap runs directly from the app, including first-run bootstrap admin/session tables
4. auth, tasks, and archive routes are registered with shared middleware
5. a built `web/dist` is served by the same process as the canonical React runtime on `/`, `static/index.html` is preserved on `/legacy/` as the rollback shell, and old `/next/*` preview URLs are redirected into the root runtime

## Current Backend Shape
- entrypoint: [main.go](../main.go)
- persistence: PostgreSQL through `pgx/pgxpool`
- auth: `APP_PASSWORD` seeds the first admin row on bootstrap; subsequent login uses the stored bcrypt hash and PostgreSQL-backed sessions
- auth audit: auth events are recorded into PostgreSQL audit rows during login, logout, and invalid session handling
- transport hardening: explicit `http.Server`, timeouts, graceful shutdown, strict JSON decode, and baseline security headers
- archive cleanup: periodic in-process goroutine with cancellation-aware lifecycle
- session cleanup: periodic in-process goroutine with cancellation-aware lifecycle

## Current Frontend Shape
- canonical Go-served `React + TypeScript + Vite` runtime from built `web/dist` on `/`
- legacy embedded rollback file: [static/index.html](../static/index.html) on `/legacy/`
- `/next/*` remains as a compatibility redirect into the canonical root routes
- non-drag create/move/archive/restore plus lane-local move up/down now exist in the React board

## Current Coupling Risks
- backend logic is concentrated in one file
- frontend markup, style, and behavior are tightly coupled
- sorting rules are split between frontend and backend
- archive retention logic is duplicated in frontend and backend
- auth is safer than the original MVP, but still centered on one admin account

## Target Architecture
### Backend
- `cmd/flux-board` entrypoint
- layered `internal/config`, `internal/http`, `internal/service`, `internal/store/postgres`
- migrations for schema evolution
- stronger auth and session model

### Frontend
- `web/` app using `React + TypeScript + Vite`
- Go-served root runtime on `/` with `/legacy/` rollback and transitional `/next/*` redirects
- componentized board UI
- touch, keyboard, and mouse friendly movement
- mobile/tablet/desktop responsive layouts

## Source Of Truth
Execution priority and rollout order are defined in [docs/MASTER_PLAN.md](MASTER_PLAN.md).
