# VP6 — Precision-Governed Advisory Kickoff

> Status: accepted. Architecture and planning artefacts derived from this VP.

## Vision Summary

Build the sixth product phase of the Cognitive Graph Engine around one practical
goal: make graph-backed kickoff precise enough to help the right tasks while
stepping back when the graph is likely to harm the agent.

VP5 gave CGE scientifically credible evidence and a cross-model survey of what
high-end coding agents actually want from graph context. The conclusion was not
"more graph everywhere." It was: precision is the product, false positives are
more dangerous than missing context, and advisory freedom is a feature rather
than a fallback.

The result should be a kickoff system that is family-aware, confidence-gated,
provenance-rich, and explicitly suppressible.

## Product Intent

We are no longer trying to answer only:

- can graph-backed kickoff reduce token cost?
- can it help implementation and diagnosis work?

VP6 must answer a stricter operational question:

- can CGE reliably help the right task families while abstaining when kickoff
  context is low-confidence or contamination-prone?

The intent is to shift CGE from a generic context injector toward a
**precision-governed advisory system**:

- implementation and diagnosis should get sharper graph help
- reporting and synthesis should default to no kickoff unless a future policy
  explicitly allows otherwise
- agents should always be able to see why context was included and to opt out
  cleanly

## Core Hypothesis

The narrow VP6 hypothesis is:

> **If workflow kickoff becomes task-family aware, confidence-gated, and
> provenance-rich, then CGE will preserve or improve its write-producing and
> diagnostic gains while reducing retrieval-contamination regressions in
> vulnerable task families.**

## Primary Outcome

Turn `graph workflow start` into a **precision-first kickoff surface** that can:

1. classify the incoming task into a kickoff family
2. select a retrieval policy for that family
3. abstain from kickoff when the family or confidence score says abstention is safer
4. explain why each included entity survived the policy
5. let the agent bypass or minimize kickoff without breaking the workflow path

## Primary Users

VP6 is for:

- maintainers who want graph-backed workflow to stay net-positive instead of
  becoming ceremony
- delegated coding agents that benefit from compact, relevant startup context
- reviewers and experimenters who need a principled explanation for when CGE
  injects context and when it declines to do so

## Core Jobs To Be Done

1. Let workflow start classify a delegated task into a retrieval family.
2. Let each family apply explicit entity allowlists, suppressions, and token
   expectations.
3. Let reporting and synthesis tasks default to no kickoff.
4. Let the kickoff surface expose a confidence signal and abstain when
   confidence is low.
5. Let each included entity carry a short inclusion reason.
6. Let agents choose no kickoff or minimal kickoff explicitly without breaking
   the delegated workflow contract.
7. Let the system degrade gracefully on sparse repos and ambiguous task text.

## Product Principles

- **Precision is the product**: retrieval quality matters more than retrieval breadth.
- **False positives are worse than misses**: low-confidence injection is more
  harmful than making the agent rediscover context.
- **Advisory by design**: kickoff should recommend, not coerce.
- **Explain inclusion, not just ranking**: every kickoff entity should justify
  its presence in one line.
- **Family-aware behavior over one-size-fits-all heuristics**: different task
  families need different context policies.
- **Abstention is a valid success path**: reporting and synthesis tasks should
  default to no kickoff until evidence justifies broader injection.

## VP6 Scope

### Included

- task-family classification for delegated workflow start
- family-specific retrieval allowlists and suppressions
- no-kickoff default for reporting and synthesis tasks
- family-aware kickoff confidence thresholds
- one-line inclusion reasons in kickoff briefs
- explicit no-kickoff and minimal-kickoff controls
- graceful degradation for sparse repos and ambiguous task requests

### Excluded

- broad LLM-as-judge evaluation hardening in this phase
- full multi-family benchmark redesign
- new remote retrieval services or hosted policy engines
- forcing graph kickoff into task families where abstention is currently safer

## Command Intent

### `graph workflow start`

VP6 should make workflow start return one of three honest states:

1. **inject** — graph context is relevant and confidence is high enough
2. **minimal** — graph context is allowed but deliberately constrained
3. **abstain** — the family or confidence score says kickoff should step back

In all cases, the command should keep the workflow usable and machine-readable.

## Success Criteria

- workflow start can classify tasks into explicit kickoff families
- retrieval policies are family-specific rather than global
- reporting and synthesis tasks receive no kickoff by default
- kickoff output includes per-entity inclusion reasons
- low-confidence kickoff results can abstain and recommend pull-on-demand
- agents can explicitly request no kickoff or minimal kickoff without breaking
  the workflow contract

