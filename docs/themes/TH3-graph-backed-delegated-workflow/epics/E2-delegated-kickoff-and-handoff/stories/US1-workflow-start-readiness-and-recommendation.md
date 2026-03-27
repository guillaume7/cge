---
id: TH3.E2.US1
title: "Add `graph workflow start` readiness checks and workflow recommendations"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "`graph workflow start` inspects graph availability, current revision state, and health indicators relevant to delegated-task kickoff."
  - AC2: "The command returns a machine-readable recommendation such as proceed, bootstrap, inspect hygiene, or gather more context."
  - AC3: "If readiness inspection fails, the command returns a structured operational error instead of a partial kickoff result."
depends-on: [TH3.E1.US2, TH3.E1.US3]
---
# TH3.E2.US1 — Add `graph workflow start` readiness checks and workflow recommendations

**As an** agent delegating a non-trivial subtask, **I want** workflow start to inspect
whether the graph is ready and what action to take next, **so that** I do not waste
tokens reconstructing orientation blindly.

## Acceptance Criteria

- [ ] AC1: `graph workflow start` inspects graph availability, current revision state, and health indicators relevant to delegated-task kickoff.
- [ ] AC2: The command returns a machine-readable recommendation such as proceed, bootstrap, inspect hygiene, or gather more context.
- [ ] AC3: If readiness inspection fails, the command returns a structured operational error instead of a partial kickoff result.

## BDD Scenarios

### Scenario: Recommend proceed for a healthy repo-local graph
- **Given** the repository has a usable graph, a readable current revision, and acceptable health indicators
- **When** an agent runs `graph workflow start --task "implement delegated workflow start"`
- **Then** the command returns a machine-readable readiness result that recommends proceeding

### Scenario: Recommend bootstrap or hygiene before delegated work
- **Given** the repository graph is missing or clearly unhealthy for delegated use
- **When** an agent runs `graph workflow start --task "implement delegated workflow start"`
- **Then** the command returns a machine-readable recommendation to bootstrap or inspect hygiene before continuing

### Scenario: Return a structured error when readiness inspection cannot load graph state
- **Given** the command cannot inspect graph state or health indicators
- **When** an agent runs `graph workflow start --task "implement delegated workflow start"`
- **Then** the command returns a structured operational error instead of an incomplete readiness result
