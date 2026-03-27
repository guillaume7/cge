---
id: TH4.E1.US3
title: "Create run record and evaluation record schema contracts"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "The run record schema captures task ID, condition ID, model, session topology, seed, prompt variant, timing, token/usage telemetry, and artifact references as a versioned, machine-readable contract."
  - AC2: "The evaluation record schema captures run ID reference, evaluator identity, success/failure, quality score, resumability score, human intervention count, and evaluation timestamp."
  - AC3: "Both schemas are validated before write; records that violate the schema are rejected with structured errors."
depends-on: [TH4.E1.US1]
---
# TH4.E1.US3 — Create run record and evaluation record schema contracts

**As a** lab implementer, **I want** stable schema contracts for run records and
evaluation records, **so that** downstream run orchestration, evaluation, and
reporting can depend on a consistent data shape.

## Acceptance Criteria

- [ ] AC1: The run record schema captures task ID, condition ID, model, session topology, seed, prompt variant, timing, token/usage telemetry, and artifact references as a versioned, machine-readable contract.
- [ ] AC2: The evaluation record schema captures run ID reference, evaluator identity, success/failure, quality score, resumability score, human intervention count, and evaluation timestamp.
- [ ] AC3: Both schemas are validated before write; records that violate the schema are rejected with structured errors.

## BDD Scenarios

### Scenario: Validate a complete run record against the schema
- **Given** a run record contains all required fields including task ID, condition, model, topology, seed, telemetry, and timing
- **When** the record is validated against the run record schema
- **Then** validation passes and the record is accepted

### Scenario: Reject a run record missing required telemetry fields
- **Given** a run record is missing the total token count or wall-clock timing
- **When** the record is validated against the run record schema
- **Then** validation fails with a structured error identifying the missing fields

### Scenario: Validate a complete evaluation record against the schema
- **Given** an evaluation record contains a valid run ID reference, evaluator identity, success flag, quality score, resumability score, and timestamp
- **When** the record is validated against the evaluation record schema
- **Then** validation passes and the record is accepted

### Scenario: Reject an evaluation record with an invalid run ID reference
- **Given** an evaluation record references a run ID that does not exist in the run ledger
- **When** the record is validated
- **Then** validation fails with a structured error identifying the dangling run reference
