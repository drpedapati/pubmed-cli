<p align="center">
  <h1 align="center">🔬 pubmed-cli</h1>
  <p align="center">
    <strong>PubMed from your terminal. Built for humans and AI agents.</strong>
  </p>
  <p align="center">
    <a href="https://github.com/henrybloomingdale/pubmed-cli/releases/latest"><img src="https://img.shields.io/badge/version-0.5.0-blue?style=flat-square" alt="v0.5.0"></a>
    <img src="https://img.shields.io/badge/go-1.25-00ADD8?style=flat-square&logo=go" alt="Go 1.25">
    <img src="https://img.shields.io/badge/license-MIT-green?style=flat-square" alt="MIT License">
  </p>
</p>

---

Search PubMed, fetch abstracts, traverse citation networks, answer biomedical questions, and look up MeSH terms — all from the command line. Outputs structured JSON for piping into scripts, dashboards, or LLM tool-use loops.

## ✨ Features

- **Interactive wizard** — beautiful step-by-step synthesis with progress spinner
- **Literature synthesis** — search, filter by relevance, synthesize with citations
- **Multiple outputs** — Markdown, Word (.docx), RIS (for reference managers), JSON
- **Persistent config** — save your defaults, works across sessions
- **LLM integration** — works with OpenAI, Anthropic, or any OpenAI-compatible API
- **Rate-limited** — respects NCBI guidelines (3 req/s default, 10 with API key)
- **Zero dependencies** — single static binary, ~5ms startup
- **10 commands** — wizard, synth, search, fetch, cited-by, references, related, mesh, qa, config

## 📦 Installation

All methods install the `pubmed` command — a single binary with multiple subcommands.

### Homebrew (recommended)

```bash
brew tap henrybloomingdale/tools
brew install pubmed-cli

# Verify
pubmed --help
```

### Go install

```bash
go install github.com/henrybloomingdale/pubmed-cli/cmd/pubmed@latest
```

### Build from source

```bash
git clone https://github.com/henrybloomingdale/pubmed-cli.git
cd pubmed-cli
go build -o pubmed ./cmd/pubmed
```

### What you get

One command, ten subcommands:

```
pubmed wizard    # Interactive synthesis wizard ✨
pubmed synth     # Synthesize literature with citations
pubmed search    # Search PubMed
pubmed fetch     # Get article details
pubmed cited-by  # Find citing papers
pubmed references # Find referenced papers
pubmed related   # Find similar papers
pubmed mesh      # Look up MeSH terms
pubmed qa        # Answer yes/no questions (benchmark)
pubmed config    # Manage wizard settings
```

## ⚙️ Configuration

### NCBI API Key (recommended)

Without a key you're limited to 3 requests/second. With one, you get 10. Free at [ncbi.nlm.nih.gov/account/settings](https://www.ncbi.nlm.nih.gov/account/settings/).

```bash
export NCBI_API_KEY="your-key-here"
```

### LLM API (for `qa` command)

The `qa` command uses an LLM for answering questions. Three options:

#### Option 1: OpenAI API

```bash
export LLM_API_KEY="sk-..."
export LLM_MODEL="gpt-4o"  # optional, defaults to gpt-4o
```

#### Option 2: Any OpenAI-compatible API

```bash
export LLM_BASE_URL="https://api.example.com/v1"
export LLM_API_KEY="your-key"
export LLM_MODEL="your-model"
```

#### Option 3: Claude CLI (no API key needed)

```bash
pubmed qa --claude "your question"
```

The `--claude` flag uses a unique integration: instead of calling the Anthropic API directly, it shells out to the [Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code) (`claude` binary). The CLI handles OAuth authentication internally via your Anthropic account — no `ANTHROPIC_API_KEY` required.

This approach:
- **No API key management** — Uses your existing Claude Code authentication
- **Respects CLI rate limits** — Anthropic's CLI handles quotas
- **Works with Max subscriptions** — If you have Claude Code access, it just works

Install Claude Code CLI: `npm install -g @anthropic-ai/claude-code`

## 🚀 Commands

### wizard — Interactive synthesis wizard ✨

The easiest way to create a literature synthesis. Beautiful step-by-step interface with sensible defaults.

```bash
pubmed wizard
```

