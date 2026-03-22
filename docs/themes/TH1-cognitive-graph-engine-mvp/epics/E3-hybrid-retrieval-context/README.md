# TH1.E3 — Hybrid Retrieval and Context

## Epic Goal

Provide task-relevant graph retrieval that balances continuity, token economy,
and trust using local graph traversal, text relevance, context projection, and
explanation.

## Stories

- `TH1.E3.US1` — Build hybrid query retrieval
- `TH1.E3.US2` — Project compact task context
- `TH1.E3.US3` — Explain retrieval results and provenance

## Done When

- `graph query` can return ranked structured results
- `graph context` can shape those results into a token-budgeted context envelope
- `graph explain` can justify why the CLI returned a given context slice
