# TH6.E1 — Task-Family Retrieval Policies

## Epic Goal

Give workflow start an explicit task-family policy layer so retrieval behavior
matches the type of work being delegated instead of relying on one global
kickoff strategy.

## Stories

- `TH6.E1.US1` — Classify workflow-start tasks into kickoff families
- `TH6.E1.US2` — Enforce family-specific entity allowlists and suppressions
- `TH6.E1.US3` — Skip kickoff by default for reporting and synthesis tasks

## Done When

- workflow start assigns delegated tasks to a documented kickoff family
- retrieval policies differ by family instead of sharing one global default
- reporting and synthesis tasks abstain by default unless a future policy explicitly overrides that stance

