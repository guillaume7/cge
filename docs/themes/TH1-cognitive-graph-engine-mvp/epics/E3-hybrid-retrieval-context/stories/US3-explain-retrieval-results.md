---
id: TH1.E3.US3
title: "Explain retrieval results and provenance"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "`graph explain` returns structured reasons for why nodes or relationships were selected during query or context retrieval."
  - AC2: "Explanation output includes provenance references and enough ranking detail to debug surprising or stale context results."
  - AC3: "Explanation can be requested for the same task phrasing used with `graph query` or `graph context`."
depends-on: [TH1.E3.US2]
---
# TH1.E3.US3 — Explain retrieval results and provenance

**As an** agent, **I want** `graph explain` to justify retrieval decisions, **so that** I can trust or debug the context I receive before building on it.

## Acceptance Criteria

- [ ] AC1: `graph explain` returns structured reasons for why nodes or relationships were selected during query or context retrieval.
- [ ] AC2: Explanation output includes provenance references and enough ranking detail to debug surprising or stale context results.
- [ ] AC3: Explanation can be requested for the same task phrasing used with `graph query` or `graph context`.

## BDD Scenarios

### Scenario: Explain why a result was included
- **Given** a task whose retrieval result includes several entities
- **When** the agent runs `graph explain` for that task
- **Then** the CLI reports the matching graph paths, text matches, and provenance references that led to inclusion

### Scenario: Explain why a stale entity ranked unexpectedly
- **Given** the graph contains both fresh and stale entities related to a task
- **When** the agent runs `graph explain`
- **Then** the output contains enough ranking detail for the agent to understand why the stale entity appeared

### Scenario: Reject an explanation request when no retrieval basis exists
- **Given** the agent submits an empty task to `graph explain`
- **When** the command is executed
- **Then** the CLI returns a structured validation error instead of a meaningless explanation
