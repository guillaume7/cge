---
id: TH4.E2.US1
title: "Add `graph lab run` with declared condition, model, and topology"
type: standard
priority: high
size: L
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "`graph lab run` accepts a task ID, condition ID, model, session topology, and seed, and executes a controlled run that composes the existing workflow primitives for graph-backed conditions."
  - AC2: "For non-graph (baseline) conditions, `graph lab run` executes the task without graph-backed kickoff or handoff."
  - AC3: "The command returns a machine-readable run summary with the run ID, declared parameters, and status on completion."
depends-on: [TH4.E1.US2, TH4.E1.US3]
---
# TH4.E2.US1 — Add `graph lab run` with declared condition, model, and topology

**As a** maintainer running experiments, **I want** a single command to execute
a controlled benchmark run under declared conditions, **so that** I can compare
graph-backed and baseline workflow without hand-scripting each execution.

## Acceptance Criteria

- [ ] AC1: `graph lab run` accepts a task ID, condition ID, model, session topology, and seed, and executes a controlled run that composes the existing workflow primitives for graph-backed conditions.
- [ ] AC2: For non-graph (baseline) conditions, `graph lab run` executes the task without graph-backed kickoff or handoff.
- [ ] AC3: The command returns a machine-readable run summary with the run ID, declared parameters, and status on completion.

## BDD Scenarios

### Scenario: Execute a graph-backed controlled run
- **Given** a valid suite manifest, condition manifest, and a graph-backed condition
- **When** a maintainer runs `graph lab run --task task-001 --condition with-graph --model claude-sonnet --topology delegated-parallel --seed 42`
- **Then** the command executes the task using workflow start and finish primitives and returns a machine-readable run summary with the assigned run ID

### Scenario: Execute a baseline controlled run without graph context
- **Given** a valid suite manifest and a baseline (non-graph) condition
- **When** a maintainer runs `graph lab run --task task-001 --condition without-graph --model claude-sonnet --topology delegated-parallel --seed 42`
- **Then** the command executes the task without graph-backed kickoff or handoff and returns a machine-readable run summary

### Scenario: Reject a run with an unknown task or condition ID
- **Given** the suite manifest does not contain the requested task ID or the condition manifest does not contain the requested condition ID
- **When** a maintainer runs `graph lab run` with the unknown identifiers
- **Then** the command returns a structured validation error identifying which ID was not found
