---
id: TH2.E2.US1
title: "Detect orphan and duplicate-near-identical hygiene candidates"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "`graph hygiene` suggest mode identifies orphan-node candidates and duplicate-near-identical node groups from the current graph snapshot."
  - AC2: "Each orphan or duplicate candidate includes machine-readable reasons that explain why it was flagged."
  - AC3: "Suggest mode does not mutate graph content, revision state, or the current graph snapshot."
depends-on: [TH2.E1.US1]
---
# TH2.E2.US1 — Detect orphan and duplicate-near-identical hygiene candidates

**As an** agent, **I want** `graph hygiene` to find orphan and duplicate-like
graph content, **so that** I can clean obvious graph disorder before it harms
retrieval quality.

## Acceptance Criteria

- [ ] AC1: `graph hygiene` suggest mode identifies orphan-node candidates and duplicate-near-identical node groups from the current graph snapshot.
- [ ] AC2: Each orphan or duplicate candidate includes machine-readable reasons that explain why it was flagged.
- [ ] AC3: Suggest mode does not mutate graph content, revision state, or the current graph snapshot.

## BDD Scenarios

### Scenario: Return orphan and duplicate candidates for a noisy graph
- **Given** a repo-local graph contains orphan nodes and near-identical duplicate nodes
- **When** an agent runs `graph hygiene`
- **Then** the command returns those candidates with structured reasons and no graph mutation

### Scenario: Return an empty candidate set for a tidy graph
- **Given** a repo-local graph has no orphan nodes and no duplicate-near-identical groups
- **When** an agent runs `graph hygiene`
- **Then** the command returns an empty suggestion set rather than fabricated cleanup work

### Scenario: Return a structured error when hygiene analysis cannot inspect the graph
- **Given** the graph snapshot cannot be loaded for hygiene analysis
- **When** an agent runs `graph hygiene`
- **Then** the command returns a structured operational error instead of partial suggestions
