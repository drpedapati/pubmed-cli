# Changelog

All notable changes to pubmed-cli are documented in this file.

## [Unreleased]

### Fixed
- Prevented runtime panic when link commands are run with invalid limits (for example `--limit -1`).
- Added global input validation for:
  - `--limit` (must be greater than 0)
  - `--sort` (must be one of `relevance`, `date`, `cited`)
  - `--year` (must be `YYYY` or `YYYY-YYYY` with ascending range)
- Added strict PMID validation for `fetch`, `cited-by`, `references`, and `related`.
- Added support for robust comma-separated PMID parsing in `fetch`.
- Made human-output truncation UTF-8 safe.
- Added retry/backoff handling for transient NCBI `HTTP 429` responses.

### Changed
- Refreshed all project documentation for production release on non-AI `main` branch.

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
