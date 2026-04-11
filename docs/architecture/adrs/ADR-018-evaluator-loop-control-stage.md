# ADR-018: Evaluator loop as first-class control-loop stage

## Status
Proposed

## Context

Through VP1–VP7 the product developed comprehensive graph persistence,
retrieval, projection, workflow, and experiment infrastructure. However, every
context retrieval path — hybrid query, token-budgeted projection, and
delegated-workflow kickoff — treats returned graph data as provisionally
trustworthy once it passes ranking. No component explicitly asks whether
retrieved context is actually helping the current task before it reaches the
consuming agent.

VP8 identifies this missing evaluator loop as the product's core functional gap:

> Without that loop, the product cannot reliably tell whether retrieved context
> is relevant, stored graph memory is still trustworthy, or generated guidance
> should be used, revised, minimized, or rejected.

VP6 and VP7 introduced family-aware kickoff policies (ADR-016, ADR-017) that
can suppress or narrow context injection. Those policies are rule-based and
static. VP8 requires a richer evaluation stage that scores context dynamically
based on the current task, the candidate set, and consistency signals.

The seed architecture sketch in the VP8 vision describes the desired shape:

```
retrieve → evaluate → decide → act → update memory
```

This ADR establishes the evaluator as a first-class stage in the CGE pipeline.

## Decision

Add a **Context Evaluator** component that sits between retrieval and
projection/injection:

1. **Evaluation point**: every context retrieval path used for injection
   (`graph context`, `workflow start`) must pass candidate results through the
   Context Evaluator before they reach the consuming agent or the context
   projector.

2. **Scoring dimensions**: the evaluator scores each candidate bundle on at
   minimum three dimensions:
   - **relevance** — is this candidate about the current task?
   - **consistency** — does this candidate agree with other known signals, or
     does it contradict them?
   - **likely usefulness** — is including this candidate likely to help task
     completion, or is it noise?

3. **Composite confidence**: the evaluator produces a composite confidence
   score that the downstream Decision Engine (ADR-019) uses to select an
   outcome.

4. **Evaluation of outputs**: the evaluator can also score candidate task
   outputs (not just retrieved context) when the consuming agent requests
   in-loop evaluation. This supports the iterative critique/revise cycle
   described in the VP8 vision.

5. **Implementation**: the evaluator is an in-process Go component. It uses
   local heuristics — overlap scoring, provenance recency, structural
   neighborhood coherence — and optionally delegates scoring to the same
   retrieval primitives already available. It does not require a hosted
   inference service.

6. **Integration with existing retrieval**: the evaluator does not replace the
   hybrid retrieval pipeline (ADR-006) or the family-aware kickoff policies
   (ADR-016, ADR-017). It sits downstream of retrieval and upstream of
   projection. Family policies continue to suppress or narrow candidates;
   the evaluator scores what survives.

## Consequences

### Positive
- Context is no longer assumed trustworthy after ranking alone
- Graph drift becomes detectable at evaluation time rather than only through
  post-hoc lab analysis
- The product can explain why context was accepted, narrowed, or rejected before
  injection
- The evaluator foundation supports iterative generate/critique/revise loops

### Negative
- Adds latency to every retrieval path that uses evaluation
- Scoring heuristics require careful calibration to avoid over-filtering useful
  context
- Introduces a new component with its own testing and tuning surface

### Risks
- Risk: evaluator scores become arbitrary and lose agent trust
  - Mitigation: start with explainable heuristic dimensions; add richer scoring
    only when lab results justify it
- Risk: evaluation overhead costs more tokens than it saves
  - Mitigation: keep evaluation local and heuristic-based in VP8; do not
    delegate to LLM calls for scoring unless a future VP proves the need
- Risk: evaluator incorrectly rejects useful context
  - Mitigation: decision outcomes include minimal and continue paths that still
    inject narrower context (ADR-019)

## Alternatives Considered

### Keep retrieval ranking as the only quality gate
- Pros: simpler pipeline, no new component
- Cons: ranking cannot detect task-level relevance drift or consistency problems
- Rejected because: VP8 identifies this as the core product failure to correct

### Delegate evaluation to an LLM scoring call
- Pros: richer semantic evaluation
- Cons: adds latency, cost, and an external dependency; breaks local-first
  principle for the core path
- Rejected because: VP8 must stay local-first; LLM-based scoring can be
  explored in a later VP if heuristic scoring proves insufficient

### Make evaluation optional and off-by-default
- Pros: zero disruption to existing paths
- Cons: the evaluator loop would never become the normal product behavior
- Rejected because: VP8's core intent is to make evaluation first-class, not
  optional
