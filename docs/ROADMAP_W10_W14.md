# Flux Board W10-W14 Roadmap

## Purpose
- This document defines the next planning-only roadmap after the recorded `W0-W9` closure baseline.
- It is intentionally non-implementation-focused: the goal is to give the next machine or agent a clean, dependency-aware starting point without silently expanding scope.
- Final closure rule:
  - `W10-W14` do not count as complete at `artifact-complete`
  - every implementation wave must end at `remote-closed`
  - `remote-closed` means the exact current head has fresh GitHub Actions proof and the relevant hosted/deployment path has been exercised for that wave

## Guiding Principles
- `security and correctness > architecture > UX > polish`
- Do not implement identity/provider features before the local principal and authorization model exists.
- Do not treat hosted deployment as equivalent to a local binary path unless both are verified.
- Do not claim observability maturity from middleware alone; dashboards, alerts, and runbooks are part of the productized deliverable.
- Keep rollback paths explicit for every auth, deployment, and runtime ownership change.

## Recommended Execution Order
1. `W10` Build, CI, and Hosted Deploy Hardening
2. `W11` Multi-user Auth Foundation
3. `W12` Workspace and RBAC Foundation
4. `W13` OIDC and SSO Integration
5. `W14` Observability Productization

## Dependency Summary
| Wave | Depends on | Why |
|---|---|---|
| W10 | W0-W9 | hardens the already-closed runtime, CI, and release baseline |
| W11 | W10 | multi-user auth should land on the modernized build/deploy and CI baseline |
| W12 | W11 | roles and workspace membership must build on a real user model |
| W13 | W11, W12 | OIDC should bind to stable users, sessions, and workspace membership |
| W14 | W10, ideally W11-W13 | observability labels, dashboards, and alerts are more useful after principal and workspace boundaries are stable |

## Shared Remote-Closed Standard
- Local verification:
  - `./scripts/verify-go.ps1`
  - `./scripts/verify-go-race.ps1`
  - `./scripts/verify-web.ps1`
  - any wave-specific smoke or integration coverage added during that wave
- Remote verification:
  - exact-head GitHub Actions green for the required matrix
  - wave-specific checks added to CI during the wave
- Deployment verification:
  - at least one real hosted or container deployment path exercised for the wave's changed surface
  - rollback or disable path documented and tested where applicable
- Documentation:
  - `docs/MASTER_PLAN.md` updated
  - wave-specific docs updated
  - `Execution Log` evidence appended

## W10 Build, CI, and Hosted Deploy Hardening
### Goal
- Make the build, CI, release, and hosted deployment paths consistent enough that later identity and observability work does not rest on fragile assumptions.

### Why This Wave Exists
- The project now has a strong local and CI baseline, but there is still room to harden hosted deploy ergonomics, remove residual CI modernization debt, and reduce reliance on environment-specific tribal knowledge.

### Scope
- CI modernization
- hosted deployment path hardening
- release-path consistency
- asset packaging strategy review

### Work Packages
- `W10-P1` CI modernization
  - upgrade or replace third-party GitHub Actions that still emit Node 20 deprecation annotations
  - pin action versions intentionally
  - add `actionlint` or equivalent workflow validation
- `W10-P2` hosted deploy contract
  - formalize Zeabur and generic hosted deployment guidance around the repo-root `Dockerfile`
  - verify startup, health, readiness, login, and static asset behavior in a hosted-like path
- `W10-P3` asset packaging strategy
  - decide whether production should continue with container-bundled `web/dist` or move to embed-into-binary
  - document tradeoffs, rollback, and operational constraints
- `W10-P4` dependency update governance
  - add a clear policy for Go, Node, Playwright, and GitHub Action updates
  - decide whether to add Dependabot or Renovate
- `W10-P5` release-path parity
  - ensure dry-run, release workflow, and hosted deployment all exercise the same artifact assumptions

### Deliverables
- updated CI workflows with reduced deprecation debt
- hosted deployment documentation that reflects the actual supported path
- asset packaging decision record
- dependency update policy