Walks you through:
1. Enter your research question
2. Set paper count and word length (or accept defaults)
3. Choose output format (Word + RIS, Markdown, JSON)
4. Watch the synthesis happen with a progress spinner
5. Get your files saved to `~/Documents/PubMed Syntheses/`

**Configure defaults:**
```bash
pubmed config show    # View current settings
pubmed config set     # Interactive editor
pubmed config reset   # Reset to defaults
```

Config is stored in `~/.config/pubmed-cli/config.json` (cross-platform).

### synth — Synthesize literature with citations

The main research tool. Searches PubMed, scores papers for relevance, and synthesizes findings into paragraphs with proper citations.

```bash
# Basic synthesis — outputs markdown
pubmed synth "SGLT-2 inhibitors in liver fibrosis"

# Word document + RIS file for reference managers
pubmed synth "CBT for pediatric anxiety" --docx review.docx --ris refs.ris

# More papers, longer output
pubmed synth "autism biomarkers" --papers 10 --words 500

# Single paper deep dive
pubmed synth --pmid 41234567 --words 400

# JSON for agents
pubmed synth "treatments for fragile x" --json
```

**How it works:**

1. **Search** — Queries PubMed for relevant papers (default: 30)
2. **Score** — LLM rates each paper's relevance to your question (1-10)
3. **Filter** — Keeps papers above threshold (default: ≥7)
4. **Synthesize** — Generates cohesive paragraphs with inline citations
5. **Export** — Outputs markdown, Word doc, RIS, or JSON

**Output includes:**
- Synthesis paragraph(s) with inline citations (APA style)
- Numbered reference list with PMIDs and DOIs
- Token usage statistics
- RIS file for EndNote/Zotero/Mendeley import

| Flag | Default | Description |
|------|---------|-------------|
| `--papers N` | 5 | Papers to include in synthesis |
| `--search N` | 30 | Papers to search before filtering |
| `--relevance N` | 7 | Minimum relevance score (1-10) |
| `--words N` | 250 | Target word count |
| `--docx FILE` | — | Output Word document |
| `--ris FILE` | — | Output RIS for reference managers |
| `--pmid ID` | — | Deep dive on single paper |
| `--json` | — | Structured JSON output |
| `--claude` | — | Use Claude CLI (no API key) |

### qa — Answer biomedical questions

The `qa` command uses **adaptive retrieval**: it detects when a question requires recent literature (post-training knowledge) and retrieves from PubMed only when necessary.

```bash
# Basic question — model decides whether to retrieve
pubmed qa "Does CBT help hypertension-related anxiety?"
# Output: yes

# Show reasoning and sources
pubmed qa --explain "Is metformin effective for PCOS?"
# 🧠 Answer: YES
#    Strategy: parametric
#    Confidence: 9/10

# Novel knowledge — always retrieves
pubmed qa --explain "According to 2025 studies, does SGLT-2 reduce liver fibrosis?"
# 🔍 Answer: YES
#    Strategy: retrieval
#    Novel knowledge detected: yes
#    Sources: 41234567, 41234568, 41234569

# Force retrieval (never trust parametric)
pubmed qa --retrieve "Does aspirin prevent colorectal cancer?"

# JSON output for pipelines
pubmed qa --json "Is there evidence for gut-brain axis in autism?"
```

**How adaptive retrieval works:**

1. **Novelty detection** — Scans for year patterns (2024+) or recency keywords ("recent study", "latest research"). If detected, always retrieves.
2. **Confidence check** — For established knowledge, asks the model its confidence (1-10). Default threshold: 7.
3. **Smart retrieval** — If confidence is below threshold, searches PubMed and augments with evidence.
4. **Minification** — Extracts key sentences (results, conclusions, statistics) from abstracts to reduce tokens by ~74%.

| Flag | Description |
|------|-------------|
| `--explain`, `-e` | Show reasoning, strategy, confidence, sources |
| `--json` | Structured JSON output |
| `--retrieve` | Force retrieval (skip confidence check) |
| `--parametric` | Force parametric (never retrieve) |
| `--confidence N` | Confidence threshold (default: 7) |
| `--model` | LLM model name |
| `--llm-url` | LLM API base URL |
| `--claude` | Use Claude via CLI OAuth |

### search — Search PubMed

