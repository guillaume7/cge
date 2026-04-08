# ADR-019: Normalized decision outcomes for the evaluator loop

## Status
Proposed

## Context

ADR-018 introduces a Context Evaluator that scores candidate context and
candidate outputs. The evaluator produces confidence scores, but the product
also needs a component that translates those scores into actionable outcomes.

The current product has only two implicit outcomes for retrieved context:

1. inject (retrieval returned something, so use it)
2. abstain (the family-aware policy suppressed injection — ADR-016)

VP8 identifies a richer set of honest outcomes that the product needs:

> - **continue** when evidence is good enough
> - **backtrack** when the current path is degrading quality
> - **minimal** when only narrow, low-risk guidance should be injected
> - **abstain** when the harness cannot justify strong guidance
> - **write** only when the new state is strong enough to improve future
>   continuity

These outcomes matter because the dominant failure mode is not missing answers
but confidently carrying forward weak or stale state.

## Decision

Add a **Decision Engine** component downstream of the Context Evaluator
(ADR-018):

1. **Five normalized outcomes**: the Decision Engine selects exactly one outcome
   per evaluation pass:
   - **continue** — evaluator confidence is above the injection threshold;
     deliver the full scored context bundle to the consumer
   - **minimal** — confidence is moderate; deliver only the highest-scored
     subset with explicit narrowing attribution
   - **abstain** — confidence is below the injection threshold; deliver no
     context and record why
   - **backtrack** — the evaluator detects that the current retrieval or
     generation path is degrading quality compared to prior state; signal the
     consuming agent to revise its approach
   - **write** — the evaluator confirms that a candidate output is strong
     enough to persist as new graph memory; approve the memory update

2. **Threshold-driven selection**: outcomes are selected by comparing the
   evaluator's composite confidence against configurable thresholds. VP8
   should ship with conservative defaults that prefer minimal and abstain over
   aggressive injection.

3. **Machine-readable decision envelope**: every decision is returned as a
   structured envelope containing:
   - the selected outcome
   - the evaluator scores that motivated the outcome
   - the attribution records (ADR-021)
   - the surviving context bundle (when outcome is continue or minimal)

4. **Composability with existing policies**: the Decision Engine operates after
   family-aware kickoff policies (ADR-016, ADR-017) and after evaluation
   scoring (ADR-018). If a family policy already suppressed a task, the
   Decision Engine respects that suppression. The Decision Engine can further
   narrow or abstain but cannot override a family-level suppression.

5. **Backtrack signaling**: the backtrack outcome is advisory. CGE does not
   autonomously re-execute agent work. It signals the consuming agent that the
   current path appears to be degrading and recommends revision. The agent
   decides whether to act on the signal.

## Consequences

### Positive
- The product can express honest uncertainty instead of defaulting to injection
- Backtracking and abstaining become explicit success paths, not hidden failures
- Agents receive structured decision metadata alongside context
- Conservative defaults reduce irrelevant context injection

### Negative
- Five outcomes are more complex for consumers than inject/suppress
- Threshold tuning requires empirical evidence from lab experiments
- Backtrack signaling depends on consuming agents respecting the advisory

### Risks
- Risk: conservative defaults cause over-abstention and reduce useful context
  delivery
  - Mitigation: lab experiments should track abstention rate alongside token
    savings; if abstention exceeds a reasonable bound without quality gains,
    thresholds should be relaxed
- Risk: agents ignore backtrack signals
  - Mitigation: backtracking is advisory by design; CGE does not force agent
    behavior. Attribution records make ignored backtracks visible in lab
    analysis.
- Risk: the five outcomes become a premature taxonomy that constrains future
  decision richness
  - Mitigation: keep the outcome set small and add new outcomes only when lab
    evidence justifies them

## Alternatives Considered

### Keep inject/abstain as the only outcomes
- Pros: minimal change to existing consumers
- Cons: cannot express partial confidence, backtracking, or write-readiness
- Rejected because: VP8's decision model explicitly requires richer outcomes

### Let the consuming agent make all decisions without CGE guidance
- Pros: CGE stays a pure retrieval tool; no decision complexity
- Cons: the product cannot improve task quality without influencing decisions
- Rejected because: VP8's core value proposition is that CGE should help decide,
  not just retrieve

### Use a continuous confidence score instead of discrete outcomes
- Pros: avoids threshold discontinuities
- Cons: consumers must implement their own outcome mapping, reducing consistency
- Rejected because: normalized outcomes are more composable and easier to
  attribute in lab analysis
