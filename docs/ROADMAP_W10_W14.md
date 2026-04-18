# Flux Board W10-W17 Roadmap

Historical note:
- this file keeps the old `ROADMAP_W10_W14.md` path for link stability, but it now carries the formal `W10-W17` roadmap plus the current `W18` boundary decision

## Purpose
- This roadmap continues from the delivered `W0-W9` baseline, but it explicitly stays inside the product's real target: a high-quality single-user self-hosted board.
- `W10-W17` stay inside the same single-user self-hosted direction and are not reserved for multi-user, RBAC, or SSO work.
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
6. `W15` Hosted Release Operations
7. `W16` Backup and Restore Drills
8. `W17` Product Polish and Mobile Depth

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

## W15 Hosted Release Operations
### Goal
- Make tagged releases feel production-ready for hosted operators instead of being "works locally" artifacts.

### Scope
- GHCR image publishing verification
- image versioning and rollback policy
- hosted deployment evidence capture
- repo-owned post-deploy verification artifacts

### Work Packages
- `W15-P1` release artifact policy
  - pin how tagged releases map to Docker image tags, root-binary artifacts, and changelog notes
  - keep rollback notes and version metadata aligned with the actual release contract
- `W15-P2` hosted deployment evidence
  - exercise at least one real Docker-hosted path end to end
  - record `/healthz`, `/readyz`, `/api/status`, `/status`, `/login` or `/setup`, `/board`, `/settings`, and `/legacy/`
- `W15-P3` operator verification lane
  - keep a repo-owned script for status-contract and hosted-runtime checks
  - save enough artifact output that another operator can review the hosted proof later

### Closure
- a tagged build can be deployed on a real hosted Docker path, validated through repo-owned checks, and rolled back with written evidence instead of memory

## W16 Backup and Restore Drills
### Goal
- Turn export/import and PostgreSQL backup guidance into repeatable operator drills that prove recovery safety.

### Scope
- import validation hardening
- JSON export rehearsal
- PostgreSQL restore rehearsal
- optional scheduled backup guidance

### Work Packages
- `W16-P1` drillable backup baseline
  - define the exact JSON export plus PostgreSQL dump artifacts an operator should keep
  - keep destructive imports behind an explicit "backup first" workflow
- `W16-P2` restore rehearsal
  - document a scratch restore drill that proves the latest backup can be opened safely
  - require post-restore verification through `/api/status`, `/login`, `/board`, and `/settings`
- `W16-P3` operator cadence
  - document when to rerun drills, what artifacts to retain, and what failures should block release or deploy promotion

### Closure
- operators can rehearse export, backup, restore, and post-restore verification from repo-owned docs without inventing ad hoc recovery steps

## W17 Product Polish and Mobile Depth
### Goal
- Remove the last rough edges from the single-user board experience after runtime, deploy, and backup work have stabilized.

### Scope
- empty states and microcopy polish
- richer mobile ergonomics and touch audit
- final UX cleanup after deployment/runtime work settles

### Work Packages
- `W17-P1` copy and state polish
  - remove stale "internal wave" language from residual user-facing corners
  - make empty, signed-out, and recovery states read as a finished product
- `W17-P2` mobile and touch audit
  - capture the remaining touch and narrow-width rough edges after hosted/runtime work lands
  - keep keyboard and fallback interaction quality intact while polishing touch behavior
- `W17-P3` final UX acceptance
  - rerun the existing browser smoke lanes against the polished UI
  - close the last operator-facing UX roughness that blocks a confident hosted recommendation

### Closure
- the single-user runtime feels polished on desktop and mobile without sacrificing the already-proven fallback, keyboard, and deployment flows

## W18 Boundary
- `W18` does not need a formal implementation wave yet.
- The actual planning gap was that `W15-W17` existed only as short blurbs; they now need the same goal/scope/work-package/closure treatment as earlier waves.
- Keep the acceptance rule simple: `W15-W17` should still close at `remote-closed` on their own exact heads.
- Only open `W18` if a real post-polish gap appears that does not fit `W15-W17`; do not use `W18` as a catch-all bucket for closure work that should belong to the wave being closed.
