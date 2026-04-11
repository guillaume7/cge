# ADR-004: Use a flexible entity-centric graph schema with provenance-first metadata

## Status
Proposed

## Context

The vision expects the graph to represent many classes of knowledge:
reasoning units, sessions, prompts, skills, instructions, plans, ADRs, backlog
artifacts, and codebase entities. A rigid table-per-domain model would create a
large up-front schema burden and slow MVP iteration.

At the same time, the product requires provenance, trust, and explanation.

## Decision

Use a small core graph schema centered on:

- `Entity`
- `ReasoningUnit`
- `AgentSession`
- `GraphRevision`

Represent broad domain diversity through:

- entity `kind`
- relation `kind`
- structured JSON properties
- required provenance metadata on writes

## Consequences

### Positive
- Keeps the initial Kuzu schema small and adaptable
- Supports many domain concepts without schema explosion
- Preserves provenance as a first-class retrieval concern
- Makes graph cleanup and evolution easier in MVP

### Negative
- Less compile-time rigidity than a deeply specialized schema
- More validation burden moves into application logic
- Query conventions must be disciplined to avoid kind drift

### Risks
- Risk: inconsistent kind naming could reduce retrieval quality
  - Mitigation: define canonical entity and relation kind vocabularies in code

## Alternatives Considered

### Dedicated node table per domain concept
- Pros: tighter typing, clearer per-domain queries
- Cons: large schema surface, brittle MVP evolution
- Rejected because: the entity surface is too broad for an MVP-first design

### Pure generic entity/edge model with no special nodes
- Pros: maximal flexibility
- Cons: loses useful structure for reasoning units, sessions, and revisions
- Rejected because: provenance and diffing deserve explicit anchors
