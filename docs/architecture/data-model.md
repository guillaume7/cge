# Data Model — Cognitive Graph Engine VP1 + VP2

## Modeling Strategy

The MVP uses a **flexible entity-centric property graph model**.

Instead of creating a dedicated table for every domain concept, the graph keeps
a small stable core schema and represents domain diversity through entity kinds,
relation kinds, and structured properties.

This is the simplest way to satisfy the vision's wide entity surface while
remaining compatible with Kuzu's typed storage.

VP2 keeps this same model and adds graph-health analysis on top of it rather
than introducing a separate hygiene schema.

## Core Node Shapes

### `Entity`

Represents most first-class graph objects.

Suggested fields:

- `id` — stable unique identifier
- `kind` — e.g. `Skill`, `Prompt`, `Function`, `ADR`, `UserStory`
- `title` — human/machine-readable display label
- `summary` — short compact description
- `content` — optional fuller body or excerpt
- `repo_path` — optional file path when applicable
- `language` — optional source language
- `tags` — normalized tags
- `created_at`
- `updated_at`
- `props_json` — extensible structured properties

### `ReasoningUnit`

Represents an atomic persisted reasoning boundary.

Suggested fields:

- `id`
- `task`
- `summary`
- `agent_id`
- `session_id`
- `timestamp`
- `props_json`

### `AgentSession`

Represents a session-level summary and provenance anchor.

Suggested fields:

- `id`
- `agent_id`
- `started_at`
- `ended_at`
- `summary`
- `repo_root`
- `props_json`

### `GraphRevision`

Represents comparable graph states for diffing.

Suggested fields:

- `id`
- `created_at`
- `created_by`
- `reason`
- `props_json`

## Core Relationship Shapes

Suggested fixed relation kinds:

- `RELATES_TO`
- `PART_OF`
- `DEPENDS_ON`
- `DERIVED_FROM`
- `GENERATED_IN`
- `CITES`
- `ABOUT`
- `SUPERSEDES`
- `CONTRADICTS`
- `CONSOLIDATED_INTO`

Each relationship should support:

- `kind`
- `created_at`
- `created_by`
- `props_json`

## VP2 Hygiene Model

VP2 should not create a second persistence model for graph cleanliness. Instead,
it should derive hygiene suggestions from the existing entity/relationship
snapshot and apply approved changes back into the same graph.

### Duplicate-near-identical analysis

Suggested comparison inputs:

- normalized `kind`
- normalized `title`
- aliases or equivalent keys in `props_json`
- `summary`
- relationship neighborhood overlap where useful

Illustrative suggestion shape:

```json
{
  "action": "consolidate_duplicate_nodes",
  "canonical_id": "adr:ADR-002",
  "duplicates": ["adr:ADR-002-copy"],
  "reasons": ["title_similarity", "matching_alias"],
  "confidence": 0.93
}
```

### Orphan-node analysis

An orphan candidate is a node with insufficient meaningful connectivity for its
kind and role in the graph.

The model should allow:

- structural orphan detection
- exclusions for intentionally isolated nodes if needed
- explicit pruning plans rather than immediate deletion

### Contradiction model

Contradictions are not a separate store. They are detected by comparing graph
facts that appear mutually incompatible.

A contradiction suggestion should capture:

- the conflicting entities or relationships
- the reason they are considered in tension
- the proposed resolution path

Illustrative suggestion shape:

```json
{
  "action": "resolve_contradiction",
  "facts": ["node:fact-a", "node:fact-b"],
  "reason": "same subject with incompatible property values",
  "proposed_resolution": "supersede fact-a with fact-b"
}
```

Applied resolutions may be represented through:

- property updates
- node/edge removal where safe
- explicit `SUPERSEDES`, `CONTRADICTS`, or `CONSOLIDATED_INTO` relationships
- revision metadata that explains the hygiene action

## Provenance Requirements

Every persisted write should be traceable through:

- `agent_id`
- `session_id`
- `timestamp`
- source entity or write reason when available

This provenance is necessary for:

- trusted context retrieval
- `graph explain`
- graph cleanup decisions
- handoff debugging

## Native Payload Contract

The canonical write envelope should be versioned JSON, shared across file I/O
and stdin/stdout.

Illustrative shape:

```json
{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-42",
    "timestamp": "2026-03-21T14:00:00Z"
  },
  "nodes": [],
  "edges": []
}
```

The exact field names can be refined during implementation, but the contract
must remain:

- structured
- stable
- machine-oriented
- versioned
- consistent across stdin, stdout, and files

## Retrieval Model

### Structural Retrieval

Uses graph relations, kinds, tags, and repo-local metadata to identify relevant
starting nodes and neighborhoods.

### Text-Relevance Retrieval

Uses a local text index over:

- titles
- summaries
- content excerpts
- tags
- aliases

### Hybrid Ranking

The retrieval engine should merge:

- structural matches
- text-relevance matches
- provenance-aware confidence signals

The ranked set is then passed into context projection.

## Graph Stats Model

VP2 stats are snapshot-oriented and computed on demand from the current graph.

### Required snapshot counts

- total node count
- total relationship count

### Required health indicators

- duplication rate
- orphan rate
- contradictory fact count
- density / clustering indicators

These indicators are derived values, not durable system-of-record entities.

Illustrative result shape:

```json
{
  "snapshot": {
    "nodes": 540,
    "relationships": 1320
  },
  "indicators": {
    "duplication_rate": 0.08,
    "orphan_rate": 0.03,
    "contradictory_facts": 4,
    "density_score": 0.71,
    "clustering_score": 0.64
  }
}
```

## Context Projection Model

The context envelope should aim to preserve:

- the minimum useful nodes
- critical relationships
- provenance
- a compact summary field for each result

It should avoid:

- dumping raw graph state
- including unrelated neighborhoods
- exhausting the token budget on low-value detail

## State Diff Model

`graph diff` should compare revision anchors and report:

- entities added
- entities updated
- entities removed
- relations added/removed/updated
- kind or tag changes

Because MVP allows rewrite history, revisions are comparison anchors rather than
immutable audit events.

VP2 should continue using revisions for hygiene apply operations so cleanup
remains inspectable through `graph diff`.
