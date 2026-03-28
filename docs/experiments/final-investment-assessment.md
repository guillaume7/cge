# Final Investment Assessment

This report is the final product-facing conclusion from the VP5 through TH7
evidence loop.

Short version:

- CGE produced real wins in some task families.
- CGE also remained too unstable, too calibration-sensitive, and too difficult
  to interpret confidently across the full operating surface.
- As implemented in this repository, CGE is **not a strong enough foundation for
  further broad product investment**.

That conclusion is not based on sentiment alone. It is based on the combined
record from the controlled lab campaigns, the TH6/TH7 remediation work, and the
follow-up verification batch.

## Executive conclusion

The final judgment is:

- **Do not continue broad productization investment in the current CGE
  implementation.**
- **Do not spend more budget on a broader rerun from this branch state.**
- If any work continues, it should be treated as a narrow research/debugging
  track rather than as a platform to build upon.

The reason is simple: the system never crossed the line from "interesting and
sometimes helpful" to "dependable substrate that agents can safely build on."

## What the evidence did show

Across the VP5 decision-grade campaign and the TH6 confirmation batch, CGE
showed genuine value in some areas:

- `write_producing` work was repeatedly the strongest fit
- some troubleshooting tasks improved materially once retrieval became more
  selective
- TH6 reduced several obvious contamination failures
- TH7 improved observability by exposing verification subprofiles, stricter
  downgrade behavior, and machine-readable kickoff attribution

These are not fake wins. They show there is a real product idea underneath the
implementation.

## Why that still was not enough

The decision turns negative because the remaining weaknesses are exactly the
ones that block CGE from serving as a reliable base layer.

### 1. The system remained family-unstable

The VP5 decision-grade campaign already showed that performance depended heavily
on task family:

- `write_producing` was clearly positive for graph-backed kickoff
- `troubleshooting_diagnosis` looked promising but noisy
- `verification_audit` was mixed
- `reporting_synthesis` was the main regression bucket

TH6 improved reporting and contradiction-heavy troubleshooting, but the TH6
confirmation batch exposed severe verification regressions instead of resolving
them. TH7 specifically targeted that problem, yet the final verification batch
still left `audit-graph-stats-snapshot` strongly graph-negative on token cost.

That means the implementation still does not have stable cross-family behavior.

### 2. Precision improved, but not enough to make the product trustworthy

The consulted high-end models were broadly bullish on CGE only under a strict
condition:

- retrieved context must be selective
- provenance must be inspectable
- weak briefs must be suppressible
- the system must know when to stay quiet

TH6 and TH7 implemented exactly that direction. But the runtime still produced
results that were too brittle:

- one family could improve while another remained badly regressed
- abstention could still carry large graph-side token cost
- at least one retained positive-control task appeared to route through the
  wrong family in the TH7 verification batch

So the empirical result is that the calibration strategy became smarter without
becoming dependable enough.

### 3. The measurement loop itself still had decision-shaping gaps

By the end of TH7, the lab had become much stronger, but the final confirmation
batch still surfaced interpretation problems:

- the report-level verification gate came back as `not_applicable` when the
  selected runs clearly included verification-family kickoff artifacts
- the selected TH7 runs were all unscored for evaluation, leaving the final
  batch usable mainly for telemetry and token analysis rather than full
  quality/resumability assessment
- the write-producing positive control looked suspicious because its with-graph
  kickoff appeared to classify as `verification_audit / stats_audit`

These gaps matter because a system that is hard to interpret is also hard to
trust as a foundation.

## How to read the consulting-model consensus in hindsight

The cross-model consulting was useful, but it should now be read as a
conditional endorsement, not a direct investment recommendation.

The high-end models were effectively saying:

- "this could be worth investing in"
- "precision control is the core product problem"
- "the value disappears if the system becomes noisy or overconfident"

The implementation work took those recommendations seriously. The experiments
that followed are what ultimately changed the answer.

So the survey was not wrong. It identified the right bet. The problem is that
the current implementation did not clear the bar required to realize that bet in
a dependable way.

## Final opinion on investability

If the question is:

> Is this CGE implementation a useful platform to keep building broader product
> capability on?

The answer is:

**No. Not in its current form.**

If the question is:

> Is there still a small research thread here?

The answer is:

- maybe, but only if it is explicitly scoped as research into precision
  calibration and measurement hygiene
- not as continued product expansion
- not as a trusted shared-memory substrate ready for agents to rely on

## Practical recommendation

Recommended action for this repository state:

1. Treat the current CGE implementation as a concluded experimental branch of
   product exploration.
2. Preserve the artifact trail and the conclusions, because the work was still
   informative.
3. Do not spend more effort on broadening features, polishing UX, or scaling
   reruns from this state.
4. If future work happens at all, restart from a narrower thesis with stricter
   success criteria and stronger evaluation closure.

## Evidence references

Primary evidence trail:

- `docs/experiments/README.md`
- `.graph/lab/reports/report-20260327t193838z.json`
- `.graph/lab/dogfooding/generated/th6-confirmation-batch-20260328t000323z/confirmation-debrief.json`
- `.graph/lab/dogfooding/generated/th7-verification-confirmation-batch-20260328t135739z/confirmation-debrief.json`
- `.graph/lab/dogfooding/generated/th7-verification-confirmation-batch-20260328t135739z/report-response.json`

Related design history:

- `docs/vision_of_product/VP6-precision-governed-advisory-kickoff/README.md`
- `docs/vision_of_product/VP7-verification-calibrated-audit-kickoff/README.md`
- `docs/ADRs/ADR-016-precision-governed-advisory-kickoff.md`
- `docs/ADRs/ADR-017-verification-calibrated-audit-kickoff.md`
