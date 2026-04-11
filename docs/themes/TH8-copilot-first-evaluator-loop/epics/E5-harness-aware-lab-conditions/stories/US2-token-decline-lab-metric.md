---
id: TH8.E5.US2
title: "Surface token-decline comparisons in lab reports"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "Lab reports surface token consumption comparisons across harness-aware conditions as a first-class metric."
  - AC2: "Reports show total token delta between `with-harness` and `without-harness` conditions."
  - AC3: "Reports show per-task token distributions for each condition."
  - AC4: "Reports include confidence intervals on the token delta when sufficient runs exist."
depends-on:
  - TH8.E5.US1
---
# TH8.E5.US2 — Surface token-decline comparisons in lab reports

**As a** maintainer evaluating VP8 success, **I want** lab reports to surface
token-decline comparisons as a primary metric, **so that** I can verify whether
the CGE harness reduces token consumption.

## Acceptance Criteria

- [ ] AC1: Lab reports surface token consumption comparisons across harness-aware conditions as a first-class metric.
- [ ] AC2: Reports show total token delta between `with-harness` and `without-harness` conditions.
- [ ] AC3: Reports show per-task token distributions for each condition.
- [ ] AC4: Reports include confidence intervals on the token delta when sufficient runs exist.

## BDD Scenarios

### Scenario: Report shows token delta between with-harness and without-harness
- **Given** a completed lab batch with runs in both `with-harness` and `without-harness` conditions
- **When** `graph lab report` generates a report
- **Then** the report includes a token-delta section showing the difference in total tokens consumed
- **And** the delta is expressed as both absolute and percentage values

### Scenario: Report shows per-task token distributions
- **Given** a completed lab batch with multiple tasks run under each condition
- **When** `graph lab report` generates a report
- **Then** the report includes per-task token consumption for each condition
- **And** tasks are grouped so paired comparisons are visible

### Scenario: Report includes confidence intervals
- **Given** a completed lab batch with at least 10 runs per condition
- **When** `graph lab report` generates a report
- **Then** the report includes a 95% confidence interval on the mean token delta

### Scenario: Report with insufficient runs omits confidence intervals
- **Given** a completed lab batch with only 2 runs per condition
- **When** `graph lab report` generates a report
- **Then** the report shows the token delta without confidence intervals
- **And** a note indicates that insufficient runs prevent interval estimation

### Scenario: Report includes graph-only baseline when present
- **Given** a completed lab batch with `with-harness`, `without-harness`, and `graph-only` conditions
- **When** `graph lab report` generates a report
- **Then** the report shows token deltas for all three pairwise comparisons

## Notes

- Token telemetry depends on ADR-015's measurement_status contract; runs with
  unavailable telemetry are excluded from comparisons (ADR-023 §risk).
- Token-decline is the primary VP8 success signal.
