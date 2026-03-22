# Architecture Overview — Cognitive Graph Engine VP1 + VP2

## System Context

The product is a local, repo-scoped CLI named `graph` that provides a shared
graph memory for AI agents and sub-agents working inside the same repository.

Its core purpose is to improve:

- task continuity across sessions and agent handoffs
- token efficiency by returning compact context instead of large prompt reloads
- trust in retrieved context through provenance and explanation
- long-term graph maintainability through hygiene and graph-health visibility

The product remains local/offline, chainable in shell pipelines, and optimized
for machine use rather than human graph exploration.

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
- compute graph-health snapshots
- suggest graph hygiene actions for duplicates, orphans, and contradictions
- apply explicit hygiene plans and record them as graph revisions

### Non-Functional

- offline/local operation
- simple single-machine setup
- stable machine-readable input/output contract
- trustworthy provenance on writes and retrieval
- proportional MVP complexity
- safe-by-default graph mutation for hygiene workflows

## Architectural Style

The product uses a **single-process CLI with embedded storage and a small
internal service layer**.

This is intentionally simple:

- one local binary
- one graph database as the system of record
- one local text-relevance index for retrieval ranking
- one in-process graph analysis layer for stats and hygiene
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
          +-------------------+--------------------+
          |                   |                    |
          v                   v                    v
   +-------------+      +-------------+     +--------------+
   | context     |      | explain/diff|     | stats &      |
   | projector   |      | services    |     | hygiene      |
   +------+------+      +-------------+     | analyzers    |
          |                                 +------+-------+
          |                                        |
          +----------------------------------------+
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

### `graph stats`

1. Open the current repo-local graph snapshot.
2. Compute structural counts and health indicators on demand.
3. Return machine-readable snapshot metrics for agent decisions.

### `graph hygiene`

1. Analyze the current graph for duplicate-near-identical nodes, orphan nodes,
   and contradictory facts.
2. In suggest mode, return structured candidate actions and explanations.
3. In apply mode, execute only explicit requested actions.
4. Persist the resulting graph changes through the normal revision flow.

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

### Stats and Hygiene Analyzers

Own:

- graph-health metrics
- duplicate/orphan/contradiction detection logic
- suggested hygiene action plans

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
- **ADR-007** — Use a suggest-first graph hygiene workflow with explicit apply
- **ADR-008** — Compute graph stats and health indicators on demand from the
  current graph snapshot

## Why This Architecture Is Proportional

This architecture is deliberately minimal for MVP:

- no networked services
- no distributed components
- no background process
- no separate API layer
- no human visualization subsystem
- no dedicated metrics backend

The only complexity retained is complexity that directly serves the product
value:

- graph persistence
- retrieval quality
- provenance
- chainability
- graph maintainability

## Open Review Focus

Before planning themes and stories, the most important review points are:

1. the boundary between suggest-only hygiene analysis and explicit apply flows
2. the on-demand computation model for graph stats and health indicators
3. the reuse of the existing entity-centric graph model for contradiction and
   consolidation workflows
