---
id: TH7.E3.US3
title: "Gate the full rerun on verification-family neutrality"
type: standard
priority: medium
size: S
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "The rerun debrief states a clear gate rule for whether the full campaign rerun is justified."
  - AC2: "The gate rule requires verification-family regressions to move toward rough neutrality before spending the full rerun budget."
  - AC3: "The gate output records whether the decision is proceed, hold, or stop-and-recalibrate."
depends-on: [TH7.E3.US2]
---
# TH7.E3.US3 — Gate the full rerun on verification-family neutrality

**As a** maintainer controlling experiment budget, **I want** an explicit rerun
gate based on verification-family results, **so that** the full campaign is not
replayed while the main regression family is still visibly broken.

## Acceptance Criteria

- [ ] AC1: The rerun debrief states a clear gate rule for whether the full campaign rerun is justified.
- [ ] AC2: The gate rule requires verification-family regressions to move toward rough neutrality before spending the full rerun budget.
- [ ] AC3: The gate output records whether the decision is proceed, hold, or stop-and-recalibrate.

## BDD Scenarios

### Scenario: Approve the full rerun when verification is roughly neutral
- **Given** the targeted verification rerun shows verification-family means near neutral and no catastrophic regressions
- **When** the gate decision is computed
- **Then** the output records a proceed decision for the full rerun

### Scenario: Hold the full rerun when verification is still visibly negative
- **Given** the targeted verification rerun still shows strong graph-negative verification-family deltas
- **When** the gate decision is computed
- **Then** the output records hold or stop-and-recalibrate instead of proceeding

### Scenario: Preserve the gate decision machine-readably
- **Given** the rerun gate has been evaluated
- **When** the debrief is written
- **Then** the decision is preserved machine-readably as proceed, hold, or stop-and-recalibrate
