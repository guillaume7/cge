---
id: TH3.E3.US2
title: "Expose `graph workflow benchmark` summaries from local reports"
type: standard
priority: medium
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "A workflow benchmark command or subcommand can run or summarize delegated-workflow benchmark data from local artifacts."
  - AC2: "The benchmark summary compares with-graph and without-graph runs for the same scenario and flags incomplete or non-comparable data."
  - AC3: "Benchmark summary output is stable, machine-readable, and suitable for later release or review workflows."
depends-on: [TH3.E3.US1]
---
# TH3.E3.US2 — Expose `graph workflow benchmark` summaries from local reports

**As an** agent or maintainer, **I want** a CLI-facing benchmark summary, **so that** I can
see whether graph-backed delegated workflow is outperforming the non-graph
baseline without manually stitching together report files.

## Acceptance Criteria

- [ ] AC1: A workflow benchmark command or subcommand can run or summarize delegated-workflow benchmark data from local artifacts.
- [ ] AC2: The benchmark summary compares with-graph and without-graph runs for the same scenario and flags incomplete or non-comparable data.
- [ ] AC3: Benchmark summary output is stable, machine-readable, and suitable for later release or review workflows.

## BDD Scenarios

### Scenario: Summarize comparable benchmark results for a delegated subtask
- **Given** local benchmark artifacts exist for the same delegated-subtask scenario in both workflow modes
- **When** an agent runs the workflow benchmark summary command
- **Then** the CLI returns a machine-readable comparison of token, orientation, and quality-related metrics

### Scenario: Flag incomplete or non-comparable benchmark data
- **Given** local benchmark artifacts are missing one mode or required comparison fields
- **When** an agent runs the workflow benchmark summary command
- **Then** the CLI returns a machine-readable result that flags the comparison as incomplete instead of overstating confidence

### Scenario: Return a structured error when benchmark artifacts cannot be loaded
- **Given** the requested benchmark artifacts cannot be read or parsed
- **When** an agent runs the workflow benchmark summary command
- **Then** the CLI returns a structured operational error instead of partial benchmark output
