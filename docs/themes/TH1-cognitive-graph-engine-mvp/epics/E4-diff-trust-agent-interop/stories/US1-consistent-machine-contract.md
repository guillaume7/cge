---
id: TH1.E4.US1
title: "Return a consistent machine-readable command contract"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "`graph write`, `graph query`, `graph context`, `graph explain`, and `graph diff` return stable native JSON response envelopes suitable for downstream agents."
  - AC2: "Structured stdout output is consistent with file-based output for equivalent command results."
  - AC3: "Command responses include enough metadata for downstream agents to distinguish success, validation errors, and operational failures."
depends-on: [TH1.E3.US2]
---
# TH1.E4.US1 — Return a consistent machine-readable command contract

**As an** agent, **I want** every graph command to speak the same native machine contract, **so that** custom agents can reliably consume outputs without brittle translation logic.

## Acceptance Criteria

- [ ] AC1: `graph write`, `graph query`, `graph context`, `graph explain`, and `graph diff` return stable native JSON response envelopes suitable for downstream agents.
- [ ] AC2: Structured stdout output is consistent with file-based output for equivalent command results.
- [ ] AC3: Command responses include enough metadata for downstream agents to distinguish success, validation errors, and operational failures.

## BDD Scenarios

### Scenario: Return a structured success envelope
- **Given** an agent runs a successful graph command that produces structured output
- **When** the command writes its response
- **Then** the response uses the native JSON envelope with predictable success metadata

### Scenario: Return matching structured output to stdout and file
- **Given** a command supports both stdout output and file output for the same result
- **When** the agent executes both modes
- **Then** the serialized response shape is equivalent in both places

### Scenario: Return a structured error envelope
- **Given** a graph command fails validation or encounters an operational error
- **When** the command returns
- **Then** the response is still machine-readable and distinguishes the failure category clearly
