---
id: TH7.E3.US2
title: "Run a targeted verification confirmation batch"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "The targeted rerun covers the key verification regressions plus at least one retained positive comparison task."
  - AC2: "The batch uses the same model, topology, and paired-condition structure as the earlier confirmation run so results remain comparable."
  - AC3: "The batch produces a machine-readable summary and debrief for the rerun gate decision."
depends-on: [TH7.E3.US1]
---
# TH7.E3.US2 — Run a targeted verification confirmation batch

**As a** maintainer deciding whether calibration worked, **I want** a targeted
rerun focused on the verification regressions, **so that** the next decision is
grounded in a cheaper, sharper experiment than a full campaign replay.

## Acceptance Criteria

- [ ] AC1: The targeted rerun covers the key verification regressions plus at least one retained positive comparison task.
- [ ] AC2: The batch uses the same model, topology, and paired-condition structure as the earlier confirmation run so results remain comparable.
- [ ] AC3: The batch produces a machine-readable summary and debrief for the rerun gate decision.

## BDD Scenarios

### Scenario: Build a targeted rerun matrix for verification regressions
- **Given** the earlier confirmation batch identified the dominant verification regressions
- **When** the targeted rerun matrix is built
- **Then** the matrix includes those regressions plus a retained positive comparison task for control

### Scenario: Preserve comparability with the earlier confirmation batch
- **Given** the targeted rerun is executed
- **When** the batch configuration is recorded
- **Then** the model, topology, and paired-condition structure match the earlier confirmation format

### Scenario: Produce a machine-readable rerun debrief
- **Given** the targeted rerun completes
- **When** the batch artifacts are written
- **Then** the run outputs include a machine-readable summary and debrief for the gate decision
