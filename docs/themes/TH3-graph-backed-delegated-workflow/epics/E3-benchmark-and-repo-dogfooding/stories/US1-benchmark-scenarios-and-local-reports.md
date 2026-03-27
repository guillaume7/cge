---
id: TH3.E3.US1
title: "Create delegated-workflow benchmark scenarios and local report artifacts"
type: standard
priority: medium
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "The benchmark harness can store comparable delegated-subtask scenarios for with-graph and without-graph workflow modes."
  - AC2: "Benchmark reports capture token or prompt volume, orientation effort, and outcome-quality or resumability signals for each run."
  - AC3: "Benchmark reports are stored as local machine-readable artifacts rather than durable graph entities."
depends-on: [TH3.E2.US3]
---
# TH3.E3.US1 — Create delegated-workflow benchmark scenarios and local report artifacts

**As a** product maintainer, **I want** comparable local benchmark artifacts, **so that**
VP3 can prove whether graph-backed delegation actually saves recovery cost.

## Acceptance Criteria

- [ ] AC1: The benchmark harness can store comparable delegated-subtask scenarios for with-graph and without-graph workflow modes.
- [ ] AC2: Benchmark reports capture token or prompt volume, orientation effort, and outcome-quality or resumability signals for each run.
- [ ] AC3: Benchmark reports are stored as local machine-readable artifacts rather than durable graph entities.

## BDD Scenarios

### Scenario: Record a comparable with-graph and without-graph benchmark pair
- **Given** a delegated-subtask benchmark scenario is defined for both workflow modes
- **When** a benchmark run is recorded locally
- **Then** the resulting artifacts preserve the scenario identity, workflow mode, and comparable metrics for later summary

### Scenario: Keep benchmark artifacts local and separate from graph persistence
- **Given** a benchmark run completes successfully
- **When** the benchmark report is written
- **Then** the report is stored as a local machine-readable artifact instead of being persisted as graph knowledge

### Scenario: Reject a benchmark report that omits required comparison metrics
- **Given** a benchmark result is missing required token, orientation, or quality fields
- **When** the benchmark report is recorded
- **Then** the harness returns a structured validation error instead of accepting an incomplete comparison artifact
