# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

Future work can build on the current observability, release, and extension seams
without reopening the initial public-fork baseline.

## [0.1.4] - 2026-04-19

### Added

- Added repo-owned hosted verification scripts for the live deployment path,
  including GitHub deployment resolution and `/api/status` contract artifacts.
- Added operator-facing runbooks for hosted troubleshooting and backup/restore
  drills, plus a scratch restore rehearsal that proves sign-in, board access,
  settings access, and status checks on a restored instance.

### Changed

- Hardened settings import validation so malformed bundles are rejected before
  they can replace the current board snapshot.
- Expanded settings smoke coverage to prove rejected imports do not mutate the
  live board or archive-retention state before the happy-path import continues.
- Formalized `W15-W17` as real single-user roadmap waves and kept `W18`
  explicitly unopened unless a concrete post-polish scope appears.
- Polished the web runtime title from the older preview wording to the final
  product name.

## [0.1.3] - 2026-04-18

### Added

- Added first-run `/setup`, `/settings`, password rotation, session revocation,
  archive-retention controls, and JSON export/import for the single-user runtime.
- Added frontend lint, `golangci-lint`, `actionlint`, `docker-compose.yml`, and a
  Render deployment template to the productization baseline.
- Added `/api/status`, the `/status` operator page, a setup-first browser smoke
  lane, and Docker runtime smoke automation for both bootstrap and daily-login paths.

### Changed

- Switched the supported release artifact contract from `./cmd/flux-board` to the
  self-contained root binary built from `go build .`.
- Updated the Dockerfile, release dry-run scripts, and release workflow so the
  embedded root binary and Docker image now share the same runtime assumptions.
- Updated CI and tag releases so Docker is treated as a first-class contract,
  including runtime smoke in CI and GHCR image publishing on tagged releases.
- Repositioned the roadmap and docs around the real single-user self-hosted
  product target instead of future multi-user scope.

## [0.1.2] - 2026-04-17

### Fixed

- Upgraded the OpenTelemetry dependency set to a non-vulnerable release so the
  current tracing path no longer trips `govulncheck` on the exact release head.
- Added a repo-owned container build path for hosted platforms such as Zeabur so
  the Go binary, migrations, legacy rollback assets, and `web/dist` runtime are
  built and shipped together instead of depending on platform-specific guesswork.

### Changed

- Raised the documented and CI-backed Go baseline to `1.24` to match the
  dependency graph required by the current observability stack.
- Clarified deployment and repository docs so hosted environments use the Docker
  image path when they need the canonical React runtime on `/`.

## [0.1.1] - 2026-04-17

### Fixed

- Corrected the tag-triggered GitHub Release workflow so release metadata is staged
  in a non-hidden directory before `actions/upload-artifact@v4` runs, allowing the
  release job to publish notes and checksums on the exact tagged head.

## [0.1.0] - 2026-04-17

### Added

- Established repo-owned verification and browser smoke gates for the Go backend,
  React runtime, compatibility rollback path, drag-and-drop flow, and keyboard
  accessibility flow across Chromium and Firefox.
- Completed the W6 backend modularization slices with `internal/domain`,
  `internal/store/postgres`, `internal/service`, `internal/transport/http`, and
  the canonical `cmd/flux-board` entrypoint.
- Promoted the React runtime to own `/`, retained `/legacy/` as the rollback path,
  and preserved `/next/*` compatibility redirects.
- Added mobile-first board layout, same-lane drag reorder, roving tabindex support,
  focus restoration, and accessibility coverage for the board interactions.
- Introduced `VERSION` as the single release source, a Keep a Changelog release log,
  version-aware dry-run packaging, and a tag-triggered GitHub Release workflow scaffold.

### Changed

- Release dry-run packaging now builds `./cmd/flux-board`, emits versioned
  platform-tagged artifacts, writes per-artifact and consolidated SHA-256 checksums,
  and validates `VERSION`, tag naming, and changelog coverage before publishing.
