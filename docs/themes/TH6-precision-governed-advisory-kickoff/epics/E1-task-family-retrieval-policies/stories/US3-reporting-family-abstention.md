---
id: TH6.E1.US3
title: "Skip kickoff by default for reporting and synthesis tasks"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "When a delegated task is classified as reporting/synthesis, workflow start defaults to no kickoff instead of projecting normal graph context."
  - AC2: "The machine-readable workflow-start output explicitly reports that kickoff abstained because the selected family defaults to no kickoff."
  - AC3: "Abstaining from kickoff for reporting/synthesis tasks preserves the rest of the workflow-start contract so downstream tooling still works."
depends-on: [TH6.E1.US2]
---
# TH6.E1.US3 — Skip kickoff by default for reporting and synthesis tasks

**As a** maintainer protecting agents from contamination, **I want** reporting
and synthesis tasks to abstain from kickoff by default, **so that** CGE steps
back when evidence says context injection is more likely to hurt than help.

## Acceptance Criteria

- [ ] AC1: When a delegated task is classified as reporting/synthesis, workflow start defaults to no kickoff instead of projecting normal graph context.
- [ ] AC2: The machine-readable workflow-start output explicitly reports that kickoff abstained because the selected family defaults to no kickoff.
- [ ] AC3: Abstaining from kickoff for reporting/synthesis tasks preserves the rest of the workflow-start contract so downstream tooling still works.

## BDD Scenarios

### Scenario: Abstain on a report-writing request
- **Given** a delegated task asks the agent to summarize findings into a report
- **When** `graph workflow start` processes the task
- **Then** the kickoff result abstains from graph injection and records the reporting/synthesis family as the reason

### Scenario: Keep workflow output machine-readable after abstention
- **Given** workflow start abstains from kickoff for a reporting task
- **When** downstream tooling reads the workflow-start result
- **Then** the result remains machine-readable and indicates abstention instead of returning malformed or missing data

### Scenario: Avoid accidental context projection for a synthesis task
- **Given** the graph contains high-scoring entities for a synthesis task
- **When** the reporting/synthesis family policy is applied
- **Then** workflow start still abstains by default instead of projecting those entities automatically

