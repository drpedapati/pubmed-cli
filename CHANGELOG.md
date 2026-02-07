# Changelog

All notable changes to pubmed-cli will be documented in this file.

## [0.5.1] - 2026-02-07

### Fixed
- **Codex 5.3 code review fixes** across wizard.go, synth.go, engine.go
- Nil-pointer guards throughout
- Context cancellation respected during LLM scoring
- UTF-8 safe text truncation
- APA formatting for collective/group authors
- Clear error messages when API key missing or all scoring fails
- DOCX fallback to markdown when pandoc unavailable
- Multi-word queries work without quotes

## [0.5.0] - 2026-02-07

### Added
- **`pubmed wizard`** — Beautiful interactive synthesis wizard
  - Step-by-step prompts with sensible defaults
  - Progress spinner during synthesis
  - Cross-platform config storage (`~/.config/pubmed-cli/config.json`)
  - Outputs to `~/Documents/PubMed Syntheses/` by default
- **`pubmed config`** — Configuration management
  - `config show` — View current settings
  - `config set` — Interactive editor
  - `config reset` — Reset to defaults
- Uses Charm's `huh` library for beautiful terminal forms

### Changed
- README reorganized with wizard as primary entry point

## [0.4.0] - 2026-02-07

### Added
- **`pubmed synth` command** — Literature synthesis with citations
  - Searches PubMed and scores papers for relevance using LLM
  - Filters to top relevant papers (configurable threshold)
  - Synthesizes findings into cohesive paragraphs with inline citations
  - Outputs: Markdown (default), Word document (`--docx`), RIS (`--ris`), JSON (`--json`)
  - Single paper deep dive with `--pmid`
  - Configurable paper count, word count, relevance threshold
  - Token usage tracking
- RIS export for reference manager import (EndNote, Zotero, Mendeley)
- APA citation formatting

### Changed
- `pubmed qa` clarified as yes/no benchmark tool
- README reorganized with `synth` as primary research command

## [0.3.0] - 2026-02-07

### Added
- **`pubmed qa` command** — Answer biomedical yes/no questions with adaptive retrieval
  - Novelty detection (scans for 2024+ year patterns and recency keywords)
  - Confidence-gated retrieval (only fetches from PubMed when model is uncertain)
  - Abstract minification (extracts key sentences, ~74% token savings)
  - `--explain` flag for reasoning trace with sources
  - `--retrieve` / `--parametric` flags to force strategy
  - `--claude` flag for Claude CLI integration
- LLM client abstraction (`internal/llm/`) supporting OpenAI-compatible APIs
- Adaptive retrieval engine (`internal/qa/`) with configurable confidence threshold
- `make release` target for cross-compilation

### Changed
- README completely rewritten with comprehensive documentation
- Added Homebrew installation instructions

## [0.2.0] - 2026-02-05

### Fixed
- ELink commands (`cited-by`, `references`, `related`) now correctly parse NCBI JSON format
- MeSH lookup uses `esummary` JSON instead of broken `efetch` text parser

### Changed
- Improved documentation with badges and architecture section

## [0.1.1] - 2026-02-05

### Fixed
- Rate limiter now uses `golang.org/x/time/rate` for correct concurrent behavior
- MeSH client shares rate limiter with eutils (NCBI compliance)
- Response size limits to prevent memory exhaustion
- XML parsing handles `MedlineDate`, `CollectiveName`, and nested tags
- Context propagation from CLI commands (enables Ctrl-C cancellation)
- Publication type filters properly quoted for multi-word types

## [0.1.0] - 2026-02-04

### Added
- Initial release
- 6 commands: `search`, `fetch`, `cited-by`, `references`, `related`, `mesh`
- JSON and human-readable output modes
- NCBI API key support
- Rate limiting (3 req/s default, 10 with API key)
- Year and publication type filters
