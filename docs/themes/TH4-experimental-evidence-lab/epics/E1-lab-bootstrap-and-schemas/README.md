# TH4.E1 — Lab Bootstrap and Schemas

## Epic Goal

Give the experiment lab a stable foundation by adding a bootstrap command and
defining the schema contracts that all downstream lab operations depend on.

## Stories

- `TH4.E1.US1` — Add `graph lab init` and experiment asset scaffolding
- `TH4.E1.US2` — Define benchmark suite and condition manifest schemas
- `TH4.E1.US3` — Create run record and evaluation record schema contracts

## Done When

- `graph lab init` can create or refresh the `.graph/lab/` directory structure
  with suite manifest, condition definitions, and evaluation scaffolding
- suite and condition manifest schemas are validated on load
- run record and evaluation record schemas are defined and validated so
  downstream run and evaluation operations have a stable contract
