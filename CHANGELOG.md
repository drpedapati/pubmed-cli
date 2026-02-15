# Changelog

All notable changes to pubmed-cli are documented in this file.

## [Unreleased]

## [0.5.4] - 2026-02-15

### Added
- `pubmed version` command and branded help footer linking to GitHub/issues.
- Embedded build version support (release builds set `pubmed --version` correctly).
- Linux `arm64` release artifact published alongside `linux/amd64` for Homebrew.
- GitHub repository metadata: CI workflow, release-assets workflow, Dependabot, and issue/PR templates.

### Changed
- Updated release tooling and docs for macOS and Linux Homebrew distribution.

## [0.2.0] - 2026-02-05

### Fixed
- `cited-by`, `references`, and `related` now correctly parse NCBI JSON formats.
- `mesh` lookup uses `esummary` JSON instead of legacy broken parser.

### Changed
- Improved documentation with badges and architecture overview.

## [0.1.1] - 2026-02-05

### Fixed
- Rate limiting now uses `golang.org/x/time/rate` for concurrent behavior.
- NCBI context propagation enables clean cancellation.
- Publication type filters now quote multi-word values safely.
- XML parsing supports date/author edge cases.

### Changed
- Added response-size guards in NCBI clients.

## [0.1.0] - 2026-02-04

### Added
- Initial release with `search`, `fetch`, `cited-by`, `references`, `related`, and `mesh`.
- JSON and human-readable output modes.
- NCBI API key support.
- Rate limiting (3 req/s default, 10 req/s with API key).
- Year and publication type filters.
