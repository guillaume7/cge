---
id: TH4.E2.US2
title: "Persist immutable run records and outcome artifacts to the run ledger"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "Each completed run produces an immutable run record under `.graph/lab/runs/<run-id>/run.json` with full experimental context, telemetry, and artifact references."
  - AC2: "Outcome artifacts (kickoff inputs, session structure, writeback outputs) are preserved alongside the run record under `.graph/lab/runs/<run-id>/artifacts/`."
  - AC3: "Run records cannot be overwritten after completion; attempting to write a duplicate run ID produces a structured error."
depends-on: [TH4.E2.US1]
---
# TH4.E2.US2 — Persist immutable run records and outcome artifacts to the run ledger

**As an** experiment auditor, **I want** every completed run to produce an
immutable, self-contained record with preserved artifacts, **so that** results
can be inspected, re-evaluated, and reproduced later.

## Acceptance Criteria

- [ ] AC1: Each completed run produces an immutable run record under `.graph/lab/runs/<run-id>/run.json` with full experimental context, telemetry, and artifact references.
- [ ] AC2: Outcome artifacts (kickoff inputs, session structure, writeback outputs) are preserved alongside the run record under `.graph/lab/runs/<run-id>/artifacts/`.
- [ ] AC3: Run records cannot be overwritten after completion; attempting to write a duplicate run ID produces a structured error.

## BDD Scenarios

### Scenario: Persist a complete run record with telemetry and artifact references
- **Given** a controlled run has completed successfully
- **When** the run ledger persists the result
- **Then** a `run.json` file is written under `.graph/lab/runs/<run-id>/` containing the task ID, condition, model, topology, seed, timing, token telemetry, and artifact path references

### Scenario: Preserve outcome artifacts alongside the run record
- **Given** a controlled run has produced kickoff inputs and writeback outputs
- **When** the run ledger persists the result
- **Then** the outcome artifacts are stored under `.graph/lab/runs/<run-id>/artifacts/` and referenced from the run record

### Scenario: Reject an attempt to overwrite an existing run record
- **Given** a run record already exists for a given run ID
- **When** a new result attempts to write to the same run ID
- **Then** the ledger returns a structured error and does not modify the existing record
