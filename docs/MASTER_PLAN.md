# Flux Board 10-Wave Master Plan

## Purpose
- This document is the single source of truth for upgrading Flux Board from MVP to enterprise-grade quality with public-fork readiness.
- Scope includes: security, data integrity, Go modularization, React + TypeScript + Vite frontend rebuild, Trello-level interaction quality, RWD, accessibility, CI, release governance, and resumable execution records.
- Rule: every task must leave a progress record so work can resume cleanly after interruption.

## Execution Rules
- Work model: `Wave -> Epic -> Task -> Gate -> Log`.
- Priority rule: `security and correctness > architecture > UX > polish`.
- No wave may bypass unresolved blockers from earlier waves unless explicitly recorded as an exception.
- High-risk areas always require strict review: auth, sessions, schema, reorder logic, CI, release process.
- Any change that breaks touch, keyboard, or mobile core flows is at least `P1`.

## Progress Recording Protocol
- Every task update must append one record to the `Execution Log` section at the end of this file.
- Record format:
  - `Date`
  - `Wave / Epic / Task`
  - `Status`: `planned | in_progress | blocked | done`
  - `Action`
  - `Result`
  - `Next`
  - `Risk / Blocker`
- Logging is append-only; do not overwrite earlier records except to fix obvious factual errors.
- When interrupted, resume by reading:
  1. `Wave Status Board`
  2. latest `Execution Log` entry
  3. unfinished task's `Next` field

## Wave Status Board
| Wave | Name | Owner | Status | Gate |
|---|---|---|---|---|
| W0 | Baseline Audit | Main agent | done | Risk map approved |
| W1 | Public Fork Baseline | Main agent | done | New contributor can boot project |
| W2 | CI and Reproducibility | Main agent | done | CI stable on clean env |
| W3 | Server Security Hardening | Main agent | done | No obvious public-deploy security gaps |
| W4 | Auth and Session Redesign | Main agent | done | Shared-password model retired for the current single-admin baseline |
| W5 | Schema and Data Integrity | Main agent | done | Migrations and reorder correctness verified |
| W6 | Go Modularization | Main agent | in_progress | Core logic testable and layered |
| W7 | Frontend Foundation | Main agent | in_progress | New React frontend builds and talks to API |
| W8 | Trello-grade UX, RWD, A11y | Main agent | in_progress | Mouse/touch/keyboard all pass core flows |
| W9 | Quality Gates, Release, Enterprise Hooks | Main agent | in_progress | Public-fork release ready |

## W0 Baseline Audit
- Goal: freeze the real MVP state and identify blockers before implementation.
- Epics: repo inventory, tracked vs untracked asset review, risk classification, architecture snapshot.
- Tasks: inspect current files, list P0/P1/P2 issues, capture current API/frontend behavior, define non-negotiable blockers.
- Gate: known blockers, hidden dependencies, and current architecture are documented.
- Parallel lanes: architecture review, security review, frontend review.
- Current status: done.
- Current gaps: none for W0 itself; its output remains the baseline for W1-W4.
- Corrected gate checklist:
  - repo inventory exists
  - P0/P1/P2 risks are documented
  - current architecture and blockers are documented
  - later waves reference W0 findings instead of replacing them

## W1 Public Fork Baseline
- Goal: make the repo understandable and bootable by strangers.
- Epics: repo cleanup, core docs, environment docs, contributor workflow.
- Tasks: add `README`, `LICENSE`, `SECURITY`, `CONTRIBUTING`, `ARCHITECTURE`, `.env.example`; clean root noise; document local startup and deployment basics.
- Gate: a new contributor can clone, configure, and run the project without tribal knowledge.
- Parallel lanes: docs writing, root cleanup, deploy instructions.
- Current status: done for the current wave scope.
- Current gaps:
  - no remaining W1 blockers for the current wave scope
- Corrected gate checklist:
  - `README`, `LICENSE`, `SECURITY`, `CONTRIBUTING`, `.env.example`, `docs/ARCHITECTURE.md`, and `docs/MASTER_PLAN.md` are present and aligned with reality
  - root contains only intentional source, tooling, or ignored build/test artifacts
  - onboarding steps reference only repo-owned assets
  - a fresh clone can configure env, run verification, and understand current limitations
  - tracked-history proof and clean-clone proof exist

## W2 CI and Reproducibility
- Goal: create a stable, repeatable quality baseline.
- Epics: GitHub Actions, local scripts, smoke validation, test/lint gates.
- Tasks: add Go build/test/vet, frontend build/typecheck/lint, basic smoke checks, deterministic setup steps.
- Gate: CI is green on a clean environment and failures are diagnosable.
- Parallel lanes: CI workflow, local dev scripts, smoke test setup.
- Current status: done for the current embedded-frontend scope.
- Current gaps:
  - broader frontend build/typecheck/lint remains deferred until the React/Vite waves
- Corrected gate checklist:
  - `go test ./...`, `go vet ./...`, and `go build ./...` run in CI and locally
  - local verification scripts are repo-owned and documented
  - Node dependency installation for smoke tooling is pinned and repeatable
  - browser smoke for the current embedded frontend is repo-owned, reproducible, and not hard-coded to production
  - an observed green GitHub Actions run exists for the tracked workflow
  - `README` and `MASTER_PLAN` only claim the exact reproducibility that actually exists

## W3 Server Security Hardening
- Goal: harden the public-facing Go service.
- Epics: explicit `http.Server`, security middleware, input limits, transport and header protections.
- Tasks: add timeouts, graceful shutdown, request size limits, strict JSON decoding, security headers, rate limiting, unified error envelopes.
- Gate: no bare `ListenAndServe`, no unlimited request bodies, no missing baseline security headers.
- Parallel lanes: HTTP middleware, validation layer, security verification.
- Current status: done for the current wave scope.
- Current gaps:
  - stronger distributed abuse control is deferred beyond this wave
- Corrected gate checklist:
  - explicit `http.Server`, timeouts, and graceful shutdown are in place
  - mutating endpoints enforce body limits and strict JSON decoding
  - baseline security headers are applied to relevant responses
  - abuse control has a documented trust boundary
  - automated verification covers core hardening paths

