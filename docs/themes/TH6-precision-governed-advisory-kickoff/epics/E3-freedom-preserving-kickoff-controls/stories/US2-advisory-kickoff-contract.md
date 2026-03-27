---
id: TH6.E3.US2
title: "Expose advisory kickoff state without breaking workflow compatibility"
type: standard
priority: medium
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "Workflow-start output exposes whether kickoff was injected, minimal, or abstained in a stable machine-readable field."
  - AC2: "Existing workflow consumers can continue to parse the workflow-start result without being forced onto a breaking contract change."
  - AC3: "Advisory kickoff state is available regardless of whether the final result came from default policy or explicit caller choice."
depends-on: [TH6.E3.US1]
---
# TH6.E3.US2 — Expose advisory kickoff state without breaking workflow compatibility

**As a** downstream workflow consumer, **I want** the advisory kickoff state to
be explicit, **so that** I can understand what happened without losing
compatibility with the existing workflow contract.

## Acceptance Criteria

- [ ] AC1: Workflow-start output exposes whether kickoff was injected, minimal, or abstained in a stable machine-readable field.
- [ ] AC2: Existing workflow consumers can continue to parse the workflow-start result without being forced onto a breaking contract change.
- [ ] AC3: Advisory kickoff state is available regardless of whether the final result came from default policy or explicit caller choice.

## BDD Scenarios

### Scenario: Expose advisory state on a normal injected kickoff
- **Given** workflow start selects a normal injected kickoff
- **When** the command returns its machine-readable output
- **Then** the output includes an explicit advisory state describing the injected result

### Scenario: Expose advisory state on an abstained kickoff
- **Given** workflow start abstains from kickoff
- **When** the command returns its machine-readable output
- **Then** the output includes an explicit advisory state describing the abstained result

### Scenario: Preserve compatibility for existing consumers
- **Given** an existing consumer reads the workflow-start output using the prior contract
- **When** advisory kickoff state is added
- **Then** the prior fields remain present and parseable while the new advisory state appears alongside them

