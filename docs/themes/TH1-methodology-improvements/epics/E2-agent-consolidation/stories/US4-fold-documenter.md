---
id: TH1.E2.US4
title: "Fold documenter into orchestrator"
agents: [developer, reviewer]
skills: [the-copilot-build-method]
acceptance-criteria:
  - AC1: "documenter.agent.md is moved to .github/agents/archive/"
  - AC2: "Orchestrator generates changelog entries at epic completion"
  - AC3: "Orchestrator generates release notes at theme completion"
  - AC4: "Changelog and release notes templates are embedded in orchestrator instructions"
depends-on: [TH1.E2.US1]
---

# TH1.E2.US4 — Fold Documenter into Orchestrator

**As a** methodology user, **I want** changelog and release note generation handled by the orchestrator directly, **so that** we eliminate a low-value agent delegation.

## Acceptance Criteria

- [ ] AC1: `documenter.agent.md` is moved to `.github/agents/archive/`
- [ ] AC2: Orchestrator generates changelog entries at epic completion
- [ ] AC3: Orchestrator generates release notes at theme completion
- [ ] AC4: Changelog and release notes templates are embedded in orchestrator instructions

## BDD Scenarios

### Scenario: Documenter archived
- **Given** the `.github/agents/archive/` directory
- **When** I list archived agent files
- **Then** `documenter.agent.md` is present

### Scenario: Orchestrator produces epic changelog
- **Given** all stories in an epic are `done`
- **When** the orchestrator runs the epic completion ceremony
- **Then** it generates a changelog entry directly (no delegation to `@documenter`)

### Scenario: Orchestrator produces theme release notes
- **Given** all epics in a theme are `done`
- **When** the orchestrator runs the theme completion ceremony
- **Then** it generates release notes directly (no delegation to `@documenter`)

### Scenario: Templates available in orchestrator
- **Given** the orchestrator agent instructions
- **When** I search for changelog/release notes sections
- **Then** I find templates for both epic changelog entries and theme release notes
