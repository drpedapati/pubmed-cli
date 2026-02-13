# pubmed-cli

`pubmed-cli` is a command-line interface for NCBI PubMed E-utilities.

It focuses on deterministic, scriptable literature workflows on the `main` branch:
- `search`
- `fetch`
- `cited-by`
- `references`
- `related`
- `mesh`

## Installation

```bash
go install github.com/henrybloomingdale/pubmed-cli/cmd/pubmed@latest
```

Build from source:

```bash
git clone https://github.com/henrybloomingdale/pubmed-cli.git
cd pubmed-cli
go build -o pubmed ./cmd/pubmed
```

## Configuration

Set your NCBI API key (recommended):

```bash
export NCBI_API_KEY="your-key"
```

NCBI rate limits:
- Without key: 3 requests/second
- With key: 10 requests/second

## Quick Start

```bash
# Basic search
pubmed search "fragile x syndrome" --limit 5 --human

# Fetch one PMID
pubmed fetch 38000001 --human --full

# Fetch multiple PMIDs (space or comma-separated)
pubmed fetch 38000001 38000002 --json
pubmed fetch "38000001,38000002" --json

# Citation graph
pubmed cited-by 38000001 --limit 5 --json
pubmed references 38000001 --limit 5 --json
pubmed related 38000001 --limit 5 --human

# MeSH lookup
pubmed mesh "depression" --json
```

## Command Behavior

### Global Flags

| Flag | Description |
|------|-------------|
| `--json` | Structured JSON output |
| `--human`, `-H` | Rich terminal rendering |
| `--csv FILE` | Export current result to CSV |
| `--full` | Show full abstract text (human article output) |
| `--limit N` | Maximum results (must be `> 0`) |
| `--sort` | `relevance`, `date`, or `cited` |
| `--year` | `YYYY` or `YYYY-YYYY` |
| `--type` | Publication-type filter (`review`, `trial`, `meta-analysis`, `randomized`, `case-report`, or custom) |
| `--api-key` | NCBI API key override |

### Input Validation

The CLI now fails fast for common mistakes:
- Invalid `--limit` values (`<= 0`) are rejected.
- Invalid `--sort` values are rejected.
- Invalid year formats and descending ranges are rejected.
- Invalid PMIDs (non-digits) are rejected in `fetch`, `cited-by`, `references`, and `related`.

## Production Reliability Notes

- Shared NCBI client with rate limiting and response-size guards.
- Automatic retry with backoff for transient NCBI `HTTP 429` responses.
- UTF-8 safe text truncation in human output.

## Development

```bash
# Build
go build ./...

# Test
go test ./...

# Vet
go vet ./...
```

## Branching Note

- `main`: non-AI command set listed above.
- `ai-features`: historical branch for AI/LLM workflows.

## License

MIT
