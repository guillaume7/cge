# ADR-011: Benchmark delegated workflow through a local harness and CLI report surface

## Status
Proposed

## Context

VP3 must dogfood a narrow hypothesis rather than rely on intuition:

- graph-backed kickoff and handoff for non-trivial delegated subtasks should save
  recovery tokens and improve delegation quality

To validate that claim, the product needs evidence in two places:

- a repo-local evaluation harness that can run comparable scenarios repeatedly
- a CLI-facing report surface that exposes the evidence to agents and humans

The first benchmark family is explicitly limited to **non-trivial delegated
subtasks**.

## Decision

Support benchmarking through **two local surfaces**:

1. **Repo-local evaluation harness**
   - stores scenario definitions and run outputs locally
   - compares with-graph and without-graph modes on the same delegated subtasks

2. **CLI-facing report surface**
   - emits machine-readable benchmark summaries from local benchmark data
   - keeps benchmark evidence visible inside normal CGE workflows

The benchmark must compare both:

- token/prompt usage
- output quality / delegation quality

and not treat token reduction alone as success.

## Consequences

### Positive
- Creates a falsifiable validation loop for VP3
- Keeps benchmark evidence local, reproducible, and inspectable
- Gives both agents and humans a direct way to assess whether the workflow is
  worth its ceremony
- Supports later product expansion only if evidence justifies it

### Negative
- Benchmark design requires careful scenario selection and scoring rubrics
- Quality measurement is more subjective than token counting
- Adds a support surface beyond the three primary workflow commands

### Risks
- Risk: benchmark runs may be gamed toward lower tokens but worse outcomes
  - Mitigation: require fixed acceptance criteria and compare output quality and
    resumability alongside token usage

## Alternatives Considered

### Ad hoc manual judgment only
- Pros: simplest to start, no extra benchmark tooling
- Cons: weak evidence, hard to compare runs honestly
- Rejected because: VP3 explicitly needs scientific-style validation

### External telemetry / observability platform
- Pros: richer analytics, dashboards, historical comparisons
- Cons: hosted complexity, operational overhead, privacy concerns
- Rejected because: VP3 needs local evidence first, not a telemetry system

### Token-only benchmark
- Pros: simpler metric collection
- Cons: incentivizes shallow wins and ignores delegation quality
- Rejected because: VP3 value depends on both token savings and useful outcomes
