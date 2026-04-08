---
id: TH8.E3.US1
title: "Generate structured attribution records for every decision"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "The Attribution Recorder generates a structured attribution record every time the Decision Engine selects an outcome."
  - AC2: "The attribution record contains: decision outcome, evaluator scores per dimension, composite confidence, per-candidate fate (survived, trimmed, rejected with reason), memory decision (write approved, deferred, skipped with reason), timestamp, and task context."
  - AC3: "The record includes a compact inline summary suitable for inclusion in the decision envelope."
  - AC4: "Attribution records are generated for all five outcome types without exception."
depends-on:
  - TH8.E2.US3
---
# TH8.E3.US1 — Generate structured attribution records for every decision

**As a** maintainer analyzing evaluator-loop behavior, **I want** every decision
to produce a structured attribution record, **so that** I can trace why
guidance was injected, minimized, rejected, or persisted.

## Acceptance Criteria

- [ ] AC1: The Attribution Recorder generates a structured attribution record every time the Decision Engine selects an outcome.
- [ ] AC2: The attribution record contains: decision outcome, evaluator scores per dimension, composite confidence, per-candidate fate (survived, trimmed, rejected with reason), memory decision (write approved, deferred, skipped with reason), timestamp, and task context.
- [ ] AC3: The record includes a compact inline summary suitable for inclusion in the decision envelope.
- [ ] AC4: Attribution records are generated for all five outcome types without exception.

## BDD Scenarios

### Scenario: Attribution record for a continue decision
- **Given** the Decision Engine selects `continue` with three candidates surviving
- **When** the Attribution Recorder generates a record
- **Then** the record contains outcome `continue`, per-candidate fates showing all three survived, and the composite confidence score

### Scenario: Attribution record for a minimal decision with trimmed candidates
- **Given** the Decision Engine selects `minimal`, trimming two of four candidates
- **When** the Attribution Recorder generates a record
- **Then** the record contains outcome `minimal`, two candidates marked as survived, two marked as trimmed with per-candidate reasons

### Scenario: Attribution record for an abstain decision
- **Given** the Decision Engine selects `abstain`
- **When** the Attribution Recorder generates a record
- **Then** the record contains outcome `abstain`, all candidates marked as rejected, and the evaluator scores that motivated the abstention

### Scenario: Attribution record includes memory decision
- **Given** the Decision Engine selects `write` for a memory update
- **When** the Attribution Recorder generates a record
- **Then** the record contains outcome `write` and a memory-decision field indicating write-approved with the confidence score

### Scenario: Inline summary is compact
- **Given** any attribution record
- **When** the inline summary is extracted
- **Then** the summary is a single JSON object under 500 bytes containing the outcome, composite confidence, and candidate count

## Notes

- Attribution records extend the existing provenance model (ADR-004) without
  replacing it (ADR-021 §5).
- Full records are persisted separately from inline summaries.
