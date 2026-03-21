---
id: TH1.E5.US4
title: "Fix or remove blocked status"
agents: [developer, reviewer]
skills: [backlog-management]
acceptance-criteria:
  - AC1: "blocked status is properly implemented or removed from the schema"
  - AC2: "No dead-code status values exist in the status table"
  - AC3: "All references across skills and instructions are consistent"
depends-on: [TH1.E1.US1]
---

# TH1.E5.US4 — Fix or Remove Blocked Status

**As a** methodology user, **I want** `blocked` to either work or not exist, **so that** the schema has no dead-code status values.

## Acceptance Criteria

- [ ] AC1: `blocked` is properly implemented or removed from the schema
- [ ] AC2: No dead-code status values exist in the status table
- [ ] AC3: All references across skills and instructions are consistent

## BDD Scenarios

### Scenario: (Option A) Blocked implemented
- **Given** a story whose dependencies are not yet `done`
- **When** the orchestrator evaluates it
- **Then** it marks it `blocked` and auto-transitions to `todo` when dependencies resolve

### Scenario: (Option B) Blocked removed
- **Given** the status values table in `backlog-management` skill
- **When** I read the valid statuses
- **Then** `blocked` is not listed

### Scenario: Consistency
- **Given** a search for "blocked" across all files
- **When** I review the results
- **Then** references are either consistently present (if implemented) or absent (if removed)