## W4 Auth and Session Redesign
- Goal: replace the MVP shared-password model with a safer baseline.
- Epics: users model, password hashing, session persistence, login protection, audit logging.
- Tasks: design `users` and `sessions`, implement secure login/logout/me, add throttling/lockout, add session revocation and audit trail.
- Gate: shared-password-only production auth is removed as the default path.
- Parallel lanes: auth domain, session store, audit design.
- Current status: done for the current single-admin baseline scope.
- Current gaps:
  - multi-user auth, username-based login, richer session revocation, and OIDC remain deferred to later waves
- Corrected gate checklist:
  - `APP_PASSWORD` is bootstrap-only or otherwise no longer the live shared production secret
  - `login -> session cookie -> /api/auth/me -> logout -> post-logout 401` is verified
  - session expiration, logout invalidation, and cleanup are observable and tested for the current scope
  - auth errors distinguish infra failure from bad credentials
  - auth audit logging exists for the current wave scope
  - protected-route session store failures return `500` instead of being misreported as `401`
  - docs and architecture notes match the implemented auth/session model

## W5 Schema and Data Integrity
- Goal: make schema evolution and ordering logic correct under concurrency.
- Epics: migrations, normalized schema, constraints/indexes, transactional reorder and archive flow.
- Tasks: add versioned migrations, enforce FK/CHECK/indexes, move reorder to transactional batch updates, verify archive/restore correctness.
- Gate: schema changes are reproducible and board ordering remains correct after reload and concurrent writes.
- Parallel lanes: migration setup, DB constraints, reorder API.
- Current status: done for the current single-board scope.
- Current gaps:
  - broader multi-board/domain normalization remains deferred to later waves
- Corrected gate checklist:
  - a versioned migration baseline exists and can initialize a fresh database
  - migration history is recorded in the database
  - task/archive schema constraints exist for the current single-board scope
  - reorder correctness is handled by a dedicated transactional endpoint
  - archive/restore keeps lane ordering stable for the current scope and is covered by integration tests

## W6 Go Modularization
- Goal: turn the backend into a maintainable, testable layered service.
- Epics: `cmd/` entrypoint, config, handlers, services, repositories, domain errors.
- Tasks: shrink `main.go` to assembly, separate HTTP/service/repo layers, extract pure logic, add dependency boundaries and test seams.
- Gate: core logic is no longer trapped in `main.go` and can be unit-tested.
- Parallel lanes: handler split, service split, repo abstraction.
- Current status: in_progress.
- Current gaps:
  - config loading, mux/server assembly, auth/session handlers, auth cookies/context/runtime helpers, auth orchestration service, task/archive handlers, task reorder persistence, task mutation validation/service flow, and pure task validation rules are now extracted into dedicated files, and background cleanup loops now respect cancellation, but deeper services and most domain rules still live in the `main` package
  - `cmd/flux-board` and deeper layer splits remain for later W6 slices
- Corrected gate checklist:
  - config loading is no longer embedded directly in startup logic
  - mux/server assembly is separated from the rest of the business logic
  - auth/session and task/archive HTTP handlers are no longer embedded in `main.go`
  - auth cookies, context helpers, and login-throttle/runtime helpers are no longer concentrated in `auth_http.go`
  - task/archive CRUD persistence now has an explicit repository seam
  - task mutation validation and reorder preconditions now have a dedicated service seam
  - pure task validation and ID normalization are now separated from transport code
  - later W6 work must still extract deeper services, repositories, and pure domain rules

## W7 Frontend Foundation
- Goal: replace the monolithic HTML with a modern, maintainable frontend base.
- Epics: `web/` scaffold, routing, API client, query layer, design tokens, shell layout.
- Tasks: create React + TypeScript + Vite app, add React Router, TanStack Query, app shell, tokenized style system, auth guard and base pages.
- Gate: new frontend can build, typecheck, and communicate with the Go API.
- Parallel lanes: design system, API layer, shell and routing.
- Current status: in_progress.
- Current gaps:
  - the new `web/` app now builds, typechecks, runs a small Vitest + Testing Library baseline, routes, proxies `/api`, reads the live Go API through React Query, has auth-aware `/login` plus guarded `/board` routes, exercises create/move/archive/restore mutations in the isolated shell, and can now be served by Go on `/next/` after `web/dist` is built
  - the isolated board now keeps mutation ownership scoped to the active card/form/archive row and restores focus after create/move success, but `/next/` remains a preview route rather than the production runtime owner
  - full production runtime ownership and replacement of the embedded frontend remain deferred to later W7/W8 slices
- Corrected gate checklist:
  - `web/` has a tracked React + TypeScript + Vite scaffold
  - routing, typed API reads, typed task mutations, and a query layer exist for the current isolated-shell scope
  - the scaffold has a responsive shell and can build/typecheck/test in CI and locally
  - later W7/W8 work must still add runtime integration with the existing app, drag-and-drop enhancement, and broader UX polish

## W8 Trello-grade UX, RWD, and Accessibility
- Goal: deliver rich board interactions that work on desktop, tablet, and mobile.
- Epics: board/list/card UI, `dnd-kit`, optimistic update, explicit move controls, mobile-first layouts, accessibility.
- Tasks: implement CRUD UI, drag-and-drop as progressive enhancement, touch fallback, keyboard move controls, dialog/menu focus management, 44px targets, browser matrix checks.
- Gate: core board movement works via mouse, touch, and keyboard; mobile and tablet are first-class; drag-and-drop is not the only path.
- Parallel lanes: board UI, drag/drop, RWD, a11y verification.
- Current status: in_progress.
- Current gaps:
  - the isolated React board now has non-drag create/move/archive/restore plus lane-local move up/down fallback, action-scoped pending ownership, focus continuity for repeated board work, field-level validation, live status feedback, and list-order semantics for assistive tech, while the Go-served `/next/` preview route gives W8 its first real runtime foothold
  - drag-and-drop, keyboard-first reordering polish, full mobile-first layout work, and promotion of `/next/` from preview to production runtime owner still remain
- Corrected gate checklist:
  - a non-drag movement path now exists in the isolated React board
  - current mutation controls are button-based and work without hover-only affordances
  - lane-local fallback now exposes order semantics to assistive technology
  - later W8 work must still add drag-and-drop as progressive enhancement, deeper accessibility polish, and runtime ownership

