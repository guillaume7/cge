---
id: TH3.E3.US3
title: "Wire this repo's delegated-task workflow through graph-backed snippets and hooks"
type: standard
priority: high
size: L
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "This repo's prompts, skills, or instructions steer most non-trivial delegated subtasks through `graph workflow start` and `graph workflow finish`."
  - AC2: "Any workflow wrapper or hook used for repo dogfooding remains explicit, inspectable, and easy to opt out of."
  - AC3: "Repo-local verification demonstrates that a delegated-task kickoff and handoff can be completed end to end without inventing ad hoc prompt conventions."
depends-on: [TH3.E1.US3, TH3.E2.US3, TH3.E3.US2]
---
# TH3.E3.US3 — Wire this repo's delegated-task workflow through graph-backed snippets and hooks

**As a** maintainer dogfooding CGE, **I want** this repo's own workflow metadata to
route delegated tasks through graph-backed kickoff and handoff, **so that** VP3 is
proven in real use before it is packaged for other repos.

## Acceptance Criteria

- [ ] AC1: This repo's prompts, skills, or instructions steer most non-trivial delegated subtasks through `graph workflow start` and `graph workflow finish`.
- [ ] AC2: Any workflow wrapper or hook used for repo dogfooding remains explicit, inspectable, and easy to opt out of.
- [ ] AC3: Repo-local verification demonstrates that a delegated-task kickoff and handoff can be completed end to end without inventing ad hoc prompt conventions.

## BDD Scenarios

### Scenario: Default a non-trivial delegated task to graph-backed kickoff and handoff
- **Given** this repo has the VP3 workflow assets installed
- **When** a parent agent prepares a non-trivial delegated subtask
- **Then** the repo guidance steers that task through `graph workflow start` and `graph workflow finish` by default

### Scenario: Allow an explicit opt-out from the delegated workflow path
- **Given** a delegated task explicitly opts out of graph-backed workflow
- **When** the repo workflow snippets or wrappers are evaluated
- **Then** the opt-out is honored without hidden graph behavior

### Scenario: Verify an end-to-end delegated-task flow in this repo
- **Given** this repo has graph-backed workflow assets and a delegated subtask to perform
- **When** an agent follows the installed repo workflow guidance end to end
- **Then** the kickoff brief, task execution, and handoff path complete without inventing ad hoc repo conventions
