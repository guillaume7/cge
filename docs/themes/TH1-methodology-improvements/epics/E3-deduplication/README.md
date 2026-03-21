# E3 — Instruction Deduplication

Remove duplication across copilot-instructions.md, skills, and agent files. Establish a clear hierarchy: entry point → skills → thin agents.

## Stories

| ID | Title | Priority | Depends On |
|----|-------|----------|------------|
| TH1.E3.US1 | DRY copilot-instructions.md | P1 | TH1.E2.US1 |
| TH1.E3.US2 | Make agents thin, skills canonical | P2 | TH1.E3.US1 |
| TH1.E3.US3 | Consolidate pass-through prompts | P3 | TH1.E3.US2 |
