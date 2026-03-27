---
id: TH4.E1.US2
title: "Define benchmark suite and condition manifest schemas"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "The benchmark suite manifest schema supports a versioned task corpus with task IDs, families, descriptions, and acceptance criteria references."
  - AC2: "The condition manifest schema supports condition IDs, workflow mode (graph-backed or baseline), and blocking-factor declarations (task family, model, session topology)."
  - AC3: "Both manifests are validated on load; malformed or incomplete manifests produce structured validation errors."
depends-on: [TH4.E1.US1]
---
# TH4.E1.US2 — Define benchmark suite and condition manifest schemas

**As an** experiment designer, **I want** validated schemas for the benchmark
suite and condition definitions, **so that** every experiment starts from a
well-formed, machine-readable specification.

## Acceptance Criteria

- [ ] AC1: The benchmark suite manifest schema supports a versioned task corpus with task IDs, families, descriptions, and acceptance criteria references.
- [ ] AC2: The condition manifest schema supports condition IDs, workflow mode (graph-backed or baseline), and blocking-factor declarations (task family, model, session topology).
- [ ] AC3: Both manifests are validated on load; malformed or incomplete manifests produce structured validation errors.

## BDD Scenarios

### Scenario: Load a well-formed benchmark suite manifest
- **Given** a suite manifest file exists at `.graph/lab/suite.json` with valid schema version, tasks, and families
- **When** the lab loads the suite manifest
- **Then** the manifest parses successfully and all task definitions are accessible by task ID

### Scenario: Load a well-formed condition manifest
- **Given** a condition manifest file exists at `.graph/lab/conditions.json` with valid conditions and blocking factors
- **When** the lab loads the condition manifest
- **Then** the manifest parses successfully and each condition is accessible by condition ID with its declared workflow mode and factors

### Scenario: Reject a malformed suite manifest with a structured error
- **Given** a suite manifest is missing the schema version or contains a task without a task ID
- **When** the lab attempts to load the manifest
- **Then** the loader returns a structured validation error identifying the missing or invalid field

### Scenario: Reject a condition manifest with an unknown workflow mode
- **Given** a condition manifest contains a condition with an unrecognized workflow mode
- **When** the lab attempts to load the manifest
- **Then** the loader returns a structured validation error identifying the invalid workflow mode