## W9 Quality Gates, Release, and Enterprise Hooks
- Goal: make the project release-ready and extensible toward enterprise features.
- Epics: automated tests, browser matrix, release flow, rollback docs, observability, enterprise extension seams.
- Tasks: add unit/integration/E2E coverage, verify Chromium/Firefox/WebKit, define release/rollback process, add health/metrics/logging, reserve extension points for RBAC/SSO/workspaces.
- Gate: project is safe to publish, testable in CI, and ready for controlled future enterprise expansion.
- Parallel lanes: QA, release engineering, observability, enterprise design.
- Current status: in_progress.
- Current gaps:
  - CI quality gates now cover richer browser smoke for login/create/archive/restore, repo-owned Go verification, local Windows race proof, `web/` build/typecheck/test, repo-owned smoke orchestration, a Go-served `/next/` preview smoke path, minimal unauthenticated health/readiness probes, a minimal request-id/access-log plus auth-audit correlation baseline, and a first release dry-run / rollback baseline, but browser matrix and richer observability still remain open
  - the workflow now opts into GitHub's Node 24 JavaScript action runtime pilot, shares a repo-owned smoke orchestration path with local verification, exposes a manual release dry-run path, and adds a dedicated `/next/` preview smoke lane while the broader browser matrix plus observability work still remain
- Corrected gate checklist:
  - CI runs repo-owned Go verification, race detection, `web/` scaffold build/typecheck/test, and browser smoke
  - local Windows race verification exists in a repo-owned script
  - a manual release dry-run path now builds a checksumed artifact and reuses it for smoke verification
  - later W9 work must still add broader browser and observability gates beyond the current request-id/access-log baseline

## Standard Gates
- Security gate: no known P0 security defect; no shared-password production default; request and session protections active.
- Integrity gate: migrations exist; reorder is transaction-safe; archive/restore and CRUD pass regression checks.
- Engineering gate: layered code, tests for critical paths, reproducible CI, documented startup and deployment.
- UX gate: desktop/tablet/mobile pass; touch and keyboard pass; no hover-only critical action.
- Fork gate: public docs are sufficient, no hidden author-only knowledge, no hard-coded private assumptions.

## Work Package Index
### W0
- `W0-P1` Repo inventory: scan tracked/untracked files, map entrypoints, identify temp assets. Done: repo map is complete. Parallel: with W0-P2/W0-P3.
- `W0-P2` Risk map: classify P0/P1/P2 across auth, data, security, UI, deploy. Done: prioritized risk list approved. Parallel: security review.
- `W0-P3` Architecture snapshot: capture backend/frontend flow and coupling points. Done: current system snapshot documented. Parallel: informs later modularization.
- `W0-P4` Blocker baseline: define non-negotiable blockers for next waves. Done: exit blockers explicit. Parallel: can draft during inventory.
### W1
 - `W1-P1` Core docs: status `done`. Governance docs are tracked and aligned with the current W0-W4 reality. Parallel: with W1-P2/W1-P3.
 - `W1-P2` Build docs: status `done`. Env, architecture, and deployment docs reflect the current bootstrap/runtime model. Parallel: depends on W0 snapshot.
 - `W1-P3` Root cleanup: status `done`. Repo-owned diagnostics and tooling are intentional, tracked, and `.env.example` is publishable. Parallel: safe with docs work.
 - `W1-P4` Local onboarding: status `done`. A fresh clone can run repo-owned verification and follow the documented setup without tribal knowledge. Parallel: after env assumptions settle.
### W2
 - `W2-P1` Backend CI: add Go build/test/vet workflow. Status `done`: workflow exists and has an observed green Actions run. Parallel: with W2-P2.
 - `W2-P2` Frontend reproducibility: status `done` for the embedded frontend scope. The tracked browser smoke path now runs in CI on a clean environment. Parallel: independent of backend runtime.
 - `W2-P3` Local scripts: status `done`. Backend verification exists, browser smoke is tracked, and Node installs are pinned and repeatable. Parallel: supports later E2E.
 - `W2-P4` Failure clarity: status `done` for the current scope. CI preserves smoke diagnostics, and smoke runtime is bounded so failures become actionable. Parallel: incremental improvement allowed.
### W3
- `W3-P1` HTTP hardening: replace bare server with explicit `http.Server`, timeouts, graceful shutdown. Done: no bare `ListenAndServe`. Parallel: foundation for W4.
- `W3-P2` Input limits: add body size caps and strict JSON decode. Done: mutating endpoints reject invalid or oversized payloads. Parallel: with validation layer.
- `W3-P3` Security headers: add baseline headers and `no-store` policy. Done: responses are safer for public deploy. Parallel: middleware-only task.
- `W3-P4` Abuse control: status `done`. Login throttling, unified error responses, trust-boundary documentation, and handler-level verification now exist for the current wave scope. Parallel: with validation work.
### W4
 - `W4-P1` Auth baseline replacement: status `done` for the current scope. `users` and `sessions` exist, and bootstrap seeding is no longer the live reset path. Parallel: with session store design.
 - `W4-P2` Secure login flow: status `done` for the current single-admin baseline. `login/logout/me` exists with DB-backed sessions and clean-environment proof. Parallel: with frontend login UI.
 - `W4-P3` Brute-force defense: status `done` for the current scope. Throttling and auth audit logging exist, and infra failures are distinguished from bad credentials. Parallel: can reuse W3 middleware pieces.
 - `W4-P4` Session control: status `done` for the current scope. Expiry cleanup, logout invalidation, protected-route error handling, and DB-backed auth verification are all proven. Parallel: with test groundwork.
