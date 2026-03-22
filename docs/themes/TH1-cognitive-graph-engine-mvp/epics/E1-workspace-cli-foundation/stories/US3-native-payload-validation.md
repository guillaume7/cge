---
id: TH1.E1.US3
title: "Define and validate the native graph payload contract"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "The CLI accepts a versioned native JSON envelope for graph payloads across file input and stdin."
  - AC2: "Validation rejects malformed payloads and missing required provenance fields such as agent ID, session ID, timestamp, and schema version."
  - AC3: "Validation errors are returned as structured machine-readable responses that preserve chainability."
depends-on: [TH1.E1.US2]
---
# TH1.E1.US3 — Define and validate the native graph payload contract

**As an** agent, **I want** a stable native graph payload contract with strict validation, **so that** agents trained on the format can exchange graph data reliably and losslessly.

## Acceptance Criteria

- [ ] AC1: The CLI accepts a versioned native JSON envelope for graph payloads across file input and stdin.
- [ ] AC2: Validation rejects malformed payloads and missing required provenance fields such as agent ID, session ID, timestamp, and schema version.
- [ ] AC3: Validation errors are returned as structured machine-readable responses that preserve chainability.

## BDD Scenarios

### Scenario: Accept a valid native graph payload
- **Given** a JSON payload that matches the native graph schema and includes required provenance metadata
- **When** the agent submits it to `graph write`
- **Then** the payload passes validation and is forwarded to the persistence layer

### Scenario: Reject a payload with missing provenance
- **Given** a JSON payload that omits `agent_id` or `session_id`
- **When** the agent submits it to `graph write`
- **Then** the CLI rejects the payload with a structured validation error describing the missing fields

### Scenario: Reject an unsupported schema version
- **Given** a JSON payload that uses an unknown `schema_version`
- **When** the agent submits it to the CLI
- **Then** the CLI returns a structured error explaining that the payload contract version is unsupported
