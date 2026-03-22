---
id: TH2.E3.US2
title: "Keep hygiene changes inspectable through revision diff"
type: standard
priority: medium
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "Applied hygiene changes produce revision anchors that can be compared with `graph diff`."
  - AC2: "Duplicate consolidation, orphan pruning, and contradiction resolution remain visible as meaningful graph changes in diff output."
  - AC3: "Successful hygiene apply summaries include enough revision metadata for agents to inspect before/after states."
depends-on: [TH2.E3.US1]
---
# TH2.E3.US2 — Keep hygiene changes inspectable through revision diff

**As an** agent, **I want** hygiene changes to remain diffable, **so that** I
can understand exactly how cleanup altered the graph and trust the result.

## Acceptance Criteria

- [ ] AC1: Applied hygiene changes produce revision anchors that can be compared with `graph diff`.
- [ ] AC2: Duplicate consolidation, orphan pruning, and contradiction resolution remain visible as meaningful graph changes in diff output.
- [ ] AC3: Successful hygiene apply summaries include enough revision metadata for agents to inspect before/after states.

## BDD Scenarios

### Scenario: Diff a before and after hygiene revision
- **Given** an agent has applied a hygiene plan successfully
- **When** the agent runs `graph diff` between the before and after revision anchors
- **Then** the diff reports the resulting cleanup changes meaningfully

### Scenario: Show no-op compatible metadata for a plan with no changes
- **Given** an agent applies a hygiene plan that results in no graph changes
- **When** the command returns
- **Then** the response still makes the no-op outcome explicit without inventing fake cleanup changes

### Scenario: Return a structured error when a revision cannot be diffed
- **Given** a requested before or after revision anchor cannot be resolved for diff inspection
- **When** the agent requests the hygiene-related diff
- **Then** the command returns a structured operational error
