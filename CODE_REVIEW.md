# Code Review (Production Readiness)

Date: 2026-02-13
Scope: `main` branch, non-AI command set

## Findings First

1. Critical: negative `--limit` could panic link commands
- Area: `cmd/pubmed/main.go`
- Repro: `pubmed related <pmid> --human --limit -1`
- Impact: hard process crash
- Resolution: Added centralized global flag validation and blocked non-positive limits.

2. High: missing guardrails for global flags
- Area: `cmd/pubmed/main.go`
- Impact: unclear runtime behavior for unsupported `--sort` and malformed `--year`
- Resolution: Added strict pre-run validation for `--sort` and `--year`.

3. High: invalid PMID input not rejected at boundary
- Area: `cmd/pubmed/main.go`
- Impact: user error could lead to confusing downstream behavior
- Resolution: Added strict numeric PMID validation and normalization.

4. Medium: UTF-8 unsafe truncation in human output
- Area: `internal/output/human.go`
- Impact: potential broken glyph output on truncation boundaries
- Resolution: Switched to rune-safe truncation logic.

5. High: no native RIS export for reference-manager workflows
- Area: `cmd/pubmed/main.go`, `internal/output`
- Impact: users had no direct EndNote/Zotero import path on `main`
- Resolution: added `--ris FILE` export for `fetch`, `cited-by`, `references`, and `related`, with conservative RIS tags for broad compatibility.

## Verification

- `go test ./...` passed.
- `go vet ./...` passed.
- Manual CLI smoke tests passed across all six commands.

## Residual Risks

- NCBI service-level variability (`429`, transient network errors) still depends on external API behavior; retry logic reduces but cannot remove this class of failure.

## Overall Assessment

No remaining code-level blockers identified for production release of the current non-AI `main` branch command surface.
