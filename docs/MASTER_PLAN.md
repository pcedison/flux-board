# Flux Board Roadmap

## Purpose
- This document is the contributor roadmap for Flux Board.
- It explains what has already been delivered, what quality bar each wave must clear, and what work remains inside the single-user product direction.
- If you just want to fork, deploy, or evaluate the app, start with `README.md` and the deployment docs instead of this planning file.

## Product Direction
- Flux Board is a single-user self-hosted task board.
- One deployed instance serves one operator.
- The supported production contracts are:
  - the repo-root Docker image
  - the self-contained root binary built from `go build .`
- Multi-user accounts, RBAC, workspaces, and OIDC remain deliberate non-goals.

## Acceptance Model
- `W0-W1` focus on release hygiene, documentation, and public-fork baseline work.
- `W2-W17` close only when they have:
  - repo-owned local verification
  - exact-head GitHub Actions proof
  - hosted or release evidence when the wave touches deployment, packaging, or operator workflows
- `W18` stays unopened unless a real gap appears that does not belong inside `W15-W17`.

## Current Maturity
- Flux Board should be treated as a strong single-user self-hosted beta with release-grade CI, hosted verification, restore drills, and operator docs.
- The project is optimized for "fork -> deploy -> finish setup or sign in -> use one board as one operator".

## Wave Summary
| Wave | Focus | Public Status |
|---|---|---|
| W0 | Baseline audit | delivered |
| W1 | Public-fork baseline | delivered |
| W2 | CI and reproducibility | delivered |
| W3 | Server security hardening | delivered |
| W4 | Auth and session redesign | delivered |
| W5 | Schema and data integrity | delivered |
| W6 | Go modularization | delivered |
| W7 | Frontend foundation | delivered |
| W8 | UX, RWD, and accessibility | delivered |
| W9 | Quality gates, release, and runtime safety | release hardening |
| W10 | Build, CI, and hosted deploy hardening | delivered |
| W11 | Single-user security and settings | delivered |
| W12 | Product UX completion | delivered |
| W13 | Data portability and backup | delivered |
| W14 | Observability and operability | delivered |
| W15 | Hosted release operations | release hardening |
| W16 | Backup and restore drills | delivered |
| W17 | Product polish and mobile depth | delivered |
| W18 | Post-polish expansion | unopened by design |

## Delivered Baseline
### W0-W8
- Reproducible CI and verification scripts
- hardened HTTP baseline
- single-user auth with DB-backed sessions
- versioned migrations and stronger data-integrity checks
- modular Go packages under `internal/`
- React runtime on `/` with `/legacy/` kept as rollback shell
- mobile-first board interactions with keyboard and smoke coverage

### W10-W17
- Docker-first hosted contract and release parity
- `/setup` and `/settings` for bootstrap, password rotation, session revocation, retention, and JSON import/export
- operator-facing `/status` and `/api/status`
- repo-owned hosted deploy, hosted auth, and restore-drill verification assets
- documented hosted troubleshooting and backup/restore operations
- final product-name and copy polish for the single-user runtime

## Remaining Work
### W9
- Keep release, checksum, and runtime-safety evidence aligned with the latest tagged head.
- Keep exact-head CI closure intact as the repo evolves.

### W15
- Keep hosted deployment, GitHub Release, and GHCR evidence aligned on the same release head.
- Preserve rollback clarity and operator verification for every tagged release.

### W18 Boundary
- Do not open `W18` just to hold release-sync, hosted-proof, or polish work that still belongs to `W9`, `W15`, `W16`, or `W17`.
- Open `W18` only if a new product gap appears after the current single-user roadmap is actually complete.

## Related Docs
- Start here for adoption and local setup: [README.md](../README.md)
- Deployment contract: [docs/DEPLOYMENT.md](DEPLOYMENT.md)
- Architecture summary: [docs/ARCHITECTURE.md](ARCHITECTURE.md)
- Operator runbook: [docs/OPERATIONS_RUNBOOK.md](OPERATIONS_RUNBOOK.md)
- Backup and restore drill: [docs/BACKUP_RESTORE_DRILL.md](BACKUP_RESTORE_DRILL.md)
- Contributor roadmap detail: [docs/ROADMAP_W10_W14.md](ROADMAP_W10_W14.md)
