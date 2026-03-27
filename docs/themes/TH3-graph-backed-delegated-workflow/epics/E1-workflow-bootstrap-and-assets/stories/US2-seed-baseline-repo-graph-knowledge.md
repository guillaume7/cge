---
id: TH3.E1.US2
title: "Seed baseline repo graph knowledge during workflow init"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "`graph workflow init` can seed baseline graph knowledge from standard repo artifacts needed for delegated-task orientation."
  - AC2: "Missing optional repo artifacts are reported explicitly and skipped without failing the whole bootstrap flow."
  - AC3: "Seeded knowledge is persisted through the existing graph write path with provenance and revision compatibility."
depends-on: [TH3.E1.US1]
---
# TH3.E1.US2 — Seed baseline repo graph knowledge during workflow init

**As an** agent, **I want** workflow bootstrap to seed the graph from standard repo
artifacts, **so that** delegated subtasks inherit useful context without manual
one-off graph curation.

## Acceptance Criteria

- [ ] AC1: `graph workflow init` can seed baseline graph knowledge from standard repo artifacts needed for delegated-task orientation.
- [ ] AC2: Missing optional repo artifacts are reported explicitly and skipped without failing the whole bootstrap flow.
- [ ] AC3: Seeded knowledge is persisted through the existing graph write path with provenance and revision compatibility.

## BDD Scenarios

### Scenario: Seed baseline graph knowledge from a standard repo layout
- **Given** the repository contains standard artifacts such as `README.md`, `docs/architecture/`, and `docs/plan/backlog.yaml`
- **When** an agent runs `graph workflow init`
- **Then** baseline graph knowledge is written from those artifacts for later delegated-task retrieval

### Scenario: Skip missing optional seed sources while completing bootstrap
- **Given** one or more optional seed sources are absent from the repository
- **When** an agent runs `graph workflow init`
- **Then** the command reports the missing sources explicitly and completes bootstrap from the remaining valid inputs

### Scenario: Return a structured error when baseline seeding cannot persist graph writes
- **Given** workflow bootstrap can discover seed sources but the graph write path fails
- **When** an agent runs `graph workflow init`
- **Then** the command returns a structured error and does not report successful seeding
