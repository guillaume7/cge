# Data Model — VP1 Cognitive Graph Engine MVP

## Modeling Strategy

The MVP uses a **flexible entity-centric property graph model**.

Instead of creating a dedicated table for every domain concept, the graph keeps
a small stable core schema and represents domain diversity through entity kinds,
relation kinds, and structured properties.

This is the simplest way to satisfy the vision's wide entity surface while
remaining compatible with Kuzu's typed storage.

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

Each relationship should support:

- `kind`
- `created_at`
- `created_by`
- `props_json`

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
