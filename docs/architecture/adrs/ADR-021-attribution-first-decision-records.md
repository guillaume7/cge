# ADR-021: Attribution-first decision records

## Status
Proposed

## Context

VP6 and VP7 introduced inclusion reasons for kickoff entities (ADR-016,
ADR-017), which was the product's first step toward explaining why context was
injected. However, attribution coverage stops at the kickoff boundary. The
product cannot currently explain:

- why a context bundle was narrowed or rejected during evaluation
- why a candidate output was accepted, sent back for revision, or suppressed
- why a memory write was approved, deferred, or skipped
- what evaluator scores and decision outcomes led to the final action

VP8 makes attribution load-bearing:

> The harness must explain why guidance was injected, minimized, rejected, or
> persisted.

Without comprehensive attribution, lab experiments cannot distinguish genuine
improvement from suppressed behavior that harms quality.

## Decision

Adopt **attribution-first decision records** as a required output of the
evaluator loop (ADR-018, ADR-019):

1. **Every decision produces an attribution record**: when the Decision Engine
   selects an outcome, it emits a structured attribution record that explains
   the decision.

2. **Attribution record contents**:
   - decision outcome (continue / minimal / abstain / backtrack / write)
   - evaluator scores per dimension (relevance, consistency, usefulness)
   - composite confidence score
   - per-candidate fate: which candidates survived, which were trimmed, which
     were rejected, and why
   - memory decision: whether a write was approved, deferred, or skipped, and
     why
   - timestamp, task context, and session identity

3. **Inline and persisted attribution**: attribution records are returned
   inline in the decision envelope (so consuming agents can inspect them
   immediately) and persisted to the local workspace so lab experiments can
   analyze them later.

4. **Integration with the experiment lab**: lab run records (ADR-013) should
   reference the attribution records produced during the run. Lab reports
   (ADR-023) can aggregate attribution data to explain token reductions,
   abstention rates, and backtrack frequencies.

5. **Extension of existing provenance**: attribution records extend, not
   replace, the existing provenance model (ADR-004) and kickoff inclusion
   reasons (ADR-016). Graph provenance tracks who wrote what and when.
   Attribution records track why the evaluator loop made a particular decision.

## Consequences

### Positive
- Lab experiments can explain whether token reductions come from real
  improvement or over-suppression
- Agents can inspect attribution and calibrate trust in CGE decisions
- Decision transparency supports debugging and threshold tuning
- Attribution records provide the evidence trail for session-to-session
  continuity analysis

### Negative
- Attribution records add storage and output volume
- Every consumer must understand the attribution envelope or learn to ignore it
- Attribution generation adds work to every evaluation pass

### Risks
- Risk: attribution records become verbose and agents stop reading them
  - Mitigation: keep the inline summary compact; persist full detail separately
- Risk: attribution storage grows unbounded
  - Mitigation: attribution records follow the same workspace lifecycle as lab
    artifacts; old records can be pruned

## Alternatives Considered

### Extend provenance metadata only, without a separate attribution model
- Pros: simpler schema, reuses existing provenance fields
- Cons: provenance tracks writes; attribution tracks evaluation decisions —
  different concerns
- Rejected because: conflating write provenance with decision attribution
  obscures both

### Make attribution optional and off-by-default
- Pros: no overhead for simple uses
- Cons: lab experiments would lose their primary evidence trail
- Rejected because: VP8 requires attribution as a load-bearing product feature,
  not an optional diagnostic
