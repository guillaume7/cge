---
id: TH4.E1.US1
title: "Add `graph lab init` and experiment asset scaffolding"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "`graph lab init` creates or refreshes the `.graph/lab/` directory structure with suite manifest, condition definitions, and evaluation scaffolding, and returns a machine-readable summary."
  - AC2: "If the repo-local graph workspace is missing, `graph lab init` initializes it without damaging an existing workspace."
  - AC3: "Re-running `graph lab init` on an already-initialized lab is idempotent: existing run data, evaluations, and reports are preserved."
depends-on: []
---
# TH4.E1.US1 — Add `graph lab init` and experiment asset scaffolding

**As a** maintainer setting up an experiment, **I want** a single lab bootstrap
command, **so that** I can create the local experiment directory structure and
default manifests without hand-building file layouts.

## Acceptance Criteria

- [ ] AC1: `graph lab init` creates or refreshes the `.graph/lab/` directory structure with suite manifest, condition definitions, and evaluation scaffolding, and returns a machine-readable summary.
- [ ] AC2: If the repo-local graph workspace is missing, `graph lab init` initializes it without damaging an existing workspace.
- [ ] AC3: Re-running `graph lab init` on an already-initialized lab is idempotent: existing run data, evaluations, and reports are preserved.

## BDD Scenarios

### Scenario: Bootstrap experiment lab into a repo with no lab assets
- **Given** the repository has a graph workspace but no `.graph/lab/` directory
- **When** a maintainer runs `graph lab init`
- **Then** the command creates the lab directory structure with default suite manifest, condition definitions, runs, evaluations, and reports directories, and returns a machine-readable bootstrap summary

### Scenario: Refresh lab bootstrap without destroying existing experiment data
- **Given** the repository already has a `.graph/lab/` directory with run records and reports
- **When** a maintainer runs `graph lab init` again
- **Then** the command refreshes scaffolding idempotently, preserves all existing run data, evaluations, and reports, and reports what was preserved or refreshed

### Scenario: Return a structured error when lab init cannot determine repo scope
- **Given** the command cannot determine a valid repository root or graph workspace
- **When** a maintainer runs `graph lab init`
- **Then** the command returns a structured operational error instead of partially creating lab state