### W5
- `W5-P1` Formal migrations: status `done`. Versioned migrations, recorded checksums, and schema baseline validation now initialize a fresh database repeatably. Parallel: with W6 repo work.
- `W5-P2` Normalized model: status `done` for the current single-board scope. Stronger task/archive constraints and stored archived sort order now protect the current schema. Parallel: broader domain normalization remains deferred.
- `W5-P3` Reorder correctness: status `done` for the current scope. Transactional reorder endpoint, lane advisory locks, and `MAX()+1` race removal are now in place. Parallel: later W8 UI work can build on this API.
- `W5-P4` Archive correctness: status `done` for the current scope. Archive/restore now preserves lane position semantics and is covered by integration plus browser smoke. Parallel: future retention/reporting work remains separate.
### W6
- `W6-P1` Assembly-only main: status `in_progress`. Config loading and mux/server assembly are extracted, and reorder logic has begun moving into dedicated files, but `cmd/` entrypoint and deeper startup isolation still remain. Parallel: pure structural move first.
- `W6-P2` Layer split: status `in_progress`. Auth/session plus task/archive/reorder HTTP boundaries, task repository seams, a first task service seam for validation and reorder preconditions, and a dedicated auth orchestration/persistence/audit service file are now extracted into dedicated files, but broader service/repo separation still remains. Parallel: coordinate with W5 query changes.
- `W6-P3` Pure rules and domain errors: status `in_progress`. Task payload validation, ID normalization, and reorder preconditions now sit behind dedicated non-HTTP helpers, but broader domain rules are still embedded in the main package. Parallel: good subagent slice.
- `W6-P4` Test seams: status `in_progress`. Route wiring, probe coverage, task handler/repository seam coverage, task service tests, auth service tests, and cancellable background-loop coverage now exist, but more explicit seams and layer-level tests remain for later W6 work. Parallel: tie into W2 CI.
### W7
- `W7-P1` Frontend scaffold: status `done` for the current scope. A tracked React + TypeScript + Vite scaffold now exists under `web/`. Parallel: can overlap late W6.
- `W7-P2` Data layer: status `in_progress`. Typed API reads, an auth-session hook, scoped login/task mutations, a React Query snapshot hook, `401`-aware auth reset behavior, a frontend unit-test baseline, and a Go-served `/next/` preview runtime path now exist in the isolated shell. Parallel: with W7-P3.
- `W7-P3` Design system: status `in_progress`. Tokenized CSS variables and a responsive shell exist, but the full board design system remains later work. Parallel: with W7-P2.
- `W7-P4` Page skeletons: status `in_progress`. Overview, login, auth-aware shell navigation, explicit sign-out handling, guarded board snapshot routes, and a Go-served `/next/` preview route exist for the current isolated-shell scope, including explicit create/move/archive/restore actions and unit-test coverage, but full feature/runtime integration remains incomplete. Parallel: after W7-P1.
### W8
- `W8-P1` Core board UI: status `in_progress`. The isolated React board now renders create/move/archive/restore controls around board/list/card layout, but richer interaction polish still remains. Parallel: with W8-P2.
- `W8-P2` Drag and reorder: status `planned`. `dnd-kit` work depends on later W5 reorder API. Parallel: depends on W5 reorder API.
- `W8-P3` Non-drag movement: status `in_progress`. Explicit create/move/archive/restore controls plus lane-local move-up/move-down fallback now exist in the isolated React board so drag-and-drop is no longer the only future path. Parallel: with W8-P2.
- `W8-P4` RWD and a11y: status `in_progress`. The isolated board now has 44px targets, field-level validation, focus recovery, live status feedback, lane-order semantics for assistive tech, a more explicit signed-in/signed-out shell boundary, and a Go-served preview path for real-browser validation, but deeper mobile-first layout and keyboard interaction work still remain. Parallel: with W8-P1.
### W9
- `W9-P1` Test gates: status `in_progress`. CI now includes repo-owned Go verification, Windows-local race proof, `web/` scaffold build/typecheck/test, richer browser smoke for login/create/archive/restore through the shared verify-smoke scripts, and a dedicated `/next/` preview smoke path; this slice begins the browser matrix with `chromium` plus `firefox`, while broader frontend/E2E and deeper matrix work still remain. Parallel: with W9-P2.
- `W9-P2` CI and release flow: status `in_progress`. Workflow hardening now includes the Node 24 JavaScript action runtime pilot, a split verify/smoke job layout, repo-owned Go/web/smoke verification scripts as the CI source of truth, and a dedicated `/next/` preview verification wrapper, but release governance remains for later waves. Parallel: partial dependency on W9-P1.
- `W9-P3` Observability: status `in_progress`. Minimal unauthenticated health/readiness probes now exist, but metrics and richer logging/observability beyond the current baseline remain open. Parallel: with W9-P2.
- `W9-P4` Enterprise extension seams: status `planned`. RBAC/SSO/workspace seams are deferred. Parallel: after W7-W8 stabilize.

