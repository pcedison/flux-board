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

## Enterprise Extension Seams
- Current scope remains intentionally narrow: one bootstrap-created admin, one board domain, and one runtime-owned workspace. `W9/5-B` documents expansion seams only; it does not implement RBAC, SSO, or multi-workspace behavior.
- Identity seam:
  - keep `users.username` as the canonical local principal key for now
  - future OIDC/SAML identities should map into that local principal through a separate identity-link table such as `user_identities(provider, external_subject, username, created_at, updated_at)` instead of replacing the current primary key
  - `sessions` should continue to point at the local principal so logout, revocation, and audit behavior do not become provider-specific
- RBAC seam:
  - the current `users.role = 'admin'` column is a bootstrap compatibility field, not the final authorization model
  - future authorization should move into explicit role bindings or workspace membership rows, not into more string states on `users.role`
  - transport should authenticate and attach principal/workspace context; authorization decisions should live in service-layer policy checks so later enterprise rules do not spread across handlers
- Workspace seam:
  - future multi-workspace support should introduce first-class `workspaces` and `workspace_memberships` tables
  - board-owned rows should gain a `workspace_id` foreign key in a later migration instead of encoding tenant scope in task IDs, status values, or route-only conventions
  - once rows are workspace-scoped, ordering and lookup constraints must widen accordingly, for example `(workspace_id, status, sort_order)` for lane ordering
- Recommended migration order for later enterprise work:
  1. add a default workspace and backfill existing rows into it
  2. add workspace membership and external identity-link tables
  3. widen task/archive constraints and indexes to include `workspace_id`
  4. teach repositories and services to require explicit workspace scope on every board query
- Future enterprise slices should extend the current seams instead of replacing them wholesale:
  - `internal/service/auth` is the policy entrypoint for login/session/identity decisions
  - `internal/store/postgres` is the place to add workspace-aware queries and identity-link persistence
  - `internal/transport/http` should gain one canonical workspace-resolution path per request instead of ad hoc header or query parsing
- Non-goals for the current head:
  - no IdP callback flow
  - no SCIM or JIT provisioning
  - no claim that the current schema already provides tenant isolation

## Source Of Truth
Execution priority and rollout order are defined in [docs/MASTER_PLAN.md](MASTER_PLAN.md).
