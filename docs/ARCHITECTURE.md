# Architecture

## Product Scope
- Flux Board is a single-user board.
- One deployed instance serves one operator.
- The system is optimized for self-hosted local or cloud deployment, not shared multi-user collaboration.

## Runtime Shape
Flux Board runs as one Go process that:
- connects to PostgreSQL
- applies embedded SQL migrations at startup
- serves the JSON API
- serves the React runtime on `/`
- preserves the legacy rollback shell on `/legacy/`
- exposes `/healthz`, `/readyz`, and `/metrics`

## Packaging Shape
- Official release artifact: self-contained root binary built from `go build .`
- Official hosted path: repo-root `Dockerfile`
- Embedded assets in the root binary:
  - `migrations/*.sql`
  - `static/`
  - `web/dist`
- Historical note:
  - `cmd/flux-board` still exists as a modular entrypoint for the package layout, but the supported release contract is now the root build because it carries the embedded runtime assets

## Startup Flow
1. Load env configuration.
2. Connect to PostgreSQL.
3. Apply embedded migrations.
4. Start background cleanup loops.
5. Serve HTTP routes, API, health/readiness, and the embedded frontend.

## Auth Model
- There is one logical admin account: `admin`.
- `APP_PASSWORD` is bootstrap-only.
- If the admin does not exist yet:
  - a non-empty `APP_PASSWORD` can seed the first password at startup
  - otherwise the browser finishes setup at `/setup`
- After bootstrap, normal auth uses the stored bcrypt hash in PostgreSQL and DB-backed sessions.
- Password rotation and session revocation happen in `/settings`, not through env changes.

## Data Model
- Active tasks live in `tasks`.
- Archived tasks live in `archived_tasks`.
- Session data lives in `sessions`.
- Auth audit data lives in `auth_audit_logs`.
- Operator settings such as archive retention live in `app_settings`.

## Board Behavior
- Canonical lanes: `queued`, `active`, `done`
- Supported lifecycle:
  - create
  - edit
  - reorder
  - move between lanes
  - archive
  - restore
  - permanently delete archived cards
- Archive cleanup is background-only.
- Reading archived tasks never mutates data.

## Frontend Shape
- `web/` contains the React + TypeScript + Vite app.
- The runtime uses:
  - React Router
  - TanStack Query
  - componentized board surfaces under `web/src/components/board`
- Main user routes:
  - `/setup`
  - `/login`
  - `/board`
  - `/settings`
  - `/about`

## Quality Gates Embedded In The Repo
- Go verification scripts
- workflow lint via `actionlint`
- Go lint via `golangci-lint`
- frontend typecheck, lint, tests, and production build
- Playwright smoke flows for login, drag-and-drop, keyboard, and compatibility redirects
- release dry-run scripts

## Current Risks
- current exact-head remote CI proof still needs to be re-recorded after the single-user productization slice
- Docker/hosted verification is implemented in docs and CI config, but this local session did not execute it because Docker was unavailable
- backend tests around settings/import/export still need deeper coverage

## Deliberate Non-Goals
- multi-user accounts
- RBAC
- workspaces
- OIDC / SSO
- tenant isolation

Those ideas are intentionally outside the current product target and should not be reintroduced unless the product goal changes.
