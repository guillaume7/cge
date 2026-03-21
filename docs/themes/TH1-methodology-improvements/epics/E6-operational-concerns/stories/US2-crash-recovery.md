---
id: TH1.E6.US2
title: "Add crash recovery protocol"
agents: [developer, reviewer]
skills: [backlog-management, the-copilot-build-method]
acceptance-criteria:
  - AC1: "Orchestrator checks for in-progress stories on startup"
  - AC2: "Recovery options: continue, reset to todo, or escalate to user"
  - AC3: "backlog-management skill documents the recovery protocol"
depends-on: [TH1.E1.US2]
---

# TH1.E6.US2 — Add Crash Recovery Protocol

**As a** product builder, **I want** the orchestrator to recover gracefully from mid-session crashes, **so that** partial work is not lost or left in a broken state.

## Acceptance Criteria

- [ ] AC1: Orchestrator checks for `in-progress` stories on startup
- [ ] AC2: Recovery options: continue, reset to `todo`, or escalate to user
- [ ] AC3: `backlog-management` skill documents the recovery protocol

## BDD Scenarios

### Scenario: Orchestrator detects stale in-progress
- **Given** the orchestrator starts and the backlog has a story with status `in-progress`
- **When** the orchestrator begins its loop
- **Then** it detects the stale in-progress story and triggers recovery

### Scenario: Recovery assesses state
- **Given** a stale in-progress story
- **When** the orchestrator runs recovery
- **Then** it checks for partial code changes (new/modified files) and decides whether to continue or reset

### Scenario: User escalation
- **Given** the orchestrator can't determine if partial work is valid
- **When** recovery assessment is inconclusive
- **Then** it escalates to the user with a description of the current state
