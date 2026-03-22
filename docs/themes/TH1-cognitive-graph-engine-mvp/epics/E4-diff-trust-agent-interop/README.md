# TH1.E4 — Diff, Trust, and Agent Interoperability

## Epic Goal

Finish the MVP by ensuring every command participates in a stable native
machine-to-machine contract, supports graph diffs, and works in real chained
agent workflows.

## Stories

- `TH1.E4.US1` — Return a consistent machine-readable command contract
- `TH1.E4.US2` — Compare graph revisions with diff
- `TH1.E4.US3` — Verify end-to-end chainable agent workflows

## Done When

- command outputs remain consistent across stdin/stdout/file workflows
- `graph diff` can compare revision anchors meaningfully
- common chained agent flows work end to end without ad hoc glue code
