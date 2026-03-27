---
id: TH4.E2.US3
title: "Support condition randomization and seed-based reproducibility"
type: standard
priority: medium
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "When a batch of runs is requested, condition ordering is randomized by default to prevent order effects from biasing results."
  - AC2: "Randomization is deterministic given the same seed, so the same batch can be reproduced exactly."
  - AC3: "The batch plan (task × condition assignment with ordering) is persisted as a machine-readable artifact before execution begins."
depends-on: [TH4.E2.US1]
---
# TH4.E2.US3 — Support condition randomization and seed-based reproducibility

**As a** scientifically rigorous experimenter, **I want** condition ordering to
be randomized with a deterministic seed, **so that** order effects do not bias
results and the experiment can be exactly reproduced.

## Acceptance Criteria

- [ ] AC1: When a batch of runs is requested, condition ordering is randomized by default to prevent order effects from biasing results.
- [ ] AC2: Randomization is deterministic given the same seed, so the same batch can be reproduced exactly.
- [ ] AC3: The batch plan (task × condition assignment with ordering) is persisted as a machine-readable artifact before execution begins.

## BDD Scenarios

### Scenario: Randomize condition ordering in a batch of runs
- **Given** a batch of 4 runs is requested across 2 tasks and 2 conditions
- **When** the lab orchestrator plans the batch with `--seed 42`
- **Then** the run ordering is shuffled and differs from the natural task × condition product order

### Scenario: Reproduce the same batch ordering with the same seed
- **Given** two separate batch planning requests use the same seed, tasks, and conditions
- **When** both batches are planned
- **Then** the resulting run orderings are identical

### Scenario: Persist the batch plan before execution begins
- **Given** a batch of runs has been planned with randomized ordering
- **When** execution is about to begin
- **Then** the batch plan is written as a machine-readable artifact that records the task × condition × ordering assignment

### Scenario: Allow explicit sequential ordering when randomization is disabled
- **Given** a batch of runs is requested with `--no-randomize`
- **When** the lab orchestrator plans the batch
- **Then** the runs follow the natural task × condition product order without shuffling
