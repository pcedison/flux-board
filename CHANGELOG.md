# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

- W9 remains in progress for structured logging, Prometheus metrics, broader browser
  coverage, and exact-head release closure.

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
