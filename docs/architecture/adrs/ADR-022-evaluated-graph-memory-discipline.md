# ADR-022: Evaluated graph memory discipline

## Status
Proposed

## Context

The VP8 vision identifies graph store divergence as one of two core product
failures:

> The graph store can diverge and become less reliable over time.

Through VP1–VP7, graph writes followed a simple model: any write that passes
payload validation and provenance requirements is persisted. The graph is
treated as trustworthy once written. Hygiene workflows (ADR-007) can detect
problems after the fact, but nothing prevents stale, contradictory, or
low-value data from entering the graph in the first place.

VP8 reframes the graph as "one subsystem among several equal or near-equal
parts" and requires that memory writes be evaluated before they are trusted:

> **write** only when the new state is strong enough to improve future
> continuity.

This ADR establishes the discipline for graph memory under the VP8 evaluator
loop.

## Decision

Adopt **evaluated graph memory discipline**:

1. **Writes pass through the evaluator loop**: any memory write that originates
   from the evaluator loop (e.g. after a `workflow finish` or an explicit
   memory-update request) must be scored by the Context Evaluator (ADR-018)
   before the write is committed to the graph store.

2. **Write-confidence threshold**: the Decision Engine (ADR-019) applies a
   write threshold. If the evaluator's confidence that the new state improves
   future retrieval quality is below the threshold, the write is deferred or
   skipped. A deferred write may be retried with additional evidence; a
   skipped write is recorded in the attribution log (ADR-021) with the reason.

3. **Retrieval-time down-ranking**: stale or low-confidence graph state can be
   down-ranked during retrieval without being rewritten or deleted. The
   evaluator can flag entities as low-confidence during scoring, and the
   retrieval engine can deprioritize them. This preserves the graph as a
   record while reducing its influence on current tasks.

4. **Compatibility with existing writes**: direct `graph write` commands (the
   VP1 write path) continue to work without mandatory evaluation. Evaluated
   writes are the discipline for the VP8 control loop; the raw write primitive
   remains available for explicit agent use. This avoids breaking existing
   consumers while making the evaluated path the recommended default for
   workflow-mediated memory updates.

5. **Compatibility with hygiene workflows**: existing hygiene detection and
   apply workflows (ADR-007) continue to operate. Evaluated write discipline
   reduces future hygiene burden by preventing low-quality writes from entering
   the graph, but it does not replace post-hoc hygiene analysis.

6. **No graph schema change**: this decision adds evaluation logic around the
   write path, not new graph node or relationship types. The Entity, relation,
   and provenance schemas (ADR-004) are unchanged.

## Consequences

### Positive
- Graph reliability improves because low-quality writes are caught before
  persistence
- Retrieval quality improves because stale state is deprioritized without
  requiring manual cleanup
- Memory updates carry attribution that explains why the write was approved or
  skipped
- The graph remains important as supporting memory without dominating the
  product

### Negative
- Some useful memory updates may be deferred or skipped if the evaluator is too
  conservative
- The write path becomes slower because evaluation adds a step
- Developers must understand that raw writes and evaluated writes coexist

### Risks
- Risk: overly strict write thresholds cause memory starvation, reducing
  continuity
  - Mitigation: lab experiments should track write approval rates; if useful
    context is being deferred too often, thresholds should be relaxed
- Risk: down-ranking without deletion makes the graph grow indefinitely with
  low-value state
  - Mitigation: hygiene workflows (ADR-007) can still prune state that the
    evaluator flags as persistently low-confidence
- Risk: the raw write primitive is used to bypass evaluation discipline
  - Mitigation: workflow-mediated writes (the recommended VP8 path) go through
    evaluation by default; raw writes are an explicit opt-out, not the default

## Alternatives Considered

### Require evaluation for all writes including raw `graph write`
- Pros: uniform write discipline
- Cons: breaks backward compatibility, makes simple writes heavyweight
- Rejected because: the raw write primitive must remain available for explicit
  agent use; VP8 should make evaluated writes the default for the control loop,
  not the only path

### Delete stale graph state automatically during evaluation
- Pros: keeps the graph clean
- Cons: irreversible; useful context may be removed prematurely
- Rejected because: down-ranking is safer than deletion; deletion should remain
  an explicit hygiene action

### Abandon the graph and use only session-local memory
- Pros: eliminates drift by eliminating long-lived state
- Cons: destroys session-to-session continuity, which is a core VP8 goal
- Rejected because: VP8 wants the graph as a disciplined subsystem, not no
  graph at all
