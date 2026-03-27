# ADR-015: Measured execution telemetry with explicit completeness states

## Status

Accepted

## Context

VP4 introduced a local experiment lab with an artifact-first run ledger and
report surface. Dogfooding exposed a methodological flaw: the lab schema and
reporting logic expected token telemetry, but the execution path could still
persist synthetic placeholder counts.

This breaks the scientific intent of the lab:

- missing token telemetry can be mistaken for real measurement
- incomplete telemetry can be treated as zero in comparisons
- reports can overstate confidence in token deltas

VP5 must restore telemetry integrity without freezing the product into one
provider-specific runtime.

## Decision

Adopt a **measured execution telemetry** contract for the run ledger:

1. Token telemetry is recorded with an explicit `measurement_status`:
   - `complete`
   - `partial`
   - `unavailable`
2. Run telemetry must include a `source` describing where the measurement came
   from.
3. When known, the runtime/provider identity is recorded alongside the values.
4. Token fields (`input_tokens`, `output_tokens`, `total_tokens`) are only
   interpreted as comparable when `measurement_status = complete`.
5. When measurement is partial or unavailable, the run record must preserve
   explicit reasons rather than fabricating totals.
6. `graph lab report` must exclude incomplete token telemetry from strict token
   comparisons while still allowing quality, success, and resumability analysis.

The first concrete ingestion path is narrow:

- delegated outcome payloads consumed by `workflow finish`
- execution usage supplied in payload metadata under a reserved
  `execution_usage` property
- Copilot-CLI-first usage, with the ledger shape remaining provider-agnostic

## Consequences

### Positive

- The run ledger becomes scientifically honest about what was measured
- Reports stop treating missing token telemetry as zero
- Future provider integrations can plug into the same ledger/report model
- Partial runs remain useful for audit and quality analysis

### Negative

- Some existing experiments will lose token comparability until real telemetry is
  supplied
- The run schema becomes more explicit and slightly more verbose
- Consumers must reason about telemetry completeness instead of assuming every
  run has comparable token totals

### Risks

- Risk: callers omit execution telemetry and still expect token comparisons
  - Mitigation: persist explicit `partial` or `unavailable` states and surface
    report warnings
- Risk: provider/runtime payloads use incompatible usage semantics
  - Mitigation: keep `source` and `provider` explicit and avoid assuming all
    token fields mean the same thing across runtimes
- Risk: product teams are tempted to reintroduce estimates for convenience
  - Mitigation: explicitly forbid synthetic token totals in VP5

## Alternatives Considered

### Keep synthetic totals as a fallback

- Pros: simpler migration, always-populated reports
- Cons: destroys measurement integrity and makes token deltas untrustworthy
- Rejected because: false precision is worse than explicit missingness

### Fail every run that lacks authoritative token telemetry

- Pros: maximally strict data quality
- Cons: blocks useful quality/resumability evidence and makes early adoption too
  brittle
- Rejected because: VP5 should preserve partial evidence while clearly labeling
  it

### Provider-specific ledger schemas

- Pros: richer provider-native detail
- Cons: fragments reporting and locks the lab to current runtimes
- Rejected because: the ledger should stay stable while collectors evolve
