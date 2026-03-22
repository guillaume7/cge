---
id: TH2.E1.US1
title: "Add a graph stats command and snapshot counts"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "`graph stats` returns a stable machine-readable success envelope with the current node count and relationship count for the repo-local graph."
  - AC2: "`graph stats` reads the current repo graph snapshot without mutating graph content or revision state."
  - AC3: "If the repo graph is missing or cannot be opened, `graph stats` returns a structured operational error consistent with the existing command contract."
depends-on: []
---
# TH2.E1.US1 — Add a graph stats command and snapshot counts

**As an** agent, **I want** a `graph stats` command that returns the graph's raw
size, **so that** I can quickly assess whether the repo-local graph is present
and how large it is before relying on it.

## Acceptance Criteria

- [ ] AC1: `graph stats` returns a stable machine-readable success envelope with the current node count and relationship count for the repo-local graph.
- [ ] AC2: `graph stats` reads the current repo graph snapshot without mutating graph content or revision state.
- [ ] AC3: If the repo graph is missing or cannot be opened, `graph stats` returns a structured operational error consistent with the existing command contract.

## BDD Scenarios

### Scenario: Return snapshot counts for an initialized graph
- **Given** an initialized repo-local graph workspace with persisted nodes and relationships
- **When** an agent runs `graph stats`
- **Then** the response includes the current node count and relationship count in a structured success envelope

### Scenario: Return zero counts for an initialized but empty graph
- **Given** an initialized repo-local graph workspace with no persisted graph content
- **When** an agent runs `graph stats`
- **Then** the response returns zero counts without failing

### Scenario: Return a structured operational error for a missing workspace
- **Given** the current repository has no initialized graph workspace
- **When** an agent runs `graph stats`
- **Then** the command returns a machine-readable operational error rather than partial output
