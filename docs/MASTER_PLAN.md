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
| W5 | Schema and Data Integrity | Main agent | planned | Migrations and reorder correctness verified |
| W6 | Go Modularization | Main agent | planned | Core logic testable and layered |
| W7 | Frontend Foundation | Main agent | planned | New React frontend builds and talks to API |
| W8 | Trello-grade UX, RWD, A11y | Main agent | planned | Mouse/touch/keyboard all pass core flows |
| W9 | Quality Gates, Release, Enterprise Hooks | Main agent | planned | Public-fork release ready |

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

## W6 Go Modularization
- Goal: turn the backend into a maintainable, testable layered service.
- Epics: `cmd/` entrypoint, config, handlers, services, repositories, domain errors.
- Tasks: shrink `main.go` to assembly, separate HTTP/service/repo layers, extract pure logic, add dependency boundaries and test seams.
- Gate: core logic is no longer trapped in `main.go` and can be unit-tested.
- Parallel lanes: handler split, service split, repo abstraction.

## W7 Frontend Foundation
- Goal: replace the monolithic HTML with a modern, maintainable frontend base.
- Epics: `web/` scaffold, routing, API client, query layer, design tokens, shell layout.
- Tasks: create React + TypeScript + Vite app, add React Router, TanStack Query, app shell, tokenized style system, auth guard and base pages.
- Gate: new frontend can build, typecheck, and communicate with the Go API.
- Parallel lanes: design system, API layer, shell and routing.

## W8 Trello-grade UX, RWD, and Accessibility
- Goal: deliver rich board interactions that work on desktop, tablet, and mobile.
- Epics: board/list/card UI, `dnd-kit`, optimistic update, explicit move controls, mobile-first layouts, accessibility.
- Tasks: implement CRUD UI, drag-and-drop as progressive enhancement, touch fallback, keyboard move controls, dialog/menu focus management, 44px targets, browser matrix checks.
- Gate: core board movement works via mouse, touch, and keyboard; mobile and tablet are first-class; drag-and-drop is not the only path.
- Parallel lanes: board UI, drag/drop, RWD, a11y verification.

