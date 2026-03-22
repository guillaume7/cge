# Architecture Overview — VP1 Cognitive Graph Engine MVP

## System Context

The product is a local, repo-scoped CLI named `graph` that provides a shared
graph memory for AI agents and sub-agents working inside the same repository.

Its core purpose is to improve:

- task continuity across sessions and agent handoffs
- token efficiency by returning compact context instead of large prompt reloads
- trust in retrieved context through provenance and explanation

The MVP is local/offline, chainable in shell pipelines, and optimized for
machine use rather than human graph exploration.

## Key Requirements

### Functional

- initialize a repository-scoped graph store
- accept explicit structured writes from agents
- store reasoning, planning, project, and codebase entities
- support shell-native chaining through stdin/stdout
- retrieve relevant graph data for a task
- project a compact context slice within a token budget
- explain why context or query results were returned
- diff graph states over time
- allow graph cleanup and rewrite workflows

### Non-Functional

- offline/local operation
- simple single-machine setup
- stable machine-readable input/output contract
- trustworthy provenance on writes and retrieval
- proportional MVP complexity

## Architectural Style

The MVP uses a **single-process CLI with embedded storage and a small internal
service layer**.

This is intentionally simple:

- one local binary
- one graph database as the system of record
- one local text-relevance index for retrieval ranking
- no remote services
- no daemon requirement

## High-Level Design

```text
stdin / args / files
        │
        v
  +-------------+
  | graph CLI   |
  | commands    |
  +------+------+ 
         |
         v
  +-------------+      +------------------+
  | payload &   |----->| graph repository |
  | command     |      | manager          |
  | validation  |      +--------+---------+
  +------+------+               |
         |                      v
         |              +---------------+
         |              | Kuzu graph DB |
         |              +---------------+
         |
         v
  +-------------+      +------------------+
  | retrieval   |<---->| local text index |
  | engine      |      | (BM25/FTS)       |
  +------+------+      +------------------+
         |
         +-------------------+
         |                   |
         v                   v
  +-------------+      +-------------+
  | context     |      | explain/diff|
  | projector   |      | services    |
  +------+------+      +-------------+
         |
         v
stdout / files
```

## Request Flows

### `graph init`

1. Detect repo root.
2. Create graph workspace under the repo.
3. Initialize Kuzu storage, metadata, and text index.
4. Record schema version and repository identity.

### `graph write`

1. Read native graph payload from file or stdin.
2. Validate required metadata and payload schema.
3. Upsert entities and relationships into Kuzu.
4. Refresh text index entries for relevant nodes.
5. Emit machine-readable write summary to stdout.

### `graph query`

1. Parse task text from flags or stdin.
2. Retrieve graph candidates through structural traversal.
3. Retrieve text-relevance candidates from the local index.
4. Rank and merge candidates.
5. Return a structured result set with provenance.

### `graph context`

1. Run the same hybrid retrieval pipeline as `graph query`.
2. Expand only the neighborhood needed for continuity.
3. Project results into a token-budgeted context envelope.
4. Return compact machine-readable context for downstream agents.

### `graph explain`

1. Reconstruct retrieval decisions.
2. Show matched terms, graph paths, ranking reasons, and provenance.
3. Return a structured explanation payload.

### `graph diff`

1. Compare two graph states or revisions.
2. Report added, updated, removed, and retagged entities/relations.
3. Return machine-readable change sets.

## Data Ownership

### Graph Repository Manager

Owns:

- repository graph location
- schema version metadata
- initialization state

### Graph Store

Owns:

- entities
- relationships
- provenance metadata
- state revision markers

### Text Index

Owns:

- indexed text projections for retrieval
- searchable aliases, titles, summaries, and content excerpts

### Context Projector

Owns:

- token-budgeted output shaping rules
- result compaction policies

## Key Architectural Decisions

- **ADR-001** — Use Go and Cobra for the CLI implementation
- **ADR-002** — Use Kuzu as the embedded graph system of record
- **ADR-003** — Use repo-local graph storage with a deterministic on-disk layout
- **ADR-004** — Use a flexible entity-centric graph schema with provenance-first
  metadata
- **ADR-005** — Define a versioned native JSON payload contract for stdin,
  stdout, and files
- **ADR-006** — Use a hybrid retrieval pipeline with graph traversal, local text
  relevance ranking, context projection, and explanation

## Why This Architecture Is Proportional

This architecture is deliberately minimal for MVP:

- no networked services
- no distributed components
- no background process
- no separate API layer
- no human visualization subsystem

The only complexity retained is complexity that directly serves the product
value:

- graph persistence
- retrieval quality
- provenance
- chainability

## Open Review Focus

Before planning themes and stories, the most important review points are:

1. the choice of a local BM25/FTS retrieval index for MVP semantic recall
2. the flexible entity-centric schema shape in Kuzu
3. the proposed native JSON payload contract as the agent/tool protocol
