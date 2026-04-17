# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

- Future work can build on the current observability, release, and extension seams
  without reopening the initial public-fork baseline.

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
