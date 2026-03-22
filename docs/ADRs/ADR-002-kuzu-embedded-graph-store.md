# ADR-002: Use Kuzu as the embedded graph system of record

## Status
Proposed

## Context

The product is explicitly a cognitive graph engine. The storage layer must be a
real graph database, embedded, local, and offline. The graph is the core
product identity and should not be simulated with a generic document or key
value store.

## Decision

Use Kuzu as the embedded graph database and treat it as the system of record
for persisted graph entities, relationships, provenance metadata, and revision
anchors.

## Consequences

### Positive
- Aligns storage choice with the product's graph-native intent
- Keeps the MVP local and offline
- Avoids deploying or depending on a separate graph service
- Enables graph-oriented querying as a first-class capability

### Negative
- Introduces native dependency management complexity
- Requires deliberate schema and migration design
- Limits storage choices to what is practical through Kuzu bindings

### Risks
- Risk: Go integration or migrations may be rougher than expected
  - Mitigation: keep the schema small and use an entity-centric core model in
    MVP

## Alternatives Considered

### SQLite only
- Pros: simpler embedding, easy tooling
- Cons: not a graph database, pushes graph logic into application code
- Rejected because: it contradicts the product requirement for a real graph DB

### Neo4j
- Pros: mature graph ecosystem
- Cons: service deployment, heavier operations, conflicts with offline/local MVP
- Rejected because: it violates the simplest viable local architecture
