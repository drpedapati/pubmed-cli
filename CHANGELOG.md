# Changelog

All notable changes to pubmed-cli will be documented in this file.

## [Unreleased]

### Changed
- Mainline command set has been restored to the non-AI feature set: `search`, `fetch`, `cited-by`, `references`, `related`, `mesh`.
- AI-only features (`synth`, `wizard`, `qa`) were removed from `main` and are tracked on the `ai-features` branch.

## [0.2.0] - 2026-02-05

### Fixed
- `cited-by`, `references`, and `related` now correctly parse NCBI JSON formats.
- `mesh` lookup uses `esummary` JSON instead of legacy broken parser.

### Changed
- Improved documentation with badges and architecture overview.

## [0.1.1] - 2026-02-05

### Fixed
- Rate limiting now uses `golang.org/x/time/rate` for concurrent behavior.
- NCBI `Context` propagation enables clean cancellation.
- Publication type filters now quote multi-word values safely.
- XML parsing supports date/author edge cases.

### Changed
- Added response size guards in NCBI clients.

## [0.1.0] - 2026-02-04

### Added
- Initial release with `search`, `fetch`, `cited-by`, `references`, `related`, and `mesh`.
- JSON and human-readable output modes.
- NCBI API key support.
- Rate limiting (3 req/s default, 10 req/s with API key).
- Year and publication type filters.
