---
id: TH2.E3.US3
title: "Reject stale or unsafe hygiene plans without mutating the graph"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "`graph hygiene --apply` rejects stale plans whose target graph snapshot no longer matches the current graph state."
  - AC2: "`graph hygiene --apply` rejects unsupported actions, missing targets, or otherwise unsafe plan instructions with a structured validation or operational error."
  - AC3: "Rejected hygiene plans leave graph content and revision state unchanged."
depends-on: [TH2.E3.US1]
---
# TH2.E3.US3 — Reject stale or unsafe hygiene plans without mutating the graph

**As an** agent, **I want** stale or unsafe hygiene plans to fail safely, **so
that** graph cleanup cannot accidentally corrupt or surprise the shared memory.

## Acceptance Criteria

- [ ] AC1: `graph hygiene --apply` rejects stale plans whose target graph snapshot no longer matches the current graph state.
- [ ] AC2: `graph hygiene --apply` rejects unsupported actions, missing targets, or otherwise unsafe plan instructions with a structured validation or operational error.
- [ ] AC3: Rejected hygiene plans leave graph content and revision state unchanged.

## BDD Scenarios

### Scenario: Reject a stale hygiene plan
- **Given** an agent holds a hygiene plan built against an earlier graph snapshot
- **When** the graph changes before the agent runs `graph hygiene --apply`
- **Then** the command rejects the stale plan and leaves the graph unchanged

### Scenario: Reject an unsafe or invalid hygiene action
- **Given** a hygiene plan includes an unsupported action type or a missing target entity
- **When** the agent runs `graph hygiene --apply`
- **Then** the command returns a structured error and does not mutate the graph

### Scenario: Preserve graph state after a rejected apply attempt
- **Given** a hygiene apply request is rejected
- **When** the agent inspects the graph afterward
- **Then** no partial cleanup changes or new revision mutations have been committed
