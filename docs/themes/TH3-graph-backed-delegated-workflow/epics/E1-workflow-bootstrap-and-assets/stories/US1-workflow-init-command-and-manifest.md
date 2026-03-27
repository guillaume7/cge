---
id: TH3.E1.US1
title: "Add `graph workflow init` and a workflow asset manifest"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "`graph workflow init` creates or refreshes repo-local workflow state and returns a machine-readable summary of installed, preserved, skipped, and seeded work."
  - AC2: "If the repo-local graph workspace is missing, `graph workflow init` initializes it without damaging an existing workspace."
  - AC3: "Workflow bootstrap state is tracked in a local manifest that records installed assets and preserved overrides."
depends-on: []
---
# TH3.E1.US1 — Add `graph workflow init` and a workflow asset manifest

**As an** agent, **I want** a single workflow bootstrap command, **so that** I can
make delegated graph-backed workflow available in this repo without hand-building
repo state every time.

## Acceptance Criteria

- [ ] AC1: `graph workflow init` creates or refreshes repo-local workflow state and returns a machine-readable summary of installed, preserved, skipped, and seeded work.
- [ ] AC2: If the repo-local graph workspace is missing, `graph workflow init` initializes it without damaging an existing workspace.
- [ ] AC3: Workflow bootstrap state is tracked in a local manifest that records installed assets and preserved overrides.

## BDD Scenarios

### Scenario: Bootstrap delegated workflow into a repo with no workflow state yet
- **Given** the repository has no workflow manifest and no delegated-workflow assets installed
- **When** an agent runs `graph workflow init`
- **Then** the command initializes repo-local workflow state and returns a machine-readable bootstrap summary

### Scenario: Refresh workflow bootstrap without clobbering an existing setup
- **Given** the repository already has a workflow manifest and installed workflow assets
- **When** an agent runs `graph workflow init` again
- **Then** the command refreshes the bootstrap state idempotently and reports preserved or skipped items explicitly

### Scenario: Return a structured error when workflow bootstrap cannot determine repo scope
- **Given** the command cannot determine a valid repository root
- **When** an agent runs `graph workflow init`
- **Then** the command returns a structured operational error instead of partially installing workflow state
