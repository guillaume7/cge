---
id: TH1.E4.US1
title: "Add user validation checkpoints"
agents: [developer, reviewer]
skills: [the-copilot-build-method]
acceptance-criteria:
  - AC1: "Orchestrator pauses at theme completion and presents a demo summary to the user"
  - AC2: "User can accept, reject, or amend vision for the next VP"
  - AC3: "Vision is frozen per-theme, not globally"
  - AC4: "the-copilot-build-method skill documents the checkpoint ceremony"
depends-on: [TH1.E2.US2]
---

# TH1.E4.US1 — Add User Validation Checkpoints

**As a** product builder, **I want** the orchestrator to pause at theme boundaries for user validation, **so that** I can course-correct the vision based on what was actually delivered.

## Acceptance Criteria

- [ ] AC1: Orchestrator pauses at theme completion and presents a demo summary to the user
- [ ] AC2: User can accept, reject, or amend vision for the next VP
- [ ] AC3: Vision is frozen per-theme, not globally
- [ ] AC4: `the-copilot-build-method` skill documents the checkpoint ceremony

## BDD Scenarios

### Scenario: Theme completion triggers user checkpoint
- **Given** all epics in a theme are `done`
- **When** the orchestrator runs the theme completion ceremony
- **Then** it pauses and presents a summary to the user before proceeding to the next theme

### Scenario: User accepts and proceeds
- **Given** the orchestrator presents a theme completion summary
- **When** the user accepts the delivered theme
- **Then** the orchestrator proceeds to the next eligible theme

### Scenario: User amends vision for next theme
- **Given** the orchestrator presents a theme completion summary
- **When** the user wants to adjust the next VP
- **Then** the orchestrator waits for the user to update vision docs before planning the next theme

### Scenario: Vision frozen per-theme only
- **Given** the methodology documentation
- **When** I read the vision freeze policy
- **Then** it says vision is frozen for the theme currently in execution, not for all future themes
