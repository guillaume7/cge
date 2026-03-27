# ADR-016: Precision-governed advisory kickoff policies

## Status
Accepted

## Context

VP5 produced two relevant results for delegated workflow kickoff.

First, the 168-run campaign showed that graph-backed kickoff is already
defensible for write-producing and some diagnostic tasks, but it regresses on
some reporting and synthesis tasks when irrelevant entity types contaminate the
brief.

Second, the cross-model survey reached strong consensus:

- retrieval precision is the product
- false positives are more harmful than missing context
- kickoff should remain advisory and suppressible
- reporting and synthesis should default to abstention
- provenance-rich inclusion reasons are required for agent trust calibration

The current workflow-start path is calibrated, but it still behaves too much
like a generic context injector. VP6 needs a stable architectural decision for
how kickoff policy should be selected and when abstention is the correct result.

## Decision

Adopt a **precision-governed advisory kickoff policy** for delegated workflow
start:

1. Every workflow-start task is classified into a kickoff family before graph
   context is projected.
2. Each family owns its own entity-type allowlists, suppressions, token budget
   defaults, and abstention rules.
3. Reporting and synthesis families default to **no kickoff** in VP6.
4. Kickoff injection is confidence-gated; low-confidence results may abstain and
   recommend on-demand graph pull instead of pushing context.
5. Every included kickoff entity must carry a short inclusion reason.
6. Agents may explicitly request no kickoff or minimal kickoff without breaking
   the workflow contract.

This decision extends the existing workflow-start architecture; it does not
replace the retrieval engine or add remote policy infrastructure.

## Consequences

### Positive
- CGE can preserve strong task-family wins while reducing known contamination regressions.
- Abstention becomes an explicit, machine-readable success path instead of an implicit failure mode.
- Agents can calibrate trust in kickoff context because each entity explains why it was included.

### Negative
- Workflow start becomes more policy-driven and therefore more complex to explain and test.
- Some tasks will intentionally receive less graph context than before, which may feel conservative at first.
- Family classification errors can misroute tasks into the wrong policy unless the rules are carefully tested.

### Risks
- Risk: overly strict policies hide useful context for borderline tasks.
  - Mitigation: keep minimal-kickoff and pull-on-demand paths available.
- Risk: family classification rules drift into opaque heuristics.
  - Mitigation: keep family selection explicit and explainable in the kickoff output.
- Risk: future work broadens kickoff again without preserving abstention.
  - Mitigation: treat abstention as a first-class policy outcome in docs and interfaces.

## Alternatives Considered

### Keep one global retrieval policy and only tune ranking scores
- Pros: minimal architectural churn, simpler surface area
- Cons: does not address the family-specific regressions shown in VP5
- Rejected because: the campaign and survey both point to task-family policy as the main corrective lever

### Always inject graph context but add stronger warnings
- Pros: agents always see potentially useful graph data
- Cons: warnings do not remove contamination; false positives still shape reasoning
- Rejected because: advisory warnings are weaker than abstention when confidence is low

### Make evaluation hardening the primary next step
- Pros: improves measurement fidelity for future experiments
- Cons: does not directly reduce current workflow-start regressions
- Rejected because: VP6 is runtime-first and should improve product behavior before expanding evaluation infrastructure

