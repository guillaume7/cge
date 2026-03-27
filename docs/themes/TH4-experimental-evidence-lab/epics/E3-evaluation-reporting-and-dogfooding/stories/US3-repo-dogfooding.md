---
id: TH4.E3.US3
title: "Dogfood the experiment lab on this repo's delegated-workflow tasks"
type: standard
priority: medium
size: L
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "This repo's benchmark suite manifest contains at least two realistic delegated-workflow tasks drawn from its own engineering history."
  - AC2: "A complete lab lifecycle (init → run → evaluate → report) can be executed end to end in this repo without inventing ad hoc tooling."
  - AC3: "The generated report surfaces actionable comparison evidence for graph-backed vs non-graph workflow on this repo's own tasks."
depends-on: [TH4.E3.US2]
---
# TH4.E3.US3 — Dogfood the experiment lab on this repo's delegated-workflow tasks

**As a** maintainer dogfooding CGE, **I want** to exercise the full experiment
lab on this repo's real delegated-workflow tasks, **so that** VP4 is proven in
practice before it is recommended for other repos.

## Acceptance Criteria

- [ ] AC1: This repo's benchmark suite manifest contains at least two realistic delegated-workflow tasks drawn from its own engineering history.
- [ ] AC2: A complete lab lifecycle (init → run → evaluate → report) can be executed end to end in this repo without inventing ad hoc tooling.
- [ ] AC3: The generated report surfaces actionable comparison evidence for graph-backed vs non-graph workflow on this repo's own tasks.

## BDD Scenarios

### Scenario: Define a benchmark suite from this repo's engineering tasks
- **Given** this repo has real delegated-workflow tasks from its implementation history
- **When** the suite manifest is populated with representative task definitions
- **Then** the manifest contains at least two task entries with acceptance criteria references pointing to repo-local artifacts

### Scenario: Execute an end-to-end lab lifecycle on repo tasks
- **Given** the lab has been initialized and the suite and condition manifests are populated
- **When** a maintainer runs the full lifecycle: `graph lab init`, `graph lab run` for multiple conditions, evaluation scoring, and `graph lab report`
- **Then** the lifecycle completes without errors, produces immutable run records and evaluation scores, and generates an aggregate report

### Scenario: Report surfaces evidence relevant to this repo's workflow decisions
- **Given** the lab has completed runs under graph-backed and baseline conditions for this repo's tasks
- **When** the aggregate report is generated
- **Then** the report includes paired task comparisons with token deltas, quality and resumability effect sizes, and practical recommendations applicable to this repo
