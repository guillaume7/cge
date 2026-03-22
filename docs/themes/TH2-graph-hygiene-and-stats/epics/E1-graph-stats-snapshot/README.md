# TH2.E1 — Graph Stats Snapshot

## Epic Goal

Expose a cheap, machine-readable graph-health snapshot so agents can decide when
the graph is safe to trust and when hygiene should come first.

## Stories

- `TH2.E1.US1` — Add a graph stats command and snapshot counts
- `TH2.E1.US2` — Compute cognitive health indicators from the current graph snapshot

## Done When

- `graph stats` returns node and relationship counts
- the command includes agreed health indicators
- the output is stable enough for downstream agent automation
