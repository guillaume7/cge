---
id: TH8.E5.US1
title: "Add harness-aware experiment conditions"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "The experiment lab supports three new condition types: `with-harness` (full CGE pipeline), `without-harness` (no CGE involvement), and `graph-only` (retrieval and projection without the evaluator loop)."
  - AC2: "Existing `with-graph` and `without-graph` condition definitions remain valid and unchanged."
  - AC3: "New conditions use new condition IDs and are additive to the existing condition model."
  - AC4: "Lab run records correctly tag each run with the applicable condition."
depends-on:
  - TH8.E3.US3
  - TH8.E4.US3
---
# TH8.E5.US1 — Add harness-aware experiment conditions

**As a** lab experiment designer, **I want** harness-aware conditions that
distinguish full CGE pipeline runs from graph-only and no-CGE runs, **so that**
I can measure the evaluator loop's contribution separately from raw graph
injection.

## Acceptance Criteria

- [ ] AC1: The experiment lab supports three new condition types: `with-harness` (full CGE pipeline), `without-harness` (no CGE involvement), and `graph-only` (retrieval and projection without the evaluator loop).
- [ ] AC2: Existing `with-graph` and `without-graph` condition definitions remain valid and unchanged.
- [ ] AC3: New conditions use new condition IDs and are additive to the existing condition model.
- [ ] AC4: Lab run records correctly tag each run with the applicable condition.

## BDD Scenarios

### Scenario: Run an experiment with the with-harness condition
- **Given** a lab suite with a `with-harness` condition defined
- **When** `graph lab run` executes a task under the `with-harness` condition
- **Then** the run uses the full CGE pipeline (retrieval + evaluation + decision + attribution)
- **And** the run record tags the condition as `with-harness`

### Scenario: Run an experiment with the without-harness condition
- **Given** a lab suite with a `without-harness` condition defined
- **When** `graph lab run` executes a task under the `without-harness` condition
- **Then** the run operates with no CGE involvement
- **And** the run record tags the condition as `without-harness`

### Scenario: Run an experiment with the graph-only condition
- **Given** a lab suite with a `graph-only` condition defined
- **When** `graph lab run` executes a task under the `graph-only` condition
- **Then** the run uses graph retrieval and projection but skips the evaluator loop
- **And** the run record tags the condition as `graph-only`

### Scenario: Existing conditions remain valid
- **Given** a lab suite defined with legacy `with-graph` and `without-graph` conditions
- **When** `graph lab run` executes the suite
- **Then** the existing conditions work exactly as before
- **And** no errors or deprecation warnings are emitted

### Scenario: Mixed conditions in one suite
- **Given** a lab suite with `with-harness`, `without-harness`, and `graph-only` conditions
- **When** `graph lab run` executes the suite
- **Then** each run is tagged with its correct condition
- **And** the run ledger contains runs for all three conditions

## Notes

- Condition IDs are free-form strings (ADR-023 §5); new conditions are additive.
- The `graph-only` condition serves as a regression baseline showing pre-VP8
  behavior.
