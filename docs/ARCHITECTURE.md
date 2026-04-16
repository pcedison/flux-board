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
5. embedded files under `static/` are served by the same process, and a built `web/dist` can be exposed on `/next/` as a preview runtime route

## Current Backend Shape
- entrypoint: [main.go](../main.go)
- persistence: PostgreSQL through `pgx/pgxpool`
- auth: `APP_PASSWORD` seeds the first admin row on bootstrap; subsequent login uses the stored bcrypt hash and PostgreSQL-backed sessions
- auth audit: auth events are recorded into PostgreSQL audit rows during login, logout, and invalid session handling
- transport hardening: explicit `http.Server`, timeouts, graceful shutdown, strict JSON decode, and baseline security headers
- archive cleanup: periodic in-process goroutine with cancellation-aware lifecycle
- session cleanup: periodic in-process goroutine with cancellation-aware lifecycle

## Current Frontend Shape
- single embedded file: [static/index.html](../static/index.html)
- optional Go-served preview route for built `web/dist`: `/next/`
- state is DOM- and script-coupled
- drag-and-drop is currently the main card movement path

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
- Go-served preview route on `/next/` before full runtime takeover
- componentized board UI
- touch, keyboard, and mouse friendly movement
- mobile/tablet/desktop responsive layouts

## Source Of Truth
Execution priority and rollout order are defined in [docs/MASTER_PLAN.md](MASTER_PLAN.md).
