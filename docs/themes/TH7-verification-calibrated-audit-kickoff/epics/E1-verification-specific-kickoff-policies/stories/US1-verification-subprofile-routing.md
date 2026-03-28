---
id: TH7.E1.US1
title: "Route verification tasks into narrower sub-profiles"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "`graph workflow start` distinguishes at least stats-oriented audits, workflow-verification tasks, and general evidence/provenance audits within the broader verification family."
  - AC2: "The selected verification sub-profile is exposed in machine-readable kickoff output."
  - AC3: "When verification wording is too ambiguous for a narrow profile, workflow start falls back to a general verification profile instead of pretending the task is implementation or diagnosis work."
depends-on: []
---
# TH7.E1.US1 — Route verification tasks into narrower sub-profiles

**As a** maintainer debugging verification regressions, **I want** audit tasks
to route into narrower verification profiles, **so that** kickoff policy can
match the actual type of audit being performed.

## Acceptance Criteria

- [ ] AC1: `graph workflow start` distinguishes at least stats-oriented audits, workflow-verification tasks, and general evidence/provenance audits within the broader verification family.
- [ ] AC2: The selected verification sub-profile is exposed in machine-readable kickoff output.
- [ ] AC3: When verification wording is too ambiguous for a narrow profile, workflow start falls back to a general verification profile instead of pretending the task is implementation or diagnosis work.

## BDD Scenarios

### Scenario: Route a stats audit into the stats verification profile
- **Given** a delegated task asking the agent to verify graph stats counts or health indicators
- **When** `graph workflow start` analyzes the task
- **Then** the output assigns the task to the stats-audit verification sub-profile

### Scenario: Route a workflow verification task into the workflow verification profile
- **Given** a delegated task asking the agent to verify kickoff, handoff, or workflow embedding behavior
- **When** `graph workflow start` analyzes the task
- **Then** the output assigns the task to the workflow-verification sub-profile

### Scenario: Fall back to general verification when the audit is broad
- **Given** a delegated task asking for a general evidence audit without a clear stats or workflow target
- **When** `graph workflow start` analyzes the task
- **Then** the output assigns the task to a general verification profile instead of misclassifying it as another family
