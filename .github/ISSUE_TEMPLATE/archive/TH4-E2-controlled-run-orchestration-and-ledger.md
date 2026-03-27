---
name: "TH4 E2 — Controlled Run Orchestration and Ledger"
about: "Implement TH4.E2 Controlled Run Orchestration and Ledger"
labels: [epic, TH4, E2]
assignees: [copilot]
---

- [ ] US1 — Add `graph lab run` with declared condition, model, and topology: execute controlled runs composing workflow primitives for graph-backed conditions and skipping them for baseline.
- [ ] US2 — Persist immutable run records and outcome artifacts to the run ledger: write once run records with telemetry and preserved artifacts under `.graph/lab/runs/`.
- [ ] US3 — Support condition randomization and seed-based reproducibility: shuffle condition ordering deterministically and persist the batch plan before execution.

Full stories: `docs/themes/TH4-experimental-evidence-lab/epics/E2-controlled-run-orchestration-and-ledger/stories/`
