# ADR-008: Compute graph stats and health indicators on demand from the current graph snapshot

## Status
Proposed

## Context

VP2 requires `graph stats` to expose both raw graph counts and cognitive health
indicators such as duplication rate, orphan rate, contradictory fact count, and
density/clustering signals.

The vision also constrains the solution:

- local and offline only
- no human dashboard requirement
- snapshot metrics, not trend analytics
- proportional complexity

## Decision

Compute graph stats on demand from the current graph snapshot inside the local
CLI process.

The stats flow should:

1. load the current graph snapshot from the system of record
2. derive counts and health indicators in process
3. return structured machine-readable metrics without introducing a separate
   metrics backend or time-series store

The required output should include:

- node count
- relationship count
- duplication rate
- orphan rate
- contradictory fact count
- density/clustering indicators

## Consequences

### Positive
- Satisfies the VP2 requirement with minimal new infrastructure
- Keeps stats local, explicit, and easy to invoke before retrieval-heavy work
- Avoids synchronizing a second persistent metrics subsystem
- Aligns well with an agent-facing CLI workflow

### Negative
- Expensive metrics must be computed efficiently to preserve CLI responsiveness
- Historical trends are unavailable without a later product phase
- Indicator definitions need careful tuning so they are meaningful across graph
  sizes and domains

### Risks
- Risk: naive implementations may become slow on larger graphs
  - Mitigation: compute from efficient snapshots, keep metric scope focused, and
    optimize only where actual graph sizes justify it

## Alternatives Considered

### Persisted metrics backend
- Pros: easier historical reporting, potentially faster repeated reads
- Cons: extra storage, synchronization, and architectural complexity
- Rejected because: VP2 requires snapshots only and should remain a simple local
  CLI

### Background metric computation daemon
- Pros: could precompute results continuously
- Cons: adds lifecycle management, implicit behavior, and local operational
  complexity
- Rejected because: the product explicitly favors explicit local CLI workflows