### Verification
- rerun the full CI matrix on the exact head
- verify the selected hosted path from build to healthy login-capable runtime
- record whether Node 20 action-runtime annotations are fully removed or still intentionally accepted

### Dependencies
- relies only on the current `W0-W9` baseline
- should be completed before any deeper auth or RBAC work

### Risks
- hosted providers may behave differently around ports, writable filesystems, or asset discovery
- embed-into-binary may increase binary size and require different build ergonomics

### Out of Scope
- multi-user auth
- RBAC
- SSO
- dashboard and alert productization

## W11 Multi-user Auth Foundation
### Goal
- Replace the single-admin bootstrap model with a real multi-user local-auth baseline.

### Why This Wave Exists
- The current auth/session design is a solid security baseline, but it is still structurally a single-principal system. That blocks meaningful RBAC, workspace ownership, and provider-backed identity.

### Scope
- local multi-user account model
- user lifecycle management
- session management improvements
- auth audit taxonomy expansion

### Work Packages
- `W11-P1` user model expansion
  - add durable user fields such as email, display name, status, and creation metadata
  - define whether login identifiers are email-only or username-plus-email
- `W11-P2` user management flows
  - admin-driven invite or create flow
  - deactivate/reactivate user flow
  - password reset initiation and completion flow
- `W11-P3` session controls
  - add clearer session revocation and active-session inspection
  - decide retention and cleanup behavior
- `W11-P4` frontend account/admin surfaces
  - add minimal admin user-management screens
  - add a basic self-service account settings surface
- `W11-P5` audit and policy coverage
  - expand auth audit event taxonomy
  - verify lockout, reset, activation, deactivation, and session revocation behaviors

### Deliverables
- `users` model that supports more than one human operator
- admin user lifecycle flows
- password reset path
- stronger session management
- updated auth docs and audit-event documentation

### Verification
- at least two separate users can authenticate independently
- inactive users cannot authenticate
- revoked sessions lose access
- auth audit events capture the new lifecycle flows

### Dependencies
- should follow `W10`
- is the base for `W12` and `W13`

### Risks
- account recovery and admin recovery paths can create accidental lockout risk
- migration from the current single-admin baseline needs careful bootstrap/backfill rules

### Out of Scope
- workspace membership
- role-based authorization
- external identity providers

## W12 Workspace and RBAC Foundation
### Goal
- Turn the current enterprise seams into real workspace-scoped data boundaries and role enforcement.

### Why This Wave Exists
- Without workspace ownership and role membership, multi-user auth still shares one undifferentiated authority plane.

### Scope
- workspace data model
- membership and role model
- authorization enforcement in services and handlers
- workspace-aware frontend shell

### Work Packages
- `W12-P1` workspace schema
  - add `workspaces` and membership tables
  - add `workspace_id` to the relevant task/archive data paths
  - define backfill for the current single-workspace baseline
- `W12-P2` role model
  - define the initial roles, for example `owner`, `admin`, `member`, `viewer`
  - document per-surface permissions
- `W12-P3` authorization enforcement
  - move policy checks into stable service-level seams
  - avoid scattering authorization logic across handlers
- `W12-P4` workspace resolution and isolation
  - define how the current request chooses a workspace
  - add tests that prove cross-workspace isolation
- `W12-P5` frontend workspace UX
  - add workspace switching or selection
  - hide or disable actions according to role and membership

### Deliverables
- workspace-aware schema and service boundaries
- explicit role model
- authorization checks for task/archive/auth surfaces that depend on role or membership
- workspace-aware frontend behavior

### Verification
- data from one workspace is invisible to another
- role-specific permissions are enforced by the server, not just the UI
- migration from the existing single-workspace database succeeds cleanly

### Dependencies
- depends on `W11`
- should precede `W13`

### Risks
- accidental leakage through list/read endpoints is higher-risk than mutation bugs
- authorization logic can become inconsistent if it lands in handlers instead of service seams

