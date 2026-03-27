# TH4.E2 — Controlled Run Orchestration and Ledger

## Epic Goal

Make controlled experiment execution reproducible and auditable by adding a run
command that captures immutable records and outcome artifacts under declared
experimental conditions.

## Stories

- `TH4.E2.US1` — Add `graph lab run` with declared condition, model, and topology
- `TH4.E2.US2` — Persist immutable run records and outcome artifacts to the run ledger
- `TH4.E2.US3` — Support condition randomization and seed-based reproducibility

## Done When

- `graph lab run` can execute a controlled run with a declared task, condition,
  model, topology, and seed
- each completed run produces an immutable run record with complete telemetry
  and artifact references under `.graph/lab/runs/`
- condition ordering can be randomized or counterbalanced with deterministic
  seed tracking for reproducibility
