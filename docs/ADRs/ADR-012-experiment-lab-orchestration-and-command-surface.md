# ADR-012: Local experiment-lab orchestration and `graph lab` command surface

## Status
Proposed

## Context

VP3 introduced a benchmark harness and CLI report surface for comparing
graph-backed and non-graph delegated workflow (ADR-011). That surface covers
scenario definitions and summary reports, but it does not provide:

- systematic condition assignment across tasks, models, and topologies
- randomized or counterbalanced run ordering
- repeated controlled runs with stable seed tracking
- a first-class experiment lifecycle beyond ad hoc benchmark invocations

VP4 asks a harder question than VP3: not just "does the graph help on this
scenario?" but "across controlled conditions and repeated runs, where does the
graph help, where is it neutral, and where does it impose unnecessary ceremony?"

Answering that question requires a local experiment lab that orchestrates
controlled batches rather than individual benchmark invocations.

## Decision

Add a **`graph lab` command group** as the VP4 experiment surface:

- `graph lab init` — create or refresh experiment assets (suite manifests,
  condition definitions, artifact directories, evaluation scaffolding)
- `graph lab run` — execute controlled runs with declared task, condition,
  model, topology, and seed
- `graph lab report` — aggregate completed runs into scientific-style reports

The lab orchestrator should:

1. compose VP3 workflow primitives (`workflow start`, `workflow finish`) for
   actual task execution under graph-backed conditions
2. compose the existing benchmark harness for scenario structure
3. add experiment-level orchestration: condition assignment, randomization,
   blocking-factor awareness, and batch lifecycle
4. remain a thin local layer — no daemon, no hosted backend, no external
   orchestration service

The `graph lab` commands sit alongside `graph workflow` as a peer command group,
not as a replacement. `graph workflow` remains the operational surface;
`graph lab` is the scientific evaluation surface.

## Consequences

### Positive
- Creates a structured experiment lifecycle without ad hoc scripting
- Enables paired within-task comparisons and multi-factor designs
- Reuses existing workflow and benchmark infrastructure
- Keeps experiment control local, inspectable, and repo-scoped

### Negative
- Adds a new command group and orchestration layer to the CLI surface
- Condition assignment and randomization logic must be implemented carefully
- Experiment design complexity could exceed what a simple CLI naturally supports

### Risks
- Risk: the lab orchestrator grows into a full experiment platform
  - Mitigation: keep VP4 scoped to the narrow with-graph vs without-graph
    hypothesis and a small set of blocking factors (task, model, topology)
- Risk: command surface overlap with `graph workflow benchmark`
  - Mitigation: `graph lab` subsumes the controlled-experiment use case;
    `graph workflow benchmark` remains available for quick single-scenario checks

## Alternatives Considered

### Extend `graph workflow benchmark` instead of a new command group
- Pros: no new command group, simpler surface
- Cons: overloads the benchmark command with experiment lifecycle concerns that
  are distinct from single-scenario benchmarking
- Rejected because: experiment orchestration (batches, conditions, randomization,
  multi-run) is a different concern from scenario-level benchmarking

### External experiment runner (shell scripts or CI orchestration)
- Pros: no new CLI code, maximum flexibility
- Cons: loses reproducibility guarantees, condition tracking becomes ad hoc,
  reporting requires custom glue
- Rejected because: the experiment lifecycle must be a first-class product
  surface to enforce protocol discipline

### Full hosted experiment platform
- Pros: richer collaboration, dashboards, historical comparison
- Cons: hosted complexity, privacy, operational overhead
- Rejected because: VP4 must stay local and inspectable per product principles