### Out of Scope
- provider-backed identity federation
- advanced enterprise policies such as SCIM or just-in-time team sync

## W13 OIDC and SSO Integration
### Goal
- Add enterprise-grade provider-backed login while preserving a safe local break-glass path.

### Why This Wave Exists
- Once users, sessions, workspaces, and roles are real first-class concepts, the project can safely support SSO instead of bolting it onto a single-admin bootstrap model.

### Scope
- OIDC login flow
- identity linking
- callback/logout handling
- local break-glass admin preservation

### Work Packages
- `W13-P1` provider configuration model
  - define OIDC issuer, client, secret, scopes, redirect URIs, and mode flags
- `W13-P2` callback and token validation
  - implement state, nonce, JWKS, and claim validation
- `W13-P3` identity linking
  - add a durable identity-link table
  - decide invitation binding vs just-in-time provisioning rules
- `W13-P4` runtime modes
  - document and enforce `local-only`, `hybrid`, and `SSO-first` style modes
  - preserve a break-glass local admin path
- `W13-P5` frontend login UX
  - add provider sign-in entry points and post-login handling
  - keep failure messaging explicit and operator-friendly

### Deliverables
- OIDC provider support
- identity-link persistence
- callback/logout flows
- documented runtime modes and break-glass behavior

### Verification
- mocked or test-provider-backed OIDC login succeeds end to end
- invalid state/nonce/token flows fail safely
- break-glass local admin remains usable in the configured modes that require it

### Dependencies
- depends on `W11` and `W12`

### Risks
- callback correctness, token validation, and logout semantics are easy to get subtly wrong
- operator lockout risk is high if break-glass design is weak

### Out of Scope
- SAML
- SCIM
- enterprise directory sync beyond the identity-link model needed for OIDC

## W14 Observability Productization
### Goal
- Turn the current logs/metrics/trace baseline into an operator-grade package with dashboards, alerts, and runbooks.

### Why This Wave Exists
- Middleware and endpoints are necessary but not sufficient. Public-fork and operational maturity improve dramatically when maintainers can actually observe, diagnose, and respond to production incidents using repo-owned assets.

### Scope
- dashboard bundle
- alert bundle
- runbooks
- trace/log/metric field discipline

### Work Packages
- `W14-P1` log schema review
  - define the stable log fields and correlation expectations
  - review whether auth, workspace, and request identifiers are sufficient
- `W14-P2` metrics productization
  - define Prometheus recording rules or example queries
  - confirm the labels are useful and bounded
- `W14-P3` trace productization
  - verify ingress propagation, service spans, and DB spans
  - provide an operator-facing collector example
- `W14-P4` dashboard bundle
  - add repo-owned Grafana dashboard JSON or equivalent
  - cover latency, error rate, auth failures, task mutation load, and queue/cleanup visibility where relevant
- `W14-P5` alerts and runbooks
  - add alert definitions and response guidance
  - connect the common failure modes to concrete runbook steps

### Deliverables
- stable log-field schema
- repo-owned metrics guidance
- repo-owned trace/collector example
- dashboard bundle
- alert definitions and runbooks

### Verification
- a local or hosted environment can display request-to-service-to-store telemetry for representative flows
- at least one alert path can be intentionally triggered and observed
- the dashboards remain aligned with the implemented metrics and traces

### Dependencies
- depends on `W10`
- benefits from `W11-W13` being stable so dimensions such as user, workspace, and provider mode are meaningful

### Risks
- unbounded label cardinality can make metrics unsafe
- dashboards can drift quickly if they are not versioned with the instrumentation they assume

### Out of Scope
- full commercial observability packaging or vendor-specific lock-in
- automatic incident management integrations beyond basic documentation and examples

## Suggested Handoff After Sync
- If the next machine is starting implementation immediately, begin with `W10-P1` and `W10-P2`.
- If the next machine is doing issue planning first, split each wave into 1-3 day slices before touching code.
- Keep `docs/MASTER_PLAN.md` as the status source of truth and use this file as the forward-looking design map.
