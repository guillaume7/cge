---
id: TH8.E2.US2
title: "Select outcomes via configurable confidence thresholds"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "The Decision Engine uses four configurable thresholds: injection (continue), minimal, write, and regression-delta (backtrack)."
  - AC2: "Thresholds are loaded from the local workspace configuration and can be overridden per invocation."
  - AC3: "VP8 ships with conservative defaults that prefer minimal and abstain over aggressive injection."
  - AC4: "When thresholds overlap or are misconfigured, the Decision Engine returns a clear error instead of selecting an arbitrary outcome."
depends-on:
  - TH8.E2.US1
---
# TH8.E2.US2 — Select outcomes via configurable confidence thresholds

**As a** maintainer tuning the evaluator loop, **I want** confidence thresholds
to be configurable with conservative defaults, **so that** the system prefers
honest uncertainty over aggressive context injection.

## Acceptance Criteria

- [ ] AC1: The Decision Engine uses four configurable thresholds: injection (continue), minimal, write, and regression-delta (backtrack).
- [ ] AC2: Thresholds are loaded from the local workspace configuration and can be overridden per invocation.
- [ ] AC3: VP8 ships with conservative defaults that prefer minimal and abstain over aggressive injection.
- [ ] AC4: When thresholds overlap or are misconfigured, the Decision Engine returns a clear error instead of selecting an arbitrary outcome.

## BDD Scenarios

### Scenario: Apply default conservative thresholds
- **Given** no custom threshold configuration is present in the workspace
- **When** the Decision Engine selects an outcome for a composite confidence of 0.55
- **Then** the selected outcome is `minimal` or `abstain` (not `continue`) per conservative defaults

### Scenario: Override thresholds for a single invocation
- **Given** workspace thresholds set injection at 0.8
- **And** an invocation-level override sets injection at 0.5
- **When** the Decision Engine selects an outcome for a composite confidence of 0.6
- **Then** the invocation override applies and the outcome is `continue`

### Scenario: Load thresholds from workspace configuration
- **Given** a workspace with a threshold config file setting injection=0.75, minimal=0.45, write=0.80, regression-delta=0.10
- **When** the Decision Engine initializes
- **Then** all four thresholds match the configured values

### Scenario: Reject misconfigured thresholds
- **Given** a threshold configuration where the minimal threshold is higher than the injection threshold
- **When** the Decision Engine attempts to load the configuration
- **Then** the engine returns a validation error explaining the misconfiguration

### Scenario: Conservative defaults prefer abstain over injection
- **Given** default thresholds and a bundle with composite confidence of 0.3
- **When** the Decision Engine selects an outcome
- **Then** the selected outcome is `abstain`

## Notes

- Conservative defaults are a VP8 product principle: "cheap correction beats
  confident drift" (ADR-019 §2).
- Threshold tuning should be driven by lab experiment results.
