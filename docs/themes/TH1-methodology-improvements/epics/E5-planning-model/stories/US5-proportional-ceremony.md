---
id: TH1.E5.US5
title: "Proportional ceremony overhead"
agents: [developer, reviewer]
skills: [the-copilot-build-method]
acceptance-criteria:
  - AC1: "Small epics (≤3 stories) skip code quality pass and produce minimal changelog"
  - AC2: "Large epics (4+ stories) get full ceremony"
  - AC3: "Orchestrator instructions document the threshold"
depends-on: [TH1.E2.US3]
---

# TH1.E5.US5 — Proportional Ceremony Overhead

**As a** product builder, **I want** epic ceremony to scale with epic size, **so that** small epics don't pay the same overhead as large ones.

## Acceptance Criteria

- [ ] AC1: Small epics (≤3 stories) skip code quality pass and produce minimal changelog
- [ ] AC2: Large epics (4+ stories) get full ceremony
- [ ] AC3: Orchestrator instructions document the threshold

## BDD Scenarios

### Scenario: Small epic lightweight ceremony
- **Given** an epic with 2 stories, all `done`
- **When** the orchestrator runs epic completion
- **Then** it skips code quality review and produces a brief changelog

### Scenario: Large epic full ceremony
- **Given** an epic with 6 stories, all `done`
- **When** the orchestrator runs epic completion
- **Then** it runs code quality review + full changelog

### Scenario: Threshold documented
- **Given** the orchestrator's epic completion section
- **When** I read the ceremony rules
- **Then** it states ≤3 stories = lightweight, 4+ = full
