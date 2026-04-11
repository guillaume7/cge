---
id: TH8.E2.US1
title: "Implement the five normalized decision outcomes"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "The Decision Engine defines exactly five outcome types: continue, minimal, abstain, backtrack, and write."
  - AC2: "Each outcome is represented as a Go type with a machine-readable string label."
  - AC3: "The Decision Engine accepts an evaluation result and returns exactly one selected outcome."
  - AC4: "The outcome type determines what context (if any) is delivered to the consuming agent."
depends-on:
  - TH8.E1.US3
---
# TH8.E2.US1 — Implement the five normalized decision outcomes

**As a** consuming agent, **I want** the Decision Engine to select exactly one
normalized outcome per evaluation pass, **so that** I receive a clear,
unambiguous signal about what action to take.

## Acceptance Criteria

- [ ] AC1: The Decision Engine defines exactly five outcome types: continue, minimal, abstain, backtrack, and write.
- [ ] AC2: Each outcome is represented as a Go type with a machine-readable string label.
- [ ] AC3: The Decision Engine accepts an evaluation result and returns exactly one selected outcome.
- [ ] AC4: The outcome type determines what context (if any) is delivered to the consuming agent.

## BDD Scenarios

### Scenario: Select continue when confidence is high
- **Given** an evaluation result with composite confidence above the injection threshold
- **When** the Decision Engine processes the result
- **Then** the selected outcome is `continue`
- **And** the full scored context bundle is included in the result

### Scenario: Select minimal when confidence is moderate
- **Given** an evaluation result with composite confidence between the minimal and injection thresholds
- **When** the Decision Engine processes the result
- **Then** the selected outcome is `minimal`
- **And** only the highest-scored candidates survive in the result

### Scenario: Select abstain when confidence is low
- **Given** an evaluation result with composite confidence below the minimal threshold
- **When** the Decision Engine processes the result
- **Then** the selected outcome is `abstain`
- **And** no context bundle is included in the result

### Scenario: Select backtrack when quality regresses
- **Given** an evaluation result where the current score is lower than the prior evaluation score
- **When** the Decision Engine processes the result
- **Then** the selected outcome is `backtrack`

### Scenario: Select write when output is strong enough to persist
- **Given** an output evaluation result with composite confidence above the write threshold
- **When** the Decision Engine processes the result for a memory-write decision
- **Then** the selected outcome is `write`

## Notes

- The five outcomes map exactly to ADR-019's normalized decision set.
- `backtrack` is advisory — CGE does not re-execute agent work (ADR-019 §5).
