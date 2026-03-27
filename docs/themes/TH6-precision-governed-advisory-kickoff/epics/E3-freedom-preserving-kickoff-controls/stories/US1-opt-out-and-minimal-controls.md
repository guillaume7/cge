---
id: TH6.E3.US1
title: "Add explicit no-kickoff and minimal-kickoff controls"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "Workflow start exposes explicit controls for no-kickoff and minimal-kickoff modes."
  - AC2: "The selected kickoff mode is reflected in machine-readable workflow-start output."
  - AC3: "Explicit kickoff-mode selection overrides the default family policy only within the allowed freedom constraints of the command."
depends-on: [TH6.E2.US3]
---
# TH6.E3.US1 — Add explicit no-kickoff and minimal-kickoff controls

**As a** maintainer or agent invoking workflow start, **I want** explicit
kickoff-mode controls, **so that** I can request no kickoff or a smaller kickoff
without fighting the default behavior.

## Acceptance Criteria

- [ ] AC1: Workflow start exposes explicit controls for no-kickoff and minimal-kickoff modes.
- [ ] AC2: The selected kickoff mode is reflected in machine-readable workflow-start output.
- [ ] AC3: Explicit kickoff-mode selection overrides the default family policy only within the allowed freedom constraints of the command.

## BDD Scenarios

### Scenario: Request no kickoff explicitly
- **Given** a caller invokes workflow start with an explicit no-kickoff mode
- **When** the command returns
- **Then** the workflow-start output records that no kickoff was requested and does not inject graph context

### Scenario: Request minimal kickoff explicitly
- **Given** a caller invokes workflow start with an explicit minimal-kickoff mode
- **When** the command returns
- **Then** the workflow-start output records the minimal mode and returns only the reduced kickoff form

### Scenario: Preserve allowed safety constraints
- **Given** a caller requests a more permissive kickoff mode for a family with strict abstention defaults
- **When** workflow start applies the command rules
- **Then** the command reports the selected mode and whether any family safety constraint limited it

