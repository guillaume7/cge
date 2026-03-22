---
id: TH1.E2.US3
title: "Support graph updates and revision anchors"
type: standard
priority: medium
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "The persistence layer can update or supersede existing entities so agents can keep the graph tidy and current."
  - AC2: "Each successful write records a comparable graph revision anchor for later use by `graph diff`."
  - AC3: "Revision metadata is stored without requiring immutable history semantics in MVP."
depends-on: [TH1.E2.US2]
---
# TH1.E2.US3 — Support graph updates and revision anchors

**As an** agent, **I want** to update stale graph knowledge while recording comparable revision anchors, **so that** the graph stays useful without giving up diff support.

## Acceptance Criteria

- [ ] AC1: The persistence layer can update or supersede existing entities so agents can keep the graph tidy and current.
- [ ] AC2: Each successful write records a comparable graph revision anchor for later use by `graph diff`.
- [ ] AC3: Revision metadata is stored without requiring immutable history semantics in MVP.

## BDD Scenarios

### Scenario: Update an existing entity to reflect fresher knowledge
- **Given** the graph contains an entity whose summary or properties need refinement
- **When** the agent submits an updated payload for that entity
- **Then** the graph stores the new state and records a new revision anchor

### Scenario: Mark one entity as superseding another
- **Given** a payload that indicates a newer entity supersedes an outdated one
- **When** the agent runs `graph write`
- **Then** the graph stores the superseding relationship and the associated revision anchor

### Scenario: Fail a revision write when anchor metadata cannot be recorded
- **Given** the persistence layer cannot create the required revision metadata for a write
- **When** the agent runs `graph write`
- **Then** the CLI returns a structured error instead of silently persisting an untracked change
