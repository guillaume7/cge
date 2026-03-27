# TH4.E3 — Evaluation, Reporting, and Repo Dogfooding

## Epic Goal

Close the experiment loop by adding separated evaluation scoring, aggregate
scientific reporting, and repo-local verification that proves the lab works on
this repo's own delegated-workflow tasks.

## Stories

- `TH4.E3.US1` — Add separated evaluation scoring with blinding support
- `TH4.E3.US2` — Add `graph lab report` with paired comparisons and uncertainty
- `TH4.E3.US3` — Dogfood the experiment lab on this repo's delegated-workflow tasks

## Done When

- evaluation scores are stored separately from run records and support
  condition-blind presentation
- `graph lab report` generates aggregate reports with paired comparisons,
  effect sizes, uncertainty intervals, and explicit null-result support
- this repo has exercised the full lab lifecycle (init, run, evaluate, report)
  on its own delegated-workflow tasks

## Repo-local example

- `repo-dogfooding-example.md` documents the committed repo-local harness,
  baseline artifacts, and honesty notes for the tiny sample.
