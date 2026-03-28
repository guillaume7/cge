# VP7 — Verification-Calibrated Audit Kickoff

> Status: proposed. This VP exists to repair the verification/audit regressions exposed by the TH6 confirmation batch.

## Vision Summary

Build the seventh product phase of the Cognitive Graph Engine around one
practical goal: make graph-backed kickoff for verification and audit tasks as
disciplined as TH6 made reporting and synthesis.

TH6 proved that family-aware abstention and calibration can work. Reporting
flipped positive, contradiction diagnosis improved sharply, and write-producing
work remained a strong fit. But the same Stage 1 batch also showed that
verification/audit is still under-calibrated: stats audits and workflow
verification tasks are receiving expensive, low-value kickoff context.

VP7 exists to correct that mismatch without undoing TH6's gains elsewhere.

## Product Intent

We are no longer trying to answer only:

- can the graph abstain when reporting is contamination-prone?
- can diagnostic tasks benefit from tighter family-aware policies?

VP7 must answer the next operational question:

- can CGE distinguish among verification tasks well enough to avoid injecting
  context that is irrelevant to the exact audit being performed?

The intent is to turn verification/audit into another **precision-governed**
family instead of leaving it as a broad bucket:

- stats audits should not inherit workflow-heavy kickoff clutter
- workflow verification should not inherit irrelevant graph health or finish
  artifacts
- general evidence audits should use narrower, task-aligned kickoff policies
- the experiment harness should preserve enough attribution to explain why a
  verification kickoff helped, regressed, minimized, or abstained

## Core Hypothesis

The narrow VP7 hypothesis is:

> **If verification and audit tasks are split into narrower calibration profiles
> with stricter confidence and token discipline, then CGE will remove the large
> verification-family regressions seen after TH6 while preserving the reporting,
> troubleshooting, and write-producing gains already achieved.**

## Primary Outcome

Turn `graph workflow start` into a **verification-calibrated kickoff surface**
that can:

1. distinguish among stats audits, workflow verification, and general evidence
   audits
2. apply verification-specific suppressions, budgets, and downgrade paths
3. minimize or abstain when verification evidence is weak, mixed, or
   contamination-prone
4. expose enough attribution to explain verification-family kickoff outcomes in
   later experiment analysis
5. support a targeted rerun that decides whether a broader full rerun is now
   justified

## Primary Users

VP7 is for:

- maintainers who want verification tasks to stop being the family that erases
  broader graph gains
- delegated agents performing evidence checks, audits, or repo-policy
  verification
- experiment operators who need a precise explanation for why a verification
  kickoff injected, minimized, or abstained

## Core Jobs To Be Done

1. Let workflow start recognize verification sub-profiles instead of treating
   all audits as one coarse family.
2. Let stats-oriented audits suppress workflow-specific artifacts by default.
3. Let workflow-verification tasks suppress unrelated graph-health and
   contradiction-heavy context.
4. Let verification tasks use stricter confidence thresholds and smaller token
   budgets than implementation-oriented work.
5. Let verification kickoff downgrade to minimal or abstain when evidence
   alignment is weak.
6. Let the experiment harness preserve raw kickoff responses and baseline
   prompt-surface metadata for targeted reruns.
7. Let maintainers decide on a full rerun only after verification-family
   regressions are roughly neutralized.

## Product Principles

- **Calibration by sub-profile, not by family label alone**: verification is too
  broad to govern as one policy bucket.
- **Cheap audits beat clever but noisy audits**: verification work values
  precision and bounded evidence more than broad recall.
- **Abstention and minimization are valid audit outcomes**: an audit kickoff that
  declines to guess is often safer than one that over-injects.
- **Attribution is load-bearing**: every future verification rerun must preserve
  enough artifacts to explain why the kickoff policy behaved as it did.
- **Do not sacrifice existing gains**: reporting, diagnosis, and write-producing
  improvements from TH6 should remain intact.

## VP7 Scope

### Included

- verification/audit sub-profile classification and policy routing
- stats-specific and workflow-verification-specific suppressions
- stricter verification-family confidence thresholds and token caps
- verification-family downgrade-to-minimal or abstain behavior
- richer campaign attribution capture for workflow-start and baseline prompt
  surfaces
- a targeted rerun to decide whether the broader full rerun should be spent

### Excluded

- a full campaign rerun before verification-family regressions are reduced
- broad redesign of non-verification families
- hosted analysis services or remote policy engines
- replacing the existing experiment lab with a new harness

## Command Intent

### `graph workflow start`

VP7 should extend workflow start so verification/audit requests can resolve to
honest, narrower outcomes:

1. **inject** when the verification sub-profile is aligned and evidence is strong
2. **minimal** when the task is verification-safe but the evidence should be
   tightly bounded
3. **abstain** when verification evidence is weak, mixed, or likely to pull the
   audit toward the wrong context

### Experiment surfaces

VP7 should also make the next targeted rerun more legible by preserving:

- the raw `workflow.start` response for graph-backed runs
- the prompt-surface metadata for baseline runs
- enough per-pair attribution to explain regressions without guesswork

## Success Criteria

- verification tasks no longer behave as one undifferentiated kickoff family
- stats and workflow-verification regressions shrink materially versus the TH6
  confirmation batch
- verification-family token behavior moves closer to neutral instead of showing
  catastrophic positive deltas against graph
- targeted rerun artifacts are rich enough to explain why a verification pair
  won, regressed, minimized, or abstained
- a maintainer can decide whether to spend the full rerun budget from evidence
  instead of intuition
