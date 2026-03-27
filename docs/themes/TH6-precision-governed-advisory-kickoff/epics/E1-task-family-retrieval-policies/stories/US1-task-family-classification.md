---
id: TH6.E1.US1
title: "Classify workflow-start tasks into kickoff families"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "`graph workflow start` assigns each delegated task to a documented kickoff family such as write-producing, troubleshooting/diagnosis, verification/audit, or reporting/synthesis."
  - AC2: "The selected kickoff family is exposed in the machine-readable workflow-start output."
  - AC3: "If the task cannot be classified confidently, workflow start falls back to an explicit ambiguous-task family instead of silently pretending the task is a standard implementation task."
depends-on: []
---
# TH6.E1.US1 — Classify workflow-start tasks into kickoff families

**As a** maintainer operating delegated workflow, **I want** workflow start to
classify the incoming task into a kickoff family, **so that** later policy
choices are grounded in the type of work being delegated.

## Acceptance Criteria

- [ ] AC1: `graph workflow start` assigns each delegated task to a documented kickoff family such as write-producing, troubleshooting/diagnosis, verification/audit, or reporting/synthesis.
- [ ] AC2: The selected kickoff family is exposed in the machine-readable workflow-start output.
- [ ] AC3: If the task cannot be classified confidently, workflow start falls back to an explicit ambiguous-task family instead of silently pretending the task is a standard implementation task.

## BDD Scenarios

### Scenario: Classify an implementation request as write-producing
- **Given** a delegated task asking the agent to add or modify production code
- **When** `graph workflow start` analyzes the task
- **Then** the workflow-start output assigns the task to the write-producing family

### Scenario: Classify a reporting request as reporting/synthesis
- **Given** a delegated task asking the agent to summarize findings or produce a report
- **When** `graph workflow start` analyzes the task
- **Then** the workflow-start output assigns the task to the reporting/synthesis family

### Scenario: Mark an unclear request as ambiguous
- **Given** a delegated task whose wording does not clearly indicate a supported family
- **When** `graph workflow start` analyzes the task
- **Then** the output marks the task as ambiguous instead of over-committing to the wrong family

