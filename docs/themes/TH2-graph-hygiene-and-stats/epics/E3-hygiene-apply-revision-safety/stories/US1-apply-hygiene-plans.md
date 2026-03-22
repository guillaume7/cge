---
id: TH2.E3.US1
title: "Apply explicit hygiene plans and return revision anchors"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "`graph hygiene --apply` executes only explicit requested hygiene actions from a supplied plan."
  - AC2: "Apply mode returns a machine-readable summary of what changed, including a resulting revision anchor."
  - AC3: "The command rejects mutation attempts when no explicit apply input is supplied."
depends-on: [TH2.E2.US3]
---
# TH2.E3.US1 — Apply explicit hygiene plans and return revision anchors

**As an** agent, **I want** to apply a selected hygiene plan explicitly, **so
that** graph cleanup is intentional, reviewable, and traceable.

## Acceptance Criteria

- [ ] AC1: `graph hygiene --apply` executes only explicit requested hygiene actions from a supplied plan.
- [ ] AC2: Apply mode returns a machine-readable summary of what changed, including a resulting revision anchor.
- [ ] AC3: The command rejects mutation attempts when no explicit apply input is supplied.

## BDD Scenarios

### Scenario: Apply an explicit hygiene plan successfully
- **Given** an agent has a valid hygiene plan with approved actions
- **When** the agent runs `graph hygiene --apply`
- **Then** only the selected actions are applied and the response includes a resulting revision anchor

### Scenario: Apply only the requested subset of actions
- **Given** a hygiene plan contains multiple candidate actions but only a subset is approved
- **When** the agent runs `graph hygiene --apply`
- **Then** only the approved subset is executed

### Scenario: Reject apply mode without explicit plan input
- **Given** an agent attempts to run hygiene apply without a valid explicit plan
- **When** the command executes
- **Then** it returns a structured validation error and performs no graph mutation
