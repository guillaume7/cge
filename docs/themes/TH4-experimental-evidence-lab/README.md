# TH4 — Experimental Evidence Lab

## Theme Goal

Turn VP4 into an implementable backlog that gives CGE a local, reproducible
experiment lab for comparing graph-backed vs non-graph delegated workflow under
controlled conditions across models and session topologies.

## Scope

This theme covers:

- `graph lab init` for experiment asset scaffolding, suite manifests, condition
  definitions, and schema contracts
- `graph lab run` for controlled benchmark execution with declared conditions,
  models, topologies, and seed-based reproducibility
- `graph lab report` for aggregate scientific-style reporting with paired
  comparisons, effect sizes, and uncertainty intervals
- separated evaluation scoring with blinding support
- immutable run records and outcome artifact capture under `.graph/lab/`
- repo-local dogfooding of the experiment harness on this repo's own
  delegated-workflow tasks

## Out of Scope

- global public leaderboards or hosted telemetry backends
- autonomous prompt optimization loops
- arbitrary internet-scale benchmark task ingestion
- multi-repo federation of benchmark data
- proving causal claims beyond what the measured task suite supports

## Epics

1. **TH4.E1 — Lab Bootstrap and Schemas**
   Add `graph lab init`, benchmark suite and condition manifest schemas, and run
   record and evaluation record schema contracts so the experiment lab has a
   stable foundation.

2. **TH4.E2 — Controlled Run Orchestration and Ledger**
   Add `graph lab run` with declared conditions, immutable run ledger
   persistence, outcome artifact capture, and condition randomization so
   experiments are reproducible and auditable.

3. **TH4.E3 — Evaluation, Reporting, and Repo Dogfooding**
   Add separated evaluation scoring, `graph lab report` with paired comparisons
   and statistical summaries, and repo-local dogfooding so evidence is
   scientifically legible and proven on this repo's own workflow.

## Dependency Flow

```text
E1 → E2 → E3
```

## Success Signal

A maintainer can define a benchmark suite of realistic repo tasks, run the same
tasks under graph-backed and non-graph conditions across multiple models, collect
immutable run artifacts and separated evaluation scores, and generate a report
that surfaces effect sizes, uncertainty, and practical recommendations about
where graph-backed workflow helps, is neutral, or adds unnecessary ceremony.
