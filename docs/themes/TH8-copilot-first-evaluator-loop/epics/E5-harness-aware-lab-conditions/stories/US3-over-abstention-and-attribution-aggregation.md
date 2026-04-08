---
id: TH8.E5.US3
title: "Detect over-abstention and aggregate attribution in lab reports"
type: standard
priority: medium
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "Lab reports include the abstention rate across `with-harness` runs."
  - AC2: "Lab reports flag when token savings coincide with quality regression, indicating possible over-abstention."
  - AC3: "Lab reports aggregate attribution records to show the distribution of decision outcomes (continue, minimal, abstain, backtrack, write) across runs."
  - AC4: "Lab reports include per-task-family decision pattern summaries when family data is available."
depends-on:
  - TH8.E5.US2
---
# TH8.E5.US3 — Detect over-abstention and aggregate attribution in lab reports

**As a** maintainer tuning evaluator thresholds, **I want** lab reports to
detect over-abstention and aggregate attribution data, **so that** I can
distinguish genuine improvement from suppressed behavior that harms quality.

## Acceptance Criteria

- [ ] AC1: Lab reports include the abstention rate across `with-harness` runs.
- [ ] AC2: Lab reports flag when token savings coincide with quality regression, indicating possible over-abstention.
- [ ] AC3: Lab reports aggregate attribution records to show the distribution of decision outcomes (continue, minimal, abstain, backtrack, write) across runs.
- [ ] AC4: Lab reports include per-task-family decision pattern summaries when family data is available.

## BDD Scenarios

### Scenario: Report shows abstention rate
- **Given** a completed lab batch with 20 `with-harness` runs, 8 of which had `abstain` outcomes
- **When** `graph lab report` generates a report
- **Then** the report shows an abstention rate of 40%

### Scenario: Flag over-abstention when quality regresses
- **Given** a lab batch where `with-harness` runs have lower token consumption but also lower quality scores than `without-harness` runs
- **When** `graph lab report` generates a report
- **Then** the report includes an over-abstention warning
- **And** the warning cites the correlation between abstention rate and quality regression

### Scenario: No over-abstention flag when quality holds
- **Given** a lab batch where `with-harness` runs have lower token consumption and equal or better quality scores
- **When** `graph lab report` generates a report
- **Then** no over-abstention warning is present

### Scenario: Aggregate decision outcome distribution
- **Given** a completed lab batch with attribution records for 20 `with-harness` runs
- **When** `graph lab report` generates a report
- **Then** the report includes a decision outcome distribution showing counts and percentages for each outcome type

### Scenario: Per-task-family decision patterns
- **Given** a completed lab batch with runs across three task families (implementation, verification, diagnosis)
- **And** attribution records are available for each run
- **When** `graph lab report` generates a report
- **Then** the report includes per-family decision pattern summaries
- **And** each family shows its own outcome distribution

### Scenario: Attribution aggregation with missing records
- **Given** a lab batch where some runs lack attribution records (e.g., graph-only condition runs)
- **When** `graph lab report` generates a report
- **Then** attribution aggregation covers only runs with records
- **And** a note indicates how many runs lacked attribution data

## Notes

- Over-abstention detection is a core VP8 lab requirement (ADR-023 §3).
- Attribution aggregation keeps computation simple: counts and distributions,
  no expensive cross-record joins (ADR-023 §risk).
