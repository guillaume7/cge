# ADR-003: Use a repo-local graph workspace

## Status
Proposed

## Context

The vision requires the graph to be scoped per repository and shared by agents
working locally on one machine. The product should feel like part of the repo's
working memory, not a global machine-wide knowledge pool.

## Decision

Store graph state in a deterministic repo-local workspace, recommended as
`.graph/`, containing:

- graph database files
- derived text index files
- local graph configuration and schema version metadata

## Consequences

### Positive
- Preserves strict repo scoping
- Makes graph state easy to discover, back up, and reset
- Keeps the mental model aligned with repository-local work
- Simplifies repo root detection and graph initialization

### Negative
- Adds local repository files that need ignore/review conventions
- Prevents cross-repo sharing by default
- Couples graph lifetime to repo workspace lifecycle

### Risks
- Risk: teams may want alternate paths or ignored storage
  - Mitigation: allow configurable path override while keeping repo-local
    defaults

## Alternatives Considered

### Global graph directory in the user home
- Pros: simpler sharing across repos
- Cons: breaks repo scoping, risks context leakage between repositories
- Rejected because: repo-local isolation is a product requirement

### Background graph service
- Pros: centralization, possible future multi-client access
- Cons: operational complexity, not required for MVP
- Rejected because: it over-architects a local single-machine tool