## Execution Log
- 2026-04-16 | W0 / Planning / Master Plan | done | Created condensed 10-wave master plan and resumable logging protocol | Master plan established in `docs/MASTER_PLAN.md` | Next: decompose W0 into executable epics/tasks and start baseline audit | Risk: none
- 2026-04-16 | W0-W9 / Planning / Work Package Decomposition | done | Spawned subagents to decompose all 10 waves into executable work packages and merged them into this document | `MASTER_PLAN.md` now contains wave-level packages, done criteria, and parallel notes for W0-W9 | Next: start W0 execution by converting W0-P1 through W0-P4 into active tasks and append progress as work begins | Risk: none
- 2026-04-16 | W0 / Repo inventory + risk map + architecture snapshot | done | Used subagents to inventory tracked/untracked assets, classify P0/P1/P2 risks, and capture current backend/frontend flow and blockers | W0 baseline is now documented enough to start governance work without hidden assumptions | Next: land W1 core docs, env baseline, and repo hygiene updates | Risk: package.json and test-login assets remain local diagnostics and are not yet formalized
- 2026-04-16 | W1 / Core docs + onboarding baseline | done | Added `README.md`, `CONTRIBUTING.md`, `SECURITY.md`, `.env.example`, `docs/ARCHITECTURE.md`, and `LICENSE`; extended `.gitignore` for local diagnostics | Public-fork baseline is materially improved and a fresh contributor now has startup and governance guidance | Next: start W2 with minimal reproducible CI and decide how to formalize local Playwright diagnostics | Risk: license assumed as MIT and may need owner confirmation
- 2026-04-16 | W2 / Backend CI baseline | done | Added `.github/workflows/ci.yml` with Go test, vet, and build checks | Minimal reproducible Go CI is now defined for pushes and pull requests | Next: expand W2 later with frontend pipeline once tracked frontend tooling exists | Risk: frontend CI remains pending until a tracked frontend toolchain is established
- 2026-04-16 | W2 / Local verification scripts | done | Added `scripts/verify-go.ps1` and `scripts/verify-go.sh`, updated `README.md` verification steps, and ignored local Go build artifacts | Backend verification now has a repeatable local smoke path that matches CI expectations more closely | Next: add frontend CI and broader smoke coverage once the tracked frontend toolchain exists | Risk: W2 is still incomplete because frontend reproducibility remains pending
- 2026-04-16 | W3 / W4 / Minimal landing | done | Hardened the Go server with explicit timeouts, graceful shutdown, request body limits, strict JSON decoding, security headers, and rate-limited login handling; replaced in-memory auth sessions with DB-backed users/sessions while keeping the same login API shape | Public-facing auth/session baseline is now safer without requiring major frontend changes | Next: return to W2 frontend tooling or proceed to W5 schema hardening when ready | Risk: enterprise auth, RBAC, and versioned migrations are intentionally deferred to later waves
- 2026-04-16 | W3 / W4 / Compile recovery + auth baseline verification | done | Verified the W3/W4 landing after compile blockers, consolidated session lookup into shared helpers, and added `main_test.go` coverage for strict JSON decode, task payload validation, login throttling, and request client identification | `go test ./...`, `go vet ./...`, and `go build ./...` all pass; W3 gate is satisfied; W4 is improved but remains in progress because production auth still relies on a single bootstrap credential | Next: finish W2 frontend CI/local scripts or move into W5 migrations and reorder correctness with the current baseline locked | Risk: W4 is not complete yet; audit logging, multi-user auth, and public-fork-safe credential strategy are still open
- 2026-04-16 | W0-W4 / Review reset / Status correction | done | Re-audited W0-W4 with subagents, corrected wave statuses, downgraded overstated work packages, and replaced optimistic done claims with evidence-based gaps and gate checklists | `MASTER_PLAN.md` is now aligned with the real state: W0 done, W1/W2/W4 in progress, W3 in review | Next: complete W1 root hygiene and W2 reproducibility before attempting to close W3/W4 | Risk: some repo-owned tooling and docs still need to be brought into full alignment with the corrected plan
- 2026-04-16 | W1-W2 / Smoke tooling formalization | done | Converted the old root-level Playwright diagnosis script into repo-owned smoke tooling under `tests/e2e`, made `package.json` intentional, and removed the old production-targeted root script | W1 root hygiene and W2 smoke coverage are improved, but this is still not enough to close either wave because the smoke path is local-only and CI integration remains open | Next: verify the new smoke script syntax, align docs, and decide whether W2 will stop at local smoke or add CI execution for the embedded frontend | Risk: smoke currently requires a running local app and valid credentials, and it is not yet part of automated CI
- 2026-04-16 | W1-W2 / Documentation alignment after smoke formalization | done | Updated `README.md`, `CONTRIBUTING.md`, and the corrected W1/W2 sections of `MASTER_PLAN.md` to reflect the repo-owned smoke path and the still-open CI gaps | Wave tracking and contributor guidance now match the current local tooling more closely | Next: decide whether W1 can be closed after a final root-hygiene pass, then finish W2 by choosing either local-smoke-only scope or CI execution for the embedded frontend | Risk: package/tooling assets are still only present in the worktree until they are committed, and W2 still lacks CI execution for browser smoke
- 2026-04-16 | W2 / CI smoke workflow expansion | done | Extended `.github/workflows/ci.yml` to start PostgreSQL, run the Go app, install Playwright Chromium, execute `npm run smoke:login`, and retain server/test artifacts on failure | W2 now has a configured CI path for the current embedded frontend smoke, but the wave remains open until that workflow is actually observed passing after these repo-owned assets are committed | Next: verify the smoke contract remains aligned with the current auth/session API, then decide whether W2 can move to in-review or still needs more reproducibility work | Risk: this thread has not observed a live GitHub Actions run yet, so CI success is configured but not proven
- 2026-04-16 | W3-W4 / Verification hardening and auth error split | done | Added handler-level tests for `401`, `413`, `429`, security headers, and login error handling; introduced a password-verifier seam so auth infrastructure failures return `500` instead of being misreported as bad credentials; documented the login throttling trust boundary in `SECURITY.md` | W3 now meets its corrected narrow gate with evidence, and W4 no longer conflates DB auth failures with invalid passwords | Next: keep W4 in progress by tackling audit logging and end-to-end auth/session verification without expanding into W5 | Risk: W4 still depends on a bootstrap credential and does not yet have audit logging or full auth integration coverage
- 2026-04-16 | W4 / Bootstrap-only auth seeding | done | Changed bootstrap admin initialization so `APP_PASSWORD` seeds the first admin on initial setup instead of resetting live credentials on every startup; aligned docs and plan language with the new behavior | W4 is closer to its gate because the env secret is no longer the live shared production password path | Next: add audit logging and prove auth/session flow end to end before considering W4 complete | Risk: auth still centers on one admin account and lacks audit logging plus full integration proof
- 2026-04-16 | W4 / Audit logging and handler-level auth flow proof | done | Added auth audit logging storage/hooks and handler-level tests covering login success, blocked login, verifier failure, logout invalidation, invalid session handling, and the `login -> /api/auth/me -> logout -> 401` path; updated docs to reflect the improved state | W4 now has materially stronger evidence and observability, but still remains in progress because auth is single-admin and full DB-backed integration coverage is not yet present | Next: continue W4 with either real DB-backed auth integration tests or another narrow improvement that does not expand scope into W5 | Risk: W4 still is not a public-fork-final auth model
- 2026-04-16 | W2-W4 / Local integration smoke attempt | blocked | Attempted to run the repo-owned browser smoke against a temporary local PostgreSQL-backed app instance to raise evidence beyond static verification | The repo-side smoke path is syntactically valid and CI is configured, but this local environment could not start PostgreSQL because the Docker daemon was unavailable | Next: either observe the GitHub Actions smoke run after commit or rerun local integration once Docker Desktop is running | Risk: local end-to-end evidence is still pending because environment runtime support is unavailable in this session
- 2026-04-16 | W9 / Local Windows race enablement | done | Installed MSYS2 UCRT64 GCC, removed a polluted user npm config that forced `os=linux`, and added repo-owned Windows race verification scripts | Local Windows development now has first-class `go test -race` support instead of relying only on Linux CI | Next: keep Linux CI as the cross-check while using the repo race script locally | Risk: developers still need the documented MSYS2 toolchain on Windows machines
- 2026-04-16 | W6 / Auth-task HTTP split and route proof | done | Moved auth/session helpers plus task/archive HTTP handlers out of `main.go`, then added direct `newMux` route wiring coverage | `main.go` is slimmer and the current handler boundary is safer to extend without changing behavior | Next: later W6 slices should extract service/repository seams instead of piling logic back into handlers | Risk: SQL and domain rules still live in the `main` package
- 2026-04-16 | W7 / React foundation read-model shell | done | Rebuilt the isolated `web/` scaffold around React Router, TanStack Query, a typed API client, responsive app shell, and read-only overview/board routes | W7 is now active with a maintainable read-model frontend that builds, typechecks, and talks to the real Go API without disturbing the embedded frontend | Next: keep the scaffold read-only until later W8 work adds mutation flows and auth-owned pages | Risk: the new frontend is not deployed and does not yet own login or board mutations
- 2026-04-16 | W9 / Script-backed CI parity and manual smoke | done | Switched CI to use repo-owned Go verification scripts with dynamic Go package discovery, then ran local Go, race, web, and Docker-backed browser smoke verification against a temporary PostgreSQL instance | Quality gates now cover the new modularization slice and read-only frontend foundation with real end-to-end evidence | Next: broaden browser matrix and release/observability work in later W9 slices | Risk: broader frontend unit tests and multi-browser coverage are still pending
- 2026-04-16 | W1-W2 / Deployment and deterministic Node install tightening | done | Added `docs/DEPLOYMENT.md`, linked it from `README.md`, switched smoke install guidance to `npm ci`, updated CI to use `npm ci`, and refreshed `package-lock.json` so browser tooling installs are pinned | W1 deployment guidance and W2 dependency reproducibility are materially tighter, though both waves still remain open pending tracked-history proof and an observed passing CI run | Next: keep W1/W2 open until the repo-owned files are committed and the CI smoke is observed passing | Risk: current evidence is still limited by the absence of a fresh-clone proof and a live GitHub Actions success in this thread
- 2026-04-16 | W4 / Real DB-backed auth integration test added | done | Added `main_integration_test.go`, which skips without `DATABASE_URL` but exercises login failure, login success, `/api/auth/me`, logout, post-logout `401`, and auth audit log writes against a real PostgreSQL-backed app when a DB-capable environment exists | W4 now has repo-owned real-database integration coverage ready for CI/DB-capable runs, while local no-DB environments still remain stable because the test skips cleanly | Next: observe or rerun this test in a DB-capable environment before treating W4 as closeable | Risk: the integration test exists, but this thread has not yet observed it execute against a live database
- 2026-04-16 | W1-W2 / Tracked-history and fresh-clone proof | done | Created branch `codex/w1-w2-w4-closeout`, committed the W1/W2/W4 baseline, and validated a clean clone by checking `.env.example`, running `./scripts/verify-go.ps1`, running `npm ci`, and syntax-checking the repo-owned smoke script | W1 now has tracked-history proof and a clean-clone reproducibility record for the current scope | Next: observe GitHub Actions before closing W2 | Risk: clean-clone runtime app startup still depends on a database-capable environment
- 2026-04-16 | W2 / Smoke failure bounded for diagnosis | done | Added request-level and overall smoke timeouts plus workflow-level timeout so browser smoke can no longer hang indefinitely in CI | W2 failure mode is now actionable instead of silent hanging | Next: fix the root cause behind the first smoke 401 and rerun CI | Risk: none
- 2026-04-16 | W2-W4 / CI contamination fix and clean-environment proof | done | Removed the duplicate DB-backed auth test that polluted the default CI schema, reran the PR workflow, and observed a green GitHub Actions run for `go test`, `go vet`, `go build`, app startup, and browser smoke on run `24494722677` | W2 is now proven on a clean environment, and W4 has observed DB-backed auth/session evidence plus browser smoke proof for the current single-admin baseline | Next: keep future auth evolution and multi-user work in later waves, not by reopening W4 scope silently | Risk: later waves still need multi-user/OIDC and richer session controls
- 2026-04-16 | W0-W4 / Final verification pass | done | Re-ran local backend verification, rechecked latest CI on head `d64348f`, and confirmed the latest green workflow run `24494839272` still covers clean-environment boot, DB-backed auth tests, and browser smoke; aligned deployment docs with the tracked smoke credential requirement | W0-W4 are now re-verified and closed for their current documented scope | Next: begin W5 migration baseline and W6 startup extraction without reopening earlier waves | Risk: W1/W4 completion is scoped to the current single-admin + embedded-frontend baseline, not later-wave enterprise targets
- 2026-04-16 | W5-W6 / First execution slice | in_progress | Started W5 migration baseline by introducing versioned SQL migrations and migration history tracking; started W6 bootstrap extraction by moving config loading into `internal/config` and moving mux/server assembly into dedicated startup files | W5 and W6 are now active with a low-risk first slice that preserves current behavior while creating room for deeper schema and modularization work | Next: validate migration history in DB-capable tests, then continue with reorder correctness and deeper layer extraction | Risk: `main.go` still owns handlers and SQL, and W5 has not yet addressed reorder races or stronger constraints
- 2026-04-16 | W5-W6 / First slice validation | done | Fixed CI env isolation in the new config tests, reran local verification, and observed green GitHub Actions run `24495401377` for the W5/W6 startup+migration slice | The first W5/W6 slice now has local and clean-environment CI proof without reopening W0-W4 | Next: move W5 into reorder correctness and stronger schema guarantees, and move W6 into deeper handler/service/repo extraction | Risk: migration baseline still needs future down/rollback strategy and reorder work remains open
- 2026-04-16 | W9 / First quality-gate tightening slice | in_progress | Tightened CI toward W9 by switching cache-busted Go tests, adding race detection in CI, and moving smoke tooling to Node 22 while keeping local verification lightweight | W9 is now active without blocking W5/W6, and CI quality gates are stronger for upcoming backend/frontend work | Next: verify the tightened workflow on GitHub Actions, then decide whether to add release/observability or browser-matrix work next | Risk: GitHub-hosted JS action runtime deprecation warnings still remain and need a later W9-specific pass
- 2026-04-16 | W5 / Transactional reorder and archive integrity | done | Added a dedicated reorder endpoint, lane advisory locks, stronger task/archive constraints, archived sort-order retention, and integration coverage for reorder plus archive/restore semantics; updated the embedded frontend to call the new reorder path and stop relying on client-written `sort_order` | W5 is now complete for the current single-board scope, and the old `MAX(sort_order)+1` / single-task reorder drift path is retired | Next: continue W6 modularization while keeping W7 deferred until the backend seam is calmer | Risk: future concurrent stress tests can deepen proof, but no blocking correctness issue remains for the current scope
- 2026-04-16 | W5-W9 / Local Docker-backed verification | done | Ran `go test -count=1 ./...` against a temporary PostgreSQL container, expanded the Playwright smoke from auth-only to login/create/archive/restore/logout, and verified the upgraded smoke path end to end against a Docker-backed local app | The current W5 changes now have local DB-backed integration proof and browser smoke proof, not just compile-time or unit-level evidence | Next: push this slice and observe the remote GitHub Actions run with the richer smoke coverage | Risk: local Windows could not run `go test -race` because CGO is disabled, so final race proof still depends on Linux CI
- 2026-04-16 | W9 / Node 24 JavaScript action runtime pilot | in_progress | Updated the GitHub Actions workflow to opt into GitHub's Node 24 JavaScript action runtime pilot while keeping the existing backend and smoke pipeline intact | W9 now has a concrete path to clear the hosted Node 20 deprecation warning without reopening W5/W6 code paths | Next: observe a green remote Actions run with the pilot enabled and decide whether any action versions still need later upgrades | Risk: this slice needs an observed GitHub-hosted success run before it can be treated as fully proven
- 2026-04-16 | W9 / Remote CI proof for reorder + Node 24 pilot | done | Pushed commit `885f3eb` to PR `#1` and observed green GitHub Actions run `24496466310`, which covered Go tests, Linux race test, build, app startup, and the richer browser smoke under the Node 24 JavaScript action runtime pilot | W9 now has observed remote proof for the current reorder-integrity and smoke-expansion slice | Next: continue W6 structural extraction and later W7/W8 frontend rebuild work while leaving release/observability for later W9 passes | Risk: broader browser matrix, release flow, and observability remain open
- 2026-04-16 | W6 / Task repository seam for CRUD and archive flows | done | Added an explicit task repository seam for task/archive CRUD persistence and rewired handlers to call it instead of embedding SQL directly in the HTTP layer | W6 now has a cleaner boundary between HTTP shaping and persistence, making deeper service/repository extraction safer | Next: move reorder persistence and pure domain rules behind similar seams in later W6 slices | Risk: reorder orchestration and most domain rules still live in the `main` package
- 2026-04-16 | W7-W9 / Frontend unit-test baseline and verification expansion | done | Added Vitest + Testing Library to `web/`, covered the overview and board snapshot read-model routes, and expanded `verify-web` so local and CI verification now include frontend tests before build | W7 now has a credible frontend quality baseline, and W9 has stronger guardrails for future frontend work | Next: add auth-aware pages and mutation-path tests before beginning W8 interaction work | Risk: the new frontend still lacks deployed runtime ownership and browser-matrix coverage
- 2026-04-16 | W1-W9 / Public-fork hygiene audit after W6-W7 slice | done | Re-audited the repo for secrets, author-machine coupling, and document truthfulness after the latest modularization/frontend changes, then aligned the README with the current Node/Vite requirement and verification story | No new privacy or public-fork blockers were introduced by this slice, and the repo remains transparent about its current limits | Next: keep repeating this audit before each later wave that expands runtime ownership or deployment surface | Risk: Windows race tooling still assumes the documented MSYS2 path, which remains acceptable but platform-specific
- 2026-04-16 | W6 / Reorder repository seam | done | Moved reorder transaction orchestration behind `TaskRepository.ReorderTask`, added an invalid-anchor domain error, and covered the thinner handler mapping with repository-seam tests | W6 now keeps CRUD, archive, and reorder persistence out of the HTTP layer, reducing direct SQL in handlers | Next: keep shrinking `main` by extracting pure domain rules and deeper service seams | Risk: most domain logic still lives in the `main` package
- 2026-04-16 | W7 / Auth-aware routing slice | done | Added a guarded `/board` route, a lightweight `/login` page, an auth-session query hook, and route-level frontend tests while keeping the new shell read-only for board data | The new frontend now models authenticated vs unauthenticated flow without taking ownership of board mutations yet | Next: add board mutation architecture and runtime integration before W8 interaction work | Risk: the React shell is still isolated and not yet the production runtime owner
- 2026-04-16 | W9 / Probe contract and local verification | done | Added unauthenticated `/healthz` and `/readyz` handlers with explicit no-store headers, switched CI/deployment readiness checks to `/readyz`, added unit/integration probe coverage, and reran `verify-go`, `verify-go-race`, `verify-web`, plus Docker-backed local DB/browser smoke | Operability is now less coupled to auth semantics, and this W6/W7/W9 slice has local backend, frontend, race, integration, and browser proof | Next: observe the updated CI on GitHub, then continue deeper W6 structure work and later W9 observability/release slices | Risk: probes remain intentionally minimal and broader browser-matrix/release work is still open
- 2026-04-16 | W6 / Task mutation service seam | done | Added `task_service.go` so create/update/reorder validation and precondition checks now sit behind a task service instead of living only in handlers, then added service and HTTP-mapping tests to keep the transport contract stable | W6 now has a clearer HTTP -> service -> repository progression for task mutations without changing the live API surface | Next: keep extracting deeper domain rules and a future `cmd/` entrypoint without reopening handler correctness | Risk: the service seam still lives in the `main` package and broader domain extraction remains open
- 2026-04-16 | W7-W8 / Initial non-drag mutation path | done | Extended the isolated React board to create, move, archive, and restore tasks through typed API helpers plus React Query mutations, then added field-level validation, focus recovery, live status feedback, and frontend tests around the new controls | W7 now owns the first write-path in the isolated shell, and W8 is officially active because the future frontend no longer depends on drag-and-drop as its only movement model | Next: keep W8 focused on progressive enhancement, keyboard/touch polish, and eventual runtime integration instead of jumping straight to drag-and-drop | Risk: the React shell is still isolated from production runtime ownership, and full mobile-first/browser-matrix work remains open
- 2026-04-16 | W6-W8 / Service seam tightening and lane-local fallback | done | Added direct handler-to-service seam coverage for task creation, then expanded the isolated React board with lane-local move-up/move-down fallback, a global board feedback banner, and richer tests around non-drag mutations | W6 now has more direct evidence that handlers honor the new service seam, and W8 now covers lane-local ordering without requiring drag-and-drop | Next: continue W6 deeper domain extraction, and keep W8 focused on keyboard/touch polish plus future drag enhancement | Risk: production runtime ownership is still the embedded frontend, and broader browser-matrix work remains open
- 2026-04-16 | W6-W9 / ID normalization and repo-owned smoke orchestration | done | Moved task ID normalization for update/archive/restore/reorder/delete into the task service seam, then added `verify-smoke.ps1/sh` so local and CI smoke now share the same app-start/readiness/smoke/cleanup flow | W6 is less HTTP-coupled for task mutations, and W9 now has a tighter local/CI parity story for browser smoke | Next: keep W6 shrinking domain logic out of `main`, and continue W8/W9 toward keyboard polish, browser matrix, and release/observability work | Risk: production runtime ownership and broader multi-browser/release gates are still open
- 2026-04-16 | W6-W9 / Validation extraction, list semantics, and smoke parity proof | done | Extracted pure task validation into `task_validation.go`, added lane-order semantics plus reorder guidance to the isolated React board, fixed Windows `verify-smoke.ps1` readiness polling, and re-ran local Go/web/smoke verification against a Docker-backed PostgreSQL instance | W6 now has a cleaner pure-validation seam, W8 has a more screen-reader-friendly non-drag fallback, and W9 now has real local proof that the shared smoke orchestration works on Windows | Next: continue W6 domain extraction and push W8/W9 toward keyboard polish, multi-browser coverage, and release/observability work | Risk: runtime ownership is still split between the embedded frontend and the isolated React shell
- 2026-04-16 | W6-W9 / Auth service seam and first non-Chromium smoke gate | in_progress | Extracted auth/session persistence plus audit methods into a dedicated auth service file, parameterized Playwright smoke by browser, and split CI so base verification runs once before browser-specific smoke jobs run for `chromium` and `firefox` | W6 now has a cleaner auth transport boundary, and W9 has the first real browser-matrix expansion beyond Chromium while preserving repo-owned smoke parity | Next: run local verification, then observe CI to confirm the new split workflow and Firefox smoke remain green | Risk: Firefox may expose selector or timing assumptions that Chromium previously masked
- 2026-04-16 | W7-W9 / Session-owned shell hardening and local multi-browser proof | done | Added auth-aware shell navigation with explicit sign-out handling, centralized auth-query ownership helpers, reset auth state on `401` from protected board fetches/mutations, updated frontend tests, and re-ran local `chromium` plus `firefox` Docker-backed smoke against the shared verify-smoke flow | W7 now owns a more honest signed-in/signed-out runtime boundary, W8 has safer non-drag auth degradation, and W9 has observed local proof for the first non-Chromium browser gate before remote CI | Next: push this slice and observe the split GitHub Actions workflow so the new Firefox lane is proven remotely | Risk: runtime ownership is still split between the embedded frontend and the isolated React shell, and broader release/observability work remains open
- 2026-04-16 | W9 / Request-id and access-log baseline | done | Added low-risk request-id plus access-log middleware at the server assembly boundary so `/api/*`, `/healthz`, and `/readyz` now return `X-Request-Id` and emit matching access logs with client, method, path, status, bytes, and duration; added focused server tests and aligned README wording | API and probe diagnostics now have a concrete correlation hook without reopening logging architecture or touching handler/business logic | Next: keep W9 focused on broader browser matrix, release flow, and richer observability beyond this baseline | Risk: logs remain stdlib text output, and there is still no metrics or tracing pipeline
- 2026-04-16 | W6-W9 / Bootstrap split, scoped pending ownership, and auth-audit request correlation | done | Split root runtime state/bootstrap/background helpers out of `main.go`, added request-id propagation into auth audit events, tightened the isolated React board so create/move/archive/restore only disable their local work scope and restore focus for repeated interaction, then re-ran local Go, Windows race, web, and Docker-backed `chromium` plus `firefox` smoke verification | W6 is materially closer to assembly-only startup, W8 has a more usable keyboard/touch-friendly non-drag flow, and W9 now correlates access logs with auth audit events without reopening the logging architecture | Next: continue W6 deeper service/domain extraction and keep W9 focused on broader browser matrix, release flow, and richer observability | Risk: background cleanup loops still use unmanaged goroutines, runtime ownership is still split between the embedded frontend and the isolated React shell, and observability is still limited to stdlib logs without metrics or tracing
- 2026-04-16 | W6-W9 / Auth transport split, single-announcement cleanup, and release dry-run baseline | done | Split auth cookie/context/runtime helpers out of `auth_http.go`, removed duplicate board status announcements from the isolated React shell, added repo-owned release dry-run scripts that build a checksumed artifact and reuse it for smoke verification, wired a manual `workflow_dispatch` release-dry-run job in CI, and documented the current rollback baseline | W6 now has a cleaner auth transport boundary that mirrors the task-side seams more closely, W8 is less noisy for assistive tech, and W9 now has a concrete first release-governance path instead of only verification and smoke gates | Next: continue W6 toward deeper service/domain extraction and keep W9 focused on broader browser matrix and richer observability | Risk: release governance is still single-platform and manual, and the project still lacks metrics/tracing plus a final versioning/changelog policy
- 2026-04-16 | W6-W9 / Auth orchestration split, cancellable cleanup loops, and `/next/` preview runtime slice | done | Added a dedicated auth orchestration service, made background cleanup loops respect cancellation, introduced a Go-served `/next/` React preview route with SPA fallback for built `web/dist`, added repo-owned preview verification scripts plus preview smoke coverage, and re-ran local Go, race, web, legacy smoke, and `/next/` preview smoke verification against Docker-backed PostgreSQL | W6 now has a cleaner auth/service boundary with better runtime lifecycle control, W7/W8 now have their first real Go-owned preview route without replacing the embedded UI, and W9 now proves both legacy and preview shells through repo-owned smoke paths | Next: keep shrinking deeper domain rules out of the `main` package, then decide whether the next runtime-ownership slice should promote `/next/` further or focus on drag/mobile polish first | Risk: `/next/` is still preview-only, `cmd/flux-board` does not exist yet, and broader browser/observability/release governance work remains open
