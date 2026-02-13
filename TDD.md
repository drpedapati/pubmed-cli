# Technical Design Document: pubmed-cli (Non-AI Mainline)

This document tracks current architecture for the `main` branch after the AI feature rollback.

## Current Command Surface

- `search` — NCBI search
- `fetch` — retrieve article and abstract payloads
- `cited-by` — get citing papers
- `references` — get paper references
- `related` — get related work
- `mesh` — MeSH term lookup

## Scope and Constraints

- No AI synthesis, QA, or wizard command paths are included on `main`.
- This branch must remain NCBI-data focused and deterministic for scriptable output.

## Package Layout (Current)

- `cmd/pubmed`: Cobra CLI entrypoint and command handlers.
- `internal/eutils`: NCBI client adapters for search/fetch/link APIs.
- `internal/mesh`: MeSH lookup helpers.
- `internal/output`: Output formatters (`json`, `human`, `csv`).
- `internal/ncbi`: Shared NCBI client primitives and response guardrails.

## Reliability Criteria

- Search/filter semantics match NCBI API behavior.
- Fetch and link traversals handle API errors without crashes.
- JSON schema remains stable for scripted consumption.
- Output size guards protect against malformed or unexpectedly large responses.

## Historical Note

AI/LLM work existed in prior history and is now maintained on the `ai-features` branch.
