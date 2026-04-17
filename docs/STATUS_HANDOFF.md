# Flux Board Status Handoff

Last updated: 2026-04-17

## Read This First
If you only read one file, read this one first:
- [docs/STATUS_HANDOFF.md](STATUS_HANDOFF.md): current progress, blockers, paused items, and next-step roadmap

Then read in this order:
1. [docs/AGENT_WORK_PLAN.md](AGENT_WORK_PLAN.md): complete phased work plan, slice-by-slice tasks, priorities, and absolute prohibitions for agent handoff
2. [docs/MASTER_PLAN.md](MASTER_PLAN.md): full 10-wave source of truth and execution log
3. [README.md](../README.md): run, verify, and repo layout
4. [docs/ARCHITECTURE.md](ARCHITECTURE.md): current runtime and target shape
5. [docs/DEPLOYMENT.md](DEPLOYMENT.md): current deploy, smoke, dry-run, and rollback baseline

## Simplified Closure Rule
- `W0-W1` 的最終完成標準是 `artifact-complete`。
- `W2-W9` 的最終完成標準是 `remote-closed`。
- `locally-verified` 只代表目前 head 的本地驗證已通過，還不能視為最終關門。

## Current Acceptance Snapshot
- `W0-W1`：`artifact-complete`
- `W2-W8`：`locally-verified`
- `W9`：`in_progress`
- 最新、最強的本地證據集中在 `W7-W8`：React runtime 已接管 `/`，`/legacy/` 保留回滾，drag/mobile/keyboard slices 都已本地驗證通過。
- 目前最大的剩餘缺口只有兩件事：
  - `W2-W8` 還缺 exact current head 的 fresh GitHub Actions 綠燈紀錄
  - `W9` 的 observability、release、browser-matrix scope 仍未完成

## Evidence Snapshot
- 已通過的本地驗證：
  - `./scripts/verify-go.ps1`
  - `./scripts/verify-go-race.ps1`
  - `./scripts/verify-web.ps1`
  - `./scripts/verify-smoke.ps1`
  - `./scripts/verify-next-preview.ps1`
  - `./scripts/verify-dnd-smoke.ps1`
  - `./scripts/verify-board-keyboard-smoke.ps1`
- Docker-backed local smoke 已涵蓋：
  - canonical React runtime on `/login -> /board` in `chromium` and `firefox`
  - `/next/login` compatibility redirect plus `/legacy/` rollback path in `chromium` and `firefox`
  - W8 drag / mobile / keyboard smoke matrix
- 目前仍不能把 `W2-W8` 寫成最終完成，因為文件裡還沒有 exact current head 的 fresh remote CI 關門證據。
- Windows 注意事項：不要把 `go test ./...` 和 `npm ci` 並行執行；掃描 `web/node_modules` 可能在 Windows 上短暫失敗。
- 最新觀察到的 remote CI：
  - GitHub Actions run `24516178553`
  - `verify`: success
  - `smoke (chromium)`: success
  - `smoke (firefox)`: success
  - `preview_smoke`: success
  - 但這個 run 早於 root-runtime takeover，不能拿來當 `W7-W8` 的最終關門證據

## Wave Status
| Wave | Current status | Evidence | Final closure |
|---|---|---|---|
| `W0` | `artifact-complete` | baseline audit, blocker map, and architecture snapshot are documented | already at its final review-based closure |
| `W1` | `artifact-complete` | public-fork docs, onboarding, and repo hygiene are documented | already at its final review-based closure |
| `W2` | `locally-verified` | repo-owned verification scripts and historical CI baseline exist | record fresh remote CI for the exact current head |
| `W3` | `locally-verified` | server hardening baseline exists and local verification is available | record fresh remote CI for the exact current head |
| `W4` | `locally-verified` | single-admin auth/session baseline exists with local proof | record fresh remote CI for the exact current head |
| `W5` | `locally-verified` | migrations, reorder correctness, and archive/restore behavior have local proof | record fresh remote CI for the exact current head |
| `W6` | `locally-verified` | internal package boundaries and `cmd/flux-board` exist and were locally re-verified | record fresh remote CI for the exact current head |
| `W7` | `locally-verified` | React runtime owns `/`, `/legacy/` is the rollback shell, and `/next/*` redirects remain for compatibility | record fresh remote CI for the root-runtime takeover head |
| `W8` | `locally-verified` | same-lane drag, mobile-first layout, keyboard/focus polish, axe checks, and browser smoke are locally green | record fresh remote CI for the exact head with `dnd_smoke` and `keyboard_smoke` |
| `W9` | `in_progress` | strong local and partial CI baseline already exist | finish observability/release/browser-matrix scope, then close with fresh remote CI |

