# ADR-023: Harness-aware lab conditions and token-decline measurement

## Status
Proposed

## Context

VP4 through VP7 built a local experiment lab with controlled conditions,
immutable run records, separated evaluation, measured token telemetry, and
scientific-style reporting (ADR-012, ADR-013, ADR-014, ADR-015). The existing
condition model compares **with-graph** versus **without-graph** conditions,
measuring whether graph-backed context improves task outcomes and token
efficiency.

VP8 introduces a broader product surface: the evaluator loop, decision engine,
and attribution layer. The primary VP8 success signal is:

> Lab experiments should show a clear decline in token consumption when using
> the CGE harness.

"The CGE harness" now means more than graph-backed context injection. It means
the full evaluate → decide → act → update pipeline. The lab must be able to
measure the contribution of this pipeline, not just the contribution of graph
retrieval alone.

Additionally, VP8 requires the lab to verify that token reductions are not
caused by over-abstention — the evaluator simply refusing to inject anything,
which trivially reduces tokens but harms task quality.

## Decision

Extend the experiment lab to support **harness-aware conditions** and
**token-decline verification**:

1. **Extended condition model**: in addition to the existing `with-graph` and
   `without-graph` conditions, VP8 adds:
   - **with-harness** — the full CGE pipeline including retrieval, evaluation,
     decision, and attribution
   - **without-harness** — no CGE involvement at all; the agent operates with
     standard Copilot CLI only
   - **graph-only** — graph retrieval and projection without the evaluator loop
     (the pre-VP8 behavior), useful as a regression baseline

   Existing condition definitions remain valid. New conditions are additive.

2. **Token-decline as primary metric**: lab reports must surface token
   consumption comparisons across harness-aware conditions as a first-class
   metric. Reports should clearly show:
   - total token delta between with-harness and without-harness
   - per-task token distributions
   - confidence intervals on the delta

3. **Over-abstention detection**: lab reports must include:
   - abstention rate across with-harness runs
   - correlation between abstention and task-quality scores
   - explicit flagging when token savings coincide with quality regression

4. **Attribution aggregation**: lab reports should aggregate attribution records
   (ADR-021) across runs to show:
   - distribution of decision outcomes (continue/minimal/abstain/backtrack/write)
   - reasons for abstention and backtracking
   - per-task-family decision patterns

5. **Backward compatibility**: existing run records, evaluation records, and
   reports from VP4–VP7 experiments remain valid. The extended condition model
   does not alter the run record schema; condition IDs are already free-form
   strings. The new conditions use new condition IDs.

## Consequences

### Positive
- The lab can measure the VP8 evaluator loop's contribution specifically, not
  just graph retrieval's contribution
- Over-abstention becomes detectable before it is mistaken for improvement
- Attribution aggregation makes decision patterns visible across batches
- Existing experiment data is preserved

### Negative
- More conditions increase the number of runs needed for meaningful comparisons
- Attribution aggregation adds reporting complexity
- The graph-only baseline condition adds a third comparison arm

### Risks
- Risk: too many conditions dilute experiment power
  - Mitigation: VP8 experiments should focus on the with-harness vs
    without-harness comparison as the primary arm; graph-only is an optional
    diagnostic
- Risk: attribution aggregation is expensive to compute for large batches
  - Mitigation: keep aggregation simple (counts and distributions) and avoid
    expensive cross-record joins
- Risk: token-decline measurement depends on reliable telemetry (ADR-015), and
  incomplete telemetry weakens the primary metric
  - Mitigation: maintain ADR-015's strict measurement_status contract;
    experiments with unavailable telemetry are excluded from token comparisons

## Alternatives Considered

### Keep existing with-graph / without-graph conditions only
- Pros: no lab changes needed
- Cons: cannot measure the evaluator loop's contribution separately from raw
  graph injection
- Rejected because: VP8's value proposition is the evaluator loop, not just
  the graph; the lab must measure what VP8 adds

### Build a separate VP8 experiment framework
- Pros: clean slate, no backward-compatibility constraints
- Cons: duplicates existing lab infrastructure, wastes VP4–VP7 investment
- Rejected because: extending the existing lab is cheaper and preserves
  comparability with prior experiments

### Measure token decline without tracking abstention
- Pros: simpler reporting
- Cons: cannot distinguish real improvement from over-suppression
- Rejected because: the VP8 vision explicitly warns about this risk and
  requires the lab to detect it
