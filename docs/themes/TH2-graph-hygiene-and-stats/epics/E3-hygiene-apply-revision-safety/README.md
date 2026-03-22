# TH2.E3 — Hygiene Apply and Revision Safety

## Epic Goal

Execute approved hygiene changes explicitly while preserving graph trust through
revision anchors, rollback-safe behavior, and diff compatibility.

## Stories

- `TH2.E3.US1` — Apply explicit hygiene plans and return revision anchors
- `TH2.E3.US2` — Keep hygiene changes inspectable through revision diff
- `TH2.E3.US3` — Reject stale or unsafe hygiene plans without mutating the graph

## Done When

- hygiene apply mode executes only explicit requested actions
- applied changes produce inspectable revisions
- invalid or stale plans fail safely without partial mutation
