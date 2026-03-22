# TH2 — Graph Hygiene and Stats

## Theme Goal

Turn VP2 into an implementable backlog that keeps the shared graph both
measurable and maintainable through snapshot stats, suggest-first hygiene, and
explicit apply workflows.

## Scope

This theme covers:

- `graph stats` snapshot metrics
- graph-health indicators such as duplication, orphan, contradiction, and
  density/clustering signals
- suggest-first `graph hygiene`
- duplicate-near-identical detection
- orphan-node detection
- contradiction detection and resolution flows
- explicit hygiene apply workflows that remain revision- and diff-compatible

## Out of Scope

- continuous background cleanup daemons
- trend analytics or historical metric dashboards
- human visualization dashboards
- multi-repo hygiene or metrics federation
- automatic cleanup without explicit apply

## Epics

1. **TH2.E1 — Graph Stats Snapshot**
   Add `graph stats` so agents can inspect graph size and structural health
   before relying on the graph for retrieval-heavy work.

2. **TH2.E2 — Hygiene Suggestions**
   Add suggest-only graph hygiene for duplicate-near-identical nodes, orphan
   nodes, and contradictory facts with machine-readable explanations.

3. **TH2.E3 — Hygiene Apply and Revision Safety**
   Add explicit apply workflows that execute approved hygiene plans safely and
   preserve revision/diff inspectability.

## Dependency Flow

```text
E1 → E2 → E3
```

## Success Signal

An agent can run `graph stats` to understand whether a graph is tidy or chaotic,
run `graph hygiene` to receive structured cleanup suggestions, explicitly apply
selected fixes, and inspect the resulting cleanup through normal revision/diff
flows.
