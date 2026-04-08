---
id: TH8.E4.US1
title: "Gate workflow-mediated memory writes on evaluator confidence"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "Memory writes originating from `workflow finish` pass through the Context Evaluator and Decision Engine before committing to the graph store."
  - AC2: "The Decision Engine applies a write-confidence threshold; writes below the threshold are deferred or skipped."
  - AC3: "Deferred writes are recorded in the attribution log with a reason and may be retried with additional evidence."
  - AC4: "Skipped writes are recorded in the attribution log with a reason and are not retried automatically."
depends-on:
  - TH8.E2.US3
---
# TH8.E4.US1 — Gate workflow-mediated memory writes on evaluator confidence

**As a** maintainer managing graph reliability, **I want** workflow-mediated
memory writes gated on evaluator confidence, **so that** low-quality or stale
state does not silently enter the graph.

## Acceptance Criteria

- [ ] AC1: Memory writes originating from `workflow finish` pass through the Context Evaluator and Decision Engine before committing to the graph store.
- [ ] AC2: The Decision Engine applies a write-confidence threshold; writes below the threshold are deferred or skipped.
- [ ] AC3: Deferred writes are recorded in the attribution log with a reason and may be retried with additional evidence.
- [ ] AC4: Skipped writes are recorded in the attribution log with a reason and are not retried automatically.

## BDD Scenarios

### Scenario: Approve a high-confidence write
- **Given** a `workflow finish` outcome with entities scored above the write threshold
- **When** the evaluator loop processes the write request
- **Then** the Decision Engine selects `write`
- **And** the entities are committed to the graph store
- **And** an attribution record is persisted with outcome `write`

### Scenario: Defer a moderate-confidence write
- **Given** a `workflow finish` outcome with entities scored between the defer and write thresholds
- **When** the evaluator loop processes the write request
- **Then** the write is deferred
- **And** an attribution record is persisted explaining the deferral reason
- **And** the entities are not committed to the graph store

### Scenario: Skip a low-confidence write
- **Given** a `workflow finish` outcome with entities scored below the defer threshold
- **When** the evaluator loop processes the write request
- **Then** the write is skipped
- **And** an attribution record is persisted explaining the skip reason
- **And** the entities are not committed to the graph store

### Scenario: Retry a deferred write with additional evidence
- **Given** a previously deferred write
- **And** additional evidence raises the confidence above the write threshold
- **When** the write is retried through the evaluator loop
- **Then** the Decision Engine selects `write`
- **And** the entities are committed to the graph store

### Scenario: Attribution records capture write gating decisions
- **Given** any workflow-mediated write attempt
- **When** the evaluator loop processes the request
- **Then** the attribution record includes the write confidence score, the threshold applied, and the memory decision (approved, deferred, or skipped)

## Notes

- Write gating applies only to workflow-mediated writes, not raw `graph write`
  commands (ADR-022 §4).
- The write threshold is configured alongside other Decision Engine thresholds
  (from TH8.E2.US2).
