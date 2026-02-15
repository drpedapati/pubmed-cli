# Usability Testing Report (Production Gate)

Date: 2026-02-13
Branch: `main`
Binary under test: local build from `./cmd/pubmed`

## Scope

Manual real-user CLI flows were executed for:
- First-run discoverability (`pubmed`, `pubmed --help`)
- Happy paths (`search`, `fetch`, `cited-by`, `references`, `related`, `mesh`)
- Data export (`--json`, `--csv`, `--ris`, `--human`, `--full`)
- Error handling (invalid PMIDs, invalid flags, malformed year/sort)

## Critical Findings

1. Runtime panic on invalid limit for link commands
- Repro: `pubmed related 38000001 --human --limit -1`
- Observed: process panic (`makeslice: len out of range`)
- Severity: Critical
- Status: Fixed

## Additional Findings

1. Weak CLI flag validation
- Invalid `--sort` and malformed `--year` were not blocked early.
- Severity: High
- Status: Fixed with centralized global validation.

2. Non-digit PMID coercion risk
- Inputs like `abc123` could resolve unexpectedly through downstream behavior.
- Severity: High
- Status: Fixed with strict PMID validation.

3. UTF-8 truncation risk in human output
- Byte-based truncation could split multibyte runes.
- Severity: Medium
- Status: Fixed with rune-safe truncation.

4. Citation-manager export gap
- Missing first-class RIS export for EndNote/Zotero workflows.
- Severity: High (for research ops workflows)
- Status: Fixed with `--ris FILE` on `fetch`, `cited-by`, `references`, and `related`.

## Current User Experience (Post-Fix)

What now works well:
- Invalid `--limit` returns actionable error instead of crashing.
- Invalid `--sort` and invalid `--year` fail fast with explicit guidance.
- Invalid PMIDs fail fast on command boundary.
- Core command flows are stable and consistent across JSON/human/CSV modes.
- Core command flows are stable and consistent across JSON/human/CSV/RIS modes.
- Comma-separated and space-separated PMID inputs both work for `fetch`.
- RIS export can be requested alongside stdout output modes.

## Commands Executed (Representative)

```bash
pubmed --help
pubmed search "autism" --limit 2 --human
pubmed search "autism" --sort newest
pubmed search "autism" --year 2025-2020
pubmed fetch "38000001, 38000002" --json
pubmed fetch "38000001, 38000002" --ris /tmp/fetch.ris
pubmed fetch abc123
pubmed cited-by 38000001 --limit 2 --human
pubmed references 38000001 --limit 5 --ris /tmp/references.ris
pubmed related 38000001 --human --limit -1
pubmed search autism --ris /tmp/search.ris
pubmed mesh depression --ris /tmp/mesh.ris
pubmed mesh depression --human
```

## Recommendation

Release-ready for production use on the current non-AI command set, with the caveat that live NCBI rate-limiting behavior should continue to be monitored in operational usage.
