# ADR-013: Artifact-first run ledger and scientific reporting model

## Status
Proposed

## Context

VP4 experiments must produce reproducible, auditable evidence. Each run involves
multiple moving parts — task definition, experimental condition, model identity,
session topology, kickoff inputs, delegated session artifacts, token/usage
telemetry, timing, and outcome artifacts.

If run data is only available transiently (e.g. only in CLI output), the
experiment cannot be audited, replayed, or aggregated into statistical reports.

VP3's benchmark reports (ADR-011) stored scenario-level results as local JSON
artifacts. VP4 needs a more structured approach: a run ledger that captures
complete per-run records and a reporting layer that derives scientific summaries
from the ledger.

## Decision

Adopt an **artifact-first run ledger** model:

1. **Run ledger** — every controlled run produces a self-contained,
   machine-readable run record stored locally under `.graph/lab/runs/`
2. **Run record** — each record captures the full experimental context:
   task ID, condition, model, topology, seed, kickoff inputs, session
   structure, writeback outputs, token/usage telemetry, timing, retry
   counts, and outcome artifacts
3. **Immutable run artifacts** — once a run completes, its record is written
   once and not modified; evaluation scores are stored as separate evaluation
   records linked to the run ID
4. **Report generation** — `graph lab report` reads the run ledger and
   evaluation records to produce aggregate reports; reports are derived
   artifacts, not the system of record

The run ledger layout should be:

```text
.graph/lab/
  suite.json              # benchmark suite manifest
  conditions.json         # condition definitions
  runs/
    <run-id>/
      run.json            # complete run record
      artifacts/          # preserved outcome artifacts
  evaluations/
    <run-id>.json         # evaluation scores for a run
  reports/
    <report-id>.json      # generated aggregate reports
```

Reports should support:

- paired within-task comparisons (same task, different conditions)
- grouped comparisons by model or topology
- success/failure rates
- token and step distributions
- effect-size summaries with uncertainty intervals
- resumability and handoff quality comparisons

## Consequences

### Positive
- Every experimental claim is traceable to concrete run artifacts
- Reports are reproducible — regenerating from the same ledger produces the same
  analysis
- The immutable run / separate evaluation split enables blinded scoring
- Local filesystem storage keeps the ledger inspectable without tooling

### Negative
- Run artifacts consume local disk space proportional to experiment scale
- The ledger format is a new schema contract that must be maintained
- Aggregation logic for statistical summaries adds implementation complexity

### Risks
- Risk: run records become bloated with large outcome artifacts
  - Mitigation: store bulky artifacts by reference (file path) rather than
    inline; keep run.json focused on structured telemetry
- Risk: ledger format changes break report generation
  - Mitigation: version the run record schema and require backward-compatible
    report readers

## Alternatives Considered

### Reuse VP3 benchmark report format as-is
- Pros: no new schema, immediate reuse
- Cons: VP3 reports are scenario-level summaries without per-run detail,
  condition tracking, or evaluation separation
- Rejected because: VP4 requires run-level granularity for paired comparison
  and statistical analysis

### Store run data in the graph database
- Pros: single system of record, queryable with Cypher
- Cons: conflates experiment metadata with domain knowledge, bloats the graph
  with transient benchmark data
- Rejected because: experiment runs are evaluation artifacts, not durable graph
  knowledge (consistent with VP3's modeling rule)

### External database (SQLite, etc.) for the run ledger
- Pros: richer querying, better aggregation primitives
- Cons: adds a dependency, complicates the single-binary distribution story
- Rejected because: the run ledger is small enough for filesystem-based storage
  in VP4's scope; a database can be reconsidered if experiment scale demands it