## W9 Quality Gates, Release, and Enterprise Hooks
- Goal: make the project release-ready and extensible toward enterprise features.
- Epics: automated tests, browser matrix, release flow, rollback docs, observability, enterprise extension seams.
- Tasks: add unit/integration/E2E coverage, verify Chromium/Firefox/WebKit, define release/rollback process, add health/metrics/logging, reserve extension points for RBAC/SSO/workspaces.
- Gate: project is safe to publish, testable in CI, and ready for controlled future enterprise expansion.
- Parallel lanes: QA, release engineering, observability, enterprise design.

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
- `W5-P1` Formal migrations: add migration toolchain, up/down files, CI validation. Done: schema changes are replayable and rollbackable. Parallel: with W6 repo work.
- `W5-P2` Normalized model: redesign `boards/lists/cards/archive/sessions`, add FK / NOT NULL / CHECK / index. Done: DB enforces core invariants. Parallel: schema and query lanes can split.
- `W5-P3` Reorder correctness: move reorder to transactional batch update and remove `MAX()+1` races. Done: ordering survives concurrency and reload. Parallel: with frontend drag API design.
- `W5-P4` Archive correctness: verify archive, restore, delete, and `rows.Err()` handling. Done: CRUD and archive flows have no silent failure. Parallel: with integration tests.
### W6
- `W6-P1` Assembly-only main: move bootstrap into `cmd/`, config, router, startup wiring. Done: `main.go` no longer owns business logic. Parallel: pure structural move first.
- `W6-P2` Layer split: separate handler, service, postgres repo with DI boundaries. Done: each layer is replaceable and testable. Parallel: coordinate with W5 query changes.
- `W6-P3` Pure rules and domain errors: extract validation, ordering, mapping, error taxonomy. Done: core rules are unit-testable without HTTP/DB. Parallel: good subagent slice.
- `W6-P4` Test seams: add mocks, fixtures, handler/service tests, repo test seams. Done: major flows have repeatable unit/integration coverage. Parallel: tie into W2 CI.
### W7
- `W7-P1` Frontend scaffold: create `web/`, React + TypeScript + Vite, routing, startup flow. Done: app builds, typechecks, and runs locally. Parallel: can overlap late W6.
- `W7-P2` Data layer: add API client, auth guard, TanStack Query, error handling. Done: frontend reads API with stable loading/error states. Parallel: with W7-P3.
- `W7-P3` Design system: define tokens, CSS variables, shell layout, base component rules. Done: spacing, color, type, breakpoints are reusable. Parallel: with W7-P2.
- `W7-P4` Page skeletons: build Board/Login/Base pages, page containers, empty states. Done: new UI shell works without single-file dependency. Parallel: after W7-P1.
### W8
- `W8-P1` Core board UI: implement board/list/card CRUD and archive/restore entry points. Done: core board actions complete with consistent UI. Parallel: with W8-P2.
- `W8-P2` Drag and reorder: use `dnd-kit`, optimistic update, rollback, sync to reorder API. Done: mouse/touch/keyboard can move cards. Parallel: depends on W5 reorder API.
- `W8-P3` Non-drag movement: add move menu, move-to-list, up/down actions. Done: movement does not depend on drag-and-drop alone. Parallel: with W8-P2.
- `W8-P4` RWD and a11y: mobile-first layouts, 44px hit areas, focus trap, focus-visible, dialog/menu behavior. Done: desktop/tablet/mobile and keyboard all pass core flows. Parallel: with W8-P1.
### W9
- `W9-P1` Test gates: add backend unit/integration, frontend component tests, Playwright E2E. Done: core flows are automatically testable and repeatable. Parallel: with W9-P2.
- `W9-P2` CI and release flow: finalize GitHub Actions, release notes, rollback, versioning. Done: CI is stable and release process is reproducible. Parallel: partial dependency on W9-P1.
- `W9-P3` Observability: add health/readiness, structured logs, metrics, slow-query logging. Done: issues are monitorable and diagnosable. Parallel: with W9-P2.
- `W9-P4` Enterprise extension seams: reserve RBAC/SSO/workspace hooks and document them. Done: architecture can grow without harming public-fork simplicity. Parallel: after W7-W8 stabilize.

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
- 2026-04-16 | W1-W2 / Deployment and deterministic Node install tightening | done | Added `docs/DEPLOYMENT.md`, linked it from `README.md`, switched smoke install guidance to `npm ci`, updated CI to use `npm ci`, and refreshed `package-lock.json` so browser tooling installs are pinned | W1 deployment guidance and W2 dependency reproducibility are materially tighter, though both waves still remain open pending tracked-history proof and an observed passing CI run | Next: keep W1/W2 open until the repo-owned files are committed and the CI smoke is observed passing | Risk: current evidence is still limited by the absence of a fresh-clone proof and a live GitHub Actions success in this thread
- 2026-04-16 | W4 / Real DB-backed auth integration test added | done | Added `main_integration_test.go`, which skips without `DATABASE_URL` but exercises login failure, login success, `/api/auth/me`, logout, post-logout `401`, and auth audit log writes against a real PostgreSQL-backed app when a DB-capable environment exists | W4 now has repo-owned real-database integration coverage ready for CI/DB-capable runs, while local no-DB environments still remain stable because the test skips cleanly | Next: observe or rerun this test in a DB-capable environment before treating W4 as closeable | Risk: the integration test exists, but this thread has not yet observed it execute against a live database
- 2026-04-16 | W1-W2 / Tracked-history and fresh-clone proof | done | Created branch `codex/w1-w2-w4-closeout`, committed the W1/W2/W4 baseline, and validated a clean clone by checking `.env.example`, running `./scripts/verify-go.ps1`, running `npm ci`, and syntax-checking the repo-owned smoke script | W1 now has tracked-history proof and a clean-clone reproducibility record for the current scope | Next: observe GitHub Actions before closing W2 | Risk: clean-clone runtime app startup still depends on a database-capable environment
- 2026-04-16 | W2 / Smoke failure bounded for diagnosis | done | Added request-level and overall smoke timeouts plus workflow-level timeout so browser smoke can no longer hang indefinitely in CI | W2 failure mode is now actionable instead of silent hanging | Next: fix the root cause behind the first smoke 401 and rerun CI | Risk: none
- 2026-04-16 | W2-W4 / CI contamination fix and clean-environment proof | done | Removed the duplicate DB-backed auth test that polluted the default CI schema, reran the PR workflow, and observed a green GitHub Actions run for `go test`, `go vet`, `go build`, app startup, and browser smoke on run `24494722677` | W2 is now proven on a clean environment, and W4 has observed DB-backed auth/session evidence plus browser smoke proof for the current single-admin baseline | Next: keep future auth evolution and multi-user work in later waves, not by reopening W4 scope silently | Risk: later waves still need multi-user/OIDC and richer session controls