## Main Development Hard Points
- `Dual-runtime complexity`
  - `/` is now the React runtime
  - `/legacy/` is the embedded rollback runtime
  - `/next/*` is now a compatibility redirect, so route ownership is clearer but rollback discipline still matters
- `Deep modularization for the planned W6 scope is complete`
  - root `main` is now compatibility-focused wiring plus thin wrappers
  - future work can extend the extracted seams instead of reopening the root package
- `Frontend runtime takeover is now landed locally but still needs remote proof`
  - React is now Go-served on `/`
  - `/legacy/` remains the emergency rollback path
  - the next real risk is not ownership itself, but avoiding regressions while W8 `3-C` keyboard/focus work finishes
- `W8 is now locally complete`
  - same-lane pointer-first drag reorder is now in place and locally verified
  - mobile-first layout is now in place and locally verified
  - keyboard/focus polish plus explicit a11y checks are now in place and locally verified
- `W9 still lacks production-grade observability`
  - request-id/access log baseline exists
  - metrics, tracing, structured logging pipeline, and richer release policy do not

## Why Work Has Been Paused or Deferred
### Intentional defer reasons
- `Do not remove rollback too early`
  - React now owns `/`, but `/legacy/` is intentionally preserved until later waves prove the new runtime more deeply
- `Do not over-split Go too early`
  - service/repo extraction is happening in small verified slices to avoid regressions
- `Do not over-claim enterprise auth`
  - current auth is a safer single-admin baseline, not multi-user or OIDC
- `Do not jump to drag/drop before runtime foundation is stable`
  - W8 drag/mobile work is deferred until W6/W7 are steadier

### Real interruption causes already encountered
- earlier completion overstatement had to be corrected before work could continue honestly
- Docker/PostgreSQL availability was required for some local smoke/integration evidence
- Windows local race testing required installing and wiring a working MSYS2 UCRT64 GCC toolchain
- on Windows, `go test ./...` should not be run in parallel with `npm ci` because `web/node_modules` scans can flake
- remote CI confirmation was needed before closing some slices, so some work paused until GitHub Actions finished

## Current Blockers and Non-Blockers
### Current blockers for `remote-closed` status
- `W2-W8`
  - a fresh GitHub Actions run for the exact current head has not yet been recorded in the docs
- `W7-W8`
  - this matters most here because the runtime takeover plus new drag/keyboard smoke lanes are the newest local-only changes
- `W9`
  - the wave is still functionally incomplete, so remote closure is not yet the main problem

### Current non-blockers
- Node 24 JavaScript action runtime pilot is already in place
- canonical runtime plus `/next/*` redirect and `/legacy/` rollback all pass their current local smoke paths
- local Windows race verification is available and working

## Current Push Method
The active implementation method is:
1. choose one small high-value slice
2. review locally and with subagent before editing
3. land the smallest safe change
4. run repo-owned local verification
5. run Docker-backed smoke if the slice affects runtime behavior
6. update docs and `MASTER_PLAN`
7. push and wait for GitHub Actions proof

This is intentionally slower than MVP-style iteration, but it is much safer for a public-fork-quality baseline.

## Recommended Next Roadmap
### Immediate next wave focus
1. continue `W9`
   - add observability slices
   - add release-governance slices
   - widen browser matrix only when the runtime path is stable enough
2. promote `W2-W8` from `locally-verified` toward `remote-closed`
   - push the current head
   - observe fresh GitHub Actions proof
   - record the run evidence in `MASTER_PLAN`

### Likely next concrete work items
- `W9`: observe fresh remote CI after the new `dnd_smoke` and `keyboard_smoke` lanes are pushed
- `W9`: add a first metrics or richer structured logging slice

## What Not To Do Next
- do not remove `/legacy/` or `/next/*` compatibility coverage before the new root-runtime CI path is observed green
- do not reopen `W0-W5` unless a real regression appears
- do not introduce drag-and-drop as the only move path
- do not claim public-production-ready release governance yet
- do not merge major new scope without updating `MASTER_PLAN`

## Resume Rule
If work is interrupted again:
1. read [docs/STATUS_HANDOFF.md](STATUS_HANDOFF.md)
2. read the `Wave Status Board` in [docs/MASTER_PLAN.md](MASTER_PLAN.md)
3. read the latest `Execution Log` entry in [docs/MASTER_PLAN.md](MASTER_PLAN.md)
4. confirm the current branch and CI state
5. only then pick the next smallest verified slice