```bash
# Basic search
pubmed search "fragile x syndrome"

# With filters
pubmed search "ADHD treatment" --type review --year 2020-2025 --limit 10

# MeSH terms
pubmed search '"fragile x syndrome"[MeSH] AND "electroencephalography"[MeSH]'

# JSON for scripting
pubmed search "autism biomarkers" --json | jq '.ids[]'

# Rich terminal output
pubmed search "CRISPR therapy" --human
```

### fetch — Get article details

```bash
# Single article
pubmed fetch 38123456

# Multiple articles
pubmed fetch 38123456 37987654 37876543

# JSON with jq
pubmed fetch 38123456 --json | jq '{title: .title, doi: .doi}'
```

### cited-by — Who cited this paper?

```bash
pubmed cited-by 38123456
pubmed cited-by 38123456 --json | jq '.citing_ids'
```

### references — What does this paper cite?

```bash
pubmed references 38123456
```

### related — Find similar articles

```bash
pubmed related 38123456
```

### mesh — Look up MeSH terms

```bash
pubmed mesh "Fragile X Syndrome"
pubmed mesh "Electroencephalography" --json
```

## 🤖 Agent Tool Use

This CLI is designed as a tool for LLM agents. Rather than building a RAG pipeline with embeddings and vector databases, give your agent direct access to PubMed:

```python
# Define tools for your agent
tools = [
    {
        "name": "pubmed_qa",
        "description": "Answer biomedical yes/no questions with evidence from PubMed",
        "exec": "pubmed qa --json '{question}'"
    },
    {
        "name": "pubmed_search",
        "description": "Search PubMed for articles",
        "exec": "pubmed search --json '{query}'"
    },
    {
        "name": "pubmed_fetch",
        "description": "Get full article details by PMID",
        "exec": "pubmed fetch --json {pmid}"
    },
    {
        "name": "pubmed_cited_by",
        "description": "Find papers that cite a given paper",
        "exec": "pubmed cited-by --json {pmid}"
    }
]
```

**Why agentic tool use beats RAG:**

| Approach | How it works | Limitation |
|----------|--------------|------------|
| RAG | Embed corpus → vector search → retrieve similar | Retrieves what's *similar*, not what's *relevant* |
| Agentic | LLM decides what to search → fetches → reasons | Retrieves what's *needed* for the question |

The `qa` command implements **confidence-gated adaptive retrieval**: the model only retrieves when it's uncertain, avoiding unnecessary API calls for well-established knowledge while ensuring accuracy on novel or obscure topics.

## 📋 Global Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--json` | Structured JSON output | `false` |
| `--human`, `-H` | Rich terminal display | `false` |
| `--limit N` | Max results | `20` |
| `--sort` | `relevance` \| `date` \| `cited` | `relevance` |
| `--year` | Year range (e.g. `2020-2025`) | — |
| `--type` | `review` \| `trial` \| `meta-analysis` | — |
| `--api-key` | NCBI API key | `$NCBI_API_KEY` |
| `--csv` | Export to CSV file | — |

## 🏗️ Architecture

```
pubmed-cli/
├── cmd/pubmed/           # CLI entry point (Cobra)
│   ├── main.go           # Root command + search/fetch/mesh/link
│   ├── wizard.go         # Interactive synthesis wizard (huh)
│   ├── config.go         # Configuration management
│   ├── synth.go          # Synthesis command
│   └── qa.go             # QA benchmark command
├── internal/
│   ├── eutils/           # NCBI E-utilities client
│   ├── llm/              # LLM client (OpenAI + Claude CLI)
│   ├── synth/            # Literature synthesis engine
│   ├── qa/               # Adaptive retrieval for yes/no
│   ├── mesh/             # MeSH descriptor lookup
│   └── output/           # Formatters
└── go.mod

Config: ~/.config/pubmed-cli/config.json (cross-platform)
Output: ~/Documents/PubMed Syntheses/ (configurable)
```

## 🧪 Development

```bash
go build -o pubmed ./cmd/pubmed   # Build
go test ./...                      # Run tests
go test -race ./...                # Race detection
```

## 📄 License

MIT

## 🙏 Acknowledgments

- Built on [NCBI E-utilities](https://www.ncbi.nlm.nih.gov/books/NBK25501/)
- Inspired by the limitations of RAG for biomedical QA

---

<p align="center">
  <sub>Made with 🧬 for biomedical research</sub>
</p>
