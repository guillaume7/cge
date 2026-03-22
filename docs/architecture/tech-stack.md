# Tech Stack — VP1 Cognitive Graph Engine MVP

## Overview

The MVP stack prioritizes:

- local/offline operation
- simple binary distribution
- embedded storage
- strong shell composability
- low operational overhead

All key choices are documented in ADRs and are currently **Proposed** pending
user review.

## Chosen Stack

| Concern | Technology | Rationale | ADR |
|---|---|---|---|
| CLI language | Go 1.22+ | Simple static binaries, strong stdlib, good CLI ergonomics | ADR-001 |
| CLI framework | Cobra | Proven command/subcommand model, help generation, good fit for `graph` UX | ADR-001 |
| Config / flags | Cobra + stdlib | Avoid extra config abstraction for MVP | ADR-001 |
| Graph database | Kuzu embedded DB | Native graph storage, local embedding, no service deployment | ADR-002 |
| Text relevance index | Bleve | Embedded full-text ranking, offline, easy Go integration | ADR-006 |
| Payload format | Versioned JSON | Native machine-readable protocol for stdin/stdout/files | ADR-005 |
| Testing | Go `testing` package | Keep setup simple and standard | ADR-001 |
| Build / dependency management | Go modules | Reproducible builds with `go.mod` and `go.sum` | ADR-001 |

## Versioning Guidance

- Pin direct dependencies in `go.mod`
- Commit `go.sum`
- Revisit dependency upgrades at epic boundaries

## Technology Notes

### Go

Why it fits:

- convenient for CLI development
- easy local binary distribution
- straightforward integration with Unix pipes
- good ecosystem maturity for embedded tools

Trade-off:

- local ML/embedding integration is weaker than Python/Rust ecosystems, so MVP
  retrieval should stay lean

### Cobra

Why it fits:

- fast path to a clean command tree
- standard flag parsing and help output
- easy to test command behavior

Trade-off:

- adds a framework layer for a small CLI, but the ergonomics justify it

### Kuzu

Why it fits:

- embedded graph database aligned with the product identity
- supports local graph queries without external infrastructure
- better fit than stretching a document store into a graph product

Trade-off:

- schema changes need deliberate migration design
- Go integration needs care around native bindings

### Bleve

Why it fits:

- embedded and offline
- practical for MVP text relevance
- complements Kuzu without replacing it

Trade-off:

- does not provide dense vector semantics by itself
- requires a second local persistence mechanism alongside the graph DB

## Rejected Simpler/Heavier Options

### Python

- Better ML ecosystem
- Worse single-binary distribution story
- Rejected because the CLI-first local distribution experience matters more in
  MVP

### Rust

- Strong performance and binary story
- Higher implementation friction for fast MVP iteration
- Rejected because Go is the more pragmatic choice for this product stage

### Neo4j or remote graph service

- Mature ecosystem
- Violates local/offline simplicity
- Rejected because the MVP must remain embedded and repo-scoped

### Dense local embedding stack in MVP

- Stronger semantic retrieval potential
- Higher complexity in binaries, model packaging, and inference runtime
- Rejected for MVP in favor of local BM25/FTS plus graph-aware ranking
