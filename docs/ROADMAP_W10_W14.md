# Flux Board W10-W14 Roadmap

## Purpose
- This roadmap continues from the delivered `W0-W9` baseline, but it explicitly stays inside the product's real target: a high-quality single-user self-hosted board.
- `W10-W14` are no longer reserved for multi-user, RBAC, or SSO work.
- The planning rule is simple: optimize for a reliable single-user runtime before expanding scope that the product does not need.

## Product Guardrails
- One deployed instance serves one operator.
- No multi-user accounts, RBAC, workspaces, or OIDC are planned.
- Docker image and self-contained root binary are the only supported production contracts.
- Every wave after `W1` still closes only at `remote-closed`.

## Current Snapshot
- `W10-W13` now have local implementation work in progress on the current working tree.
- `W14` now has its first implementation slice locally: `/api/status`, `/status`, and Docker runtime smoke automation.
- Local proof already exists for:
  - frontend lint, tests, and production build
  - `go test ./...`
  - `golangci-lint`
  - `actionlint`
  - release dry run with the root self-contained binary
- Exact-head GitHub Actions proof and hosted-path proof still need to be re-recorded after these changes are pushed.

## Recommended Order
1. `W10` Build, CI, and Hosted Deploy Hardening
2. `W11` Single-User Security & Settings
3. `W12` Product UX Completion
4. `W13` Data Portability & Backup
5. `W14` Observability & Operability

## Shared Remote-Closed Standard
- Local verification:
  - `./scripts/verify-go.ps1` or `./scripts/verify-go.sh`
  - `./scripts/verify-web.ps1` or `./scripts/verify-web.sh`
  - `./scripts/verify-workflows.ps1` or `./scripts/verify-workflows.sh`
  - `golangci-lint run`
  - any wave-specific smoke or integration checks added in that wave
- Remote verification:
  - exact current head green in GitHub Actions
  - release and Docker parity checks green when the wave touches packaging or deployment
- Deployment verification:
  - at least one real Docker-hosted path exercised end to end
  - startup, `/healthz`, `/readyz`, `/login`, `/board`, and rollback notes all verified

## W10 Build, CI, and Hosted Deploy Hardening
### Goal
- Make Docker, release artifacts, CI, and local dry runs all depend on the same runtime assumptions.

### Scope
- self-contained root binary packaging
- Docker build verification
- CI lint and release parity
- hosted deployment templates and docs

### Work Packages
- `W10-P1` release parity
  - keep release workflows building `.` instead of `./cmd/flux-board`
  - require `web/dist` to exist before a release build runs
- `W10-P2` CI hardening
  - keep `actionlint`, `golangci-lint`, frontend lint, and release dry-run checks in CI
  - make Docker image builds part of the default verification lane
  - add Docker runtime smoke for both seeded-login and browser-bootstrap paths
- `W10-P3` hosted deployment docs
  - keep `docker-compose.yml`
  - keep Render template and Docker-first Railway/Zeabur guidance aligned with reality
  - publish the official Docker image from tagged releases

### Closure
- a fresh clone can build the Docker image and the root binary without hidden asset steps
- GitHub Actions proves the release artifact and Docker image both build from the same runtime assumptions

## W11 Single-User Security & Settings
### Goal
- Make the single-user auth and runtime controls feel complete without inventing unnecessary account complexity.

### Scope
- first-run setup versus daily sign-in
- password rotation
- session inspection and revocation
- archive retention settings

### Work Packages
- `W11-P1` bootstrap split
  - keep `APP_PASSWORD` bootstrap-only
  - keep `/setup` as the browser-owned first-run path when no password seed is provided
- `W11-P2` settings surface
  - keep password rotation and active session controls in `/settings`
  - keep retention policy editable without server restarts
- `W11-P3` validation and audit polish
  - strengthen API validation and auth-audit coverage for bootstrap, password change, and session revocation

### Closure
- a fresh instance can bootstrap once, sign in again later, rotate password, and revoke stale sessions without touching env vars

## W12 Product UX Completion
### Goal
- Remove the remaining "dev shell" feel from the board and make the main runtime read like a finished product.

### Scope
- auth-aware home route
- product wording
- full task lifecycle UI
- keyboard and search ergonomics

### Work Packages
- `W12-P1` route polish
  - `/` routes to `/setup`, `/login`, or `/board` based on runtime state
  - `/about` carries product language instead of wave language
- `W12-P2` board lifecycle
  - keep create, edit, move, archive, restore, and permanent delete in the canonical UI
  - keep search/filter and keyboard shortcuts stable
- `W12-P3` mobile and touch proof
  - keep the existing smoke lanes aligned with the final UX copy and controls

### Closure
- the React runtime no longer exposes internal wave or rollback terminology in normal user flows

## W13 Data Portability & Backup
### Goal
- Make a single-user deployment feel safe to operate long-term.

### Scope
- export
- import
- retention defaults
- operator-facing backup guidance

### Work Packages
- `W13-P1` export/import baseline
  - keep JSON export and replace-import support in `/settings`
  - validate payload shape before destructive import
- `W13-P2` retention safety
  - default archive retention to "keep forever"
  - keep cleanup out of read paths
- `W13-P3` operator guidance
  - document JSON export plus PostgreSQL backup/restore expectations

### Closure
- users can safely export, restore, and migrate a board without direct database surgery

## W14 Observability & Operability
### Goal
- Turn the current logs, health probes, metrics, and tracing seam into an operator-friendly package for single-user hosting.

### Scope
- startup diagnostics
- bounded metrics and log fields
- backup-health visibility
- deployment troubleshooting guidance

### Work Packages
- `W14-P1` startup diagnostics
  - surface DB reachability, bootstrap state, retention policy, and runtime ownership clearly
- `W14-P2` signal discipline
  - keep request, auth, and backup-related metrics bounded and predictable
- `W14-P3` runbook docs
  - document common failures such as bad `DATABASE_URL`, stale cookies, bad imports, and failed cleanup

### Closure
- operators can diagnose setup, auth, DB, and backup issues from repo-owned docs and signals without reverse-engineering the app

## Beyond W14
### W15 Hosted Release Operations
- Goal: make tagged releases feel production-ready for hosted operators.
- Scope:
  - GHCR image publishing verification
  - image versioning policy and rollback notes
  - hosted deployment evidence for at least one real cloud path

### W16 Backup and Restore Drills
- Goal: turn export/import and PostgreSQL backup guidance into repeatable operator drills.
- Scope:
  - import validation hardening
  - documented restore rehearsal
  - optional scheduled backup guidance

### W17 Product Polish and Mobile Depth
- Goal: remove the last rough edges from the single-user board experience.
- Scope:
  - empty states and microcopy polish
  - richer mobile ergonomics and touch audit
  - final UX cleanup after the deployment/runtime work settles
