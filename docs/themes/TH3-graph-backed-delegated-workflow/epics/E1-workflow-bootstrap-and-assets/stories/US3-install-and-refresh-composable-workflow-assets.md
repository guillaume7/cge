---
id: TH3.E1.US3
title: "Install and refresh composable workflow assets while preserving repo overrides"
type: standard
priority: medium
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "Workflow bootstrap installs the prompt, skill, instruction, and wrapper or hook assets required for graph-backed delegated workflow in this repo."
  - AC2: "Refreshing workflow assets preserves explicit repo overrides instead of overwriting them silently."
  - AC3: "Installed workflow assets remain inspectable and the bootstrap result reports what was installed, refreshed, or preserved."
depends-on: [TH3.E1.US1]
---
# TH3.E1.US3 — Install and refresh composable workflow assets while preserving repo overrides

**As a** repo maintainer, **I want** delegated-workflow assets to install and refresh
cleanly, **so that** this repo can dogfood graph-backed delegation without losing
local conventions.

## Acceptance Criteria

- [ ] AC1: Workflow bootstrap installs the prompt, skill, instruction, and wrapper or hook assets required for graph-backed delegated workflow in this repo.
- [ ] AC2: Refreshing workflow assets preserves explicit repo overrides instead of overwriting them silently.
- [ ] AC3: Installed workflow assets remain inspectable and the bootstrap result reports what was installed, refreshed, or preserved.

## BDD Scenarios

### Scenario: Install the minimum composable assets for delegated workflow
- **Given** the repository does not yet include the delegated-workflow snippets or helpers
- **When** an agent runs `graph workflow init`
- **Then** the required assets are installed in predictable locations and reported in the bootstrap summary

### Scenario: Preserve explicit repo overrides during a workflow asset refresh
- **Given** the repository contains workflow assets with declared local overrides
- **When** an agent runs `graph workflow init` to refresh assets
- **Then** the refresh preserves those overrides and records them in the manifest and summary output

### Scenario: Return a structured error when an asset refresh cannot update a required workflow file
- **Given** a required workflow asset cannot be installed or refreshed
- **When** an agent runs `graph workflow init`
- **Then** the command returns a structured error instead of silently leaving the repo in an ambiguous workflow state
