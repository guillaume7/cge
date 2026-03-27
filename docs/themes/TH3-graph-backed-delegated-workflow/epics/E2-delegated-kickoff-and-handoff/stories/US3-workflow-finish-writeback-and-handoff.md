---
id: TH3.E2.US3
title: "Add `graph workflow finish` writeback and handoff envelopes"
type: standard
priority: high
size: L
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "`graph workflow finish` accepts a structured delegated-task outcome payload and persists durable graph memory through the existing revision-aware write path."
  - AC2: "The command returns a machine-readable handoff envelope with before and after revision anchors, a write summary, and a next-agent brief or explicit no-op result."
  - AC3: "Invalid or unsafe finish inputs are rejected with a structured error and do not mutate the graph."
depends-on: [TH3.E2.US2]
---
# TH3.E2.US3 — Add `graph workflow finish` writeback and handoff envelopes

**As an** agent closing a delegated subtask, **I want** workflow finish to persist a
structured outcome and return a next-agent handoff, **so that** later work can
resume from durable graph memory instead of conversation residue.

## Acceptance Criteria

- [ ] AC1: `graph workflow finish` accepts a structured delegated-task outcome payload and persists durable graph memory through the existing revision-aware write path.
- [ ] AC2: The command returns a machine-readable handoff envelope with before and after revision anchors, a write summary, and a next-agent brief or explicit no-op result.
- [ ] AC3: Invalid or unsafe finish inputs are rejected with a structured error and do not mutate the graph.

## BDD Scenarios

### Scenario: Persist a delegated-task outcome and return a handoff envelope
- **Given** an agent has a valid structured delegated-task outcome payload
- **When** the agent runs `graph workflow finish --file task-outcome.json`
- **Then** the command persists the outcome through the existing write path and returns revision anchors plus a machine-readable handoff brief

### Scenario: Return an explicit no-op handoff when no graph changes are needed
- **Given** an agent provides a valid finish payload that contains no durable graph updates
- **When** the agent runs `graph workflow finish --file task-outcome.json`
- **Then** the command returns a machine-readable no-op result instead of pretending a graph mutation occurred

### Scenario: Reject an invalid delegated-task outcome without mutating the graph
- **Given** an agent provides an invalid or unsafe workflow finish payload
- **When** the agent runs `graph workflow finish --file task-outcome.json`
- **Then** the command returns a structured validation error and leaves the graph unchanged
