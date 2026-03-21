# E1 — Core Schema & State

Foundational fixes to the backlog data model: qualified IDs, single source of truth, pure YAML format.

These changes affect every downstream epic — all other epics depend on E1.

## Stories

| ID | Title | Priority | Depends On |
|----|-------|----------|------------|
| TH1.E1.US1 | Introduce qualified story IDs | P0 | — |
| TH1.E1.US2 | Single source of truth for status | P1 | TH1.E1.US1 |
| TH1.E1.US3 | Convert backlog to pure YAML | P2 | TH1.E1.US2 |
