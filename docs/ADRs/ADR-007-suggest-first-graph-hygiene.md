# ADR-007: Use a suggest-first graph hygiene workflow with explicit apply

## Status
Proposed

## Context

VP2 introduces graph hygiene as a first-class capability. The product must help
agents identify duplicate-near-identical nodes, orphan nodes, and contradictory
facts, but it must do so without making the shared graph unsafe or surprising to
mutate.

The vision explicitly calls for:

- manual hygiene workflows
- suggest/apply support
- suggest-only as the safe default
- explicit resolution support for contradictions

## Decision

Implement graph hygiene as a two-phase workflow:

1. **Suggest mode** by default
   - analyze the current graph snapshot
   - return structured duplicate, orphan, and contradiction candidates
   - include explanations and proposed action plans

2. **Apply mode** only when explicitly requested
   - accept a selected hygiene plan
   - execute only explicit approved actions
   - persist the result through the existing graph revision flow

Hygiene actions should support at least:

- duplicate consolidation
- orphan pruning
- contradiction resolution

## Consequences

### Positive
- Preserves trust by making graph mutation explicit
- Fits agent workflows that want reviewable machine-readable suggestions
- Reuses the existing revision and diff foundation for inspectable cleanup
- Keeps the architecture proportional without a background cleanup daemon

### Negative
- Hygiene is slower than fully automatic cleanup because an explicit apply step
  is required
- Suggestion quality must be good enough that agents trust the workflow
- Contradiction resolution semantics may require careful domain-specific tuning

### Risks
- Risk: agents may ignore suggest mode and never clean the graph
  - Mitigation: pair hygiene with lightweight graph stats so graph disorder is
    visible and operationally meaningful

## Alternatives Considered

### Fully automatic hygiene
- Pros: lowest manual effort, graph may stay cleaner continuously
- Cons: unsafe mutations, surprising behavior, harder trust model
- Rejected because: VP2 explicitly requires suggest-first behavior and explicit
  apply semantics

### Manual-only hygiene without machine suggestions
- Pros: simpler to reason about, no heuristic grouping
- Cons: too much burden on agents, weak scalability as graphs grow
- Rejected because: the product goal is not just manual cleanup; it is guided
  cleanup for growing shared memory
