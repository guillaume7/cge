# ADR-017: Verification-calibrated audit kickoff policies

## Status
Accepted

## Context

TH6 introduced family-aware kickoff, advisory modes, inclusion reasons, and
reporting-family abstention. The targeted TH6 confirmation batch produced a
mixed but important result:

- reporting/synthesis improved and flipped positive under abstention
- contradiction diagnosis improved sharply
- write-producing work remained graph-positive
- verification/audit regressed badly, especially for stats audits and
  workflow-verification tasks

This shows that verification/audit is still too broad to manage as a single
policy family. The product now needs a stable decision for how verification
requests should be routed, downgraded, or suppressed without weakening the gains
already established for other families.

## Decision

Adopt a **verification-calibrated audit kickoff policy** on top of the TH6
advisory model:

1. Verification/audit tasks are routed into narrower verification sub-profiles
   before kickoff context is projected.
2. At minimum, VP7 must distinguish among:
   - stats-oriented audits
   - workflow-verification tasks
   - general evidence/provenance audits
3. Each verification sub-profile owns its own suppressions, token budget caps,
   and downgrade rules.
4. Verification tasks use stricter confidence discipline than write-producing
   work and may downgrade to minimal or abstain even when the broader
   verification family would previously have injected.
5. Experiment reruns preserve raw `workflow.start` responses and baseline
   prompt-surface metadata so verification regressions can be explained from
   artifacts alone.

This decision extends the existing workflow-start and experiment-lab
architecture rather than replacing them.

## Consequences

### Positive
- Verification regressions can be targeted directly without broadening or undoing TH6 behavior in other families.
- Stats and workflow-verification tasks gain narrower, more explainable kickoff policies.
- Future reruns become more diagnosable because kickoff attribution is preserved as a first-class artifact.

### Negative
- Workflow-start policy selection becomes more granular and therefore more complex to test.
- Some verification tasks will receive less graph context than before, which may feel conservative when evidence happens to be useful.
- The targeted rerun adds another validation gate before a full rerun can be justified.

### Risks
- Risk: verification sub-profiles become overly narrow and miss useful evidence.
  - Mitigation: preserve minimal-kickoff and explicit pull-on-demand behavior.
- Risk: only the visible stats-audit failures are fixed while workflow-verification remains noisy.
  - Mitigation: keep both stats and workflow-verification as separate VP7 success checks.
- Risk: experiment artifacts remain too thin to diagnose the next failure wave.
  - Mitigation: make raw kickoff and prompt-surface capture part of the phase, not optional follow-up work.

## Alternatives Considered

### Keep verification/audit as one family and only lower the global threshold
- Pros: minimal implementation churn
- Cons: does not distinguish the clearly different failure modes seen in stats audits versus workflow verification
- Rejected because: the TH6 confirmation batch shows that the regression is profile-specific, not just threshold-specific

### Skip calibration and run the full rerun anyway
- Pros: faster path to another large result set
- Cons: spends campaign budget while a known family-level regression remains unresolved
- Rejected because: a full rerun would mostly confirm an already visible verification-family problem

### Redesign the whole retrieval engine for audit tasks
- Pros: could unlock deeper long-term retrieval improvements
- Cons: too broad for the immediate next phase and risks delaying evidence-backed calibration
- Rejected because: VP7 should be a focused verification-calibration phase, not a full retrieval rewrite
