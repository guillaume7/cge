---
id: TH7.E3.US1
title: "Capture richer attribution for verification-focused reruns"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "Graph-backed rerun artifacts preserve the raw `workflow.start` response for each with-graph verification run."
  - AC2: "Baseline rerun artifacts preserve prompt-surface metadata for each without-graph verification run."
  - AC3: "The generated batch summary records enough attribution to roll up effective modes, abstention rates, confidence, and token deltas by verification profile."
depends-on: [TH7.E2.US3]
---
# TH7.E3.US1 — Capture richer attribution for verification-focused reruns

**As a** maintainer interpreting calibration results, **I want** rerun artifacts
to preserve enough kickoff and baseline metadata, **so that** a surprising audit
result can be explained from artifacts alone.

## Acceptance Criteria

- [ ] AC1: Graph-backed rerun artifacts preserve the raw `workflow.start` response for each with-graph verification run.
- [ ] AC2: Baseline rerun artifacts preserve prompt-surface metadata for each without-graph verification run.
- [ ] AC3: The generated batch summary records enough attribution to roll up effective modes, abstention rates, confidence, and token deltas by verification profile.

## BDD Scenarios

### Scenario: Preserve raw workflow-start output for a graph-backed verification run
- **Given** a verification-focused rerun executes a with-graph condition
- **When** the batch artifacts are written
- **Then** the raw `workflow.start` response is preserved alongside the run outputs

### Scenario: Preserve prompt-surface metadata for a baseline verification run
- **Given** a verification-focused rerun executes a without-graph condition
- **When** the batch artifacts are written
- **Then** the baseline prompt-surface metadata is preserved alongside the run outputs

### Scenario: Roll up verification attribution into the batch summary
- **Given** a verification-focused rerun has completed several paired runs
- **When** the batch summary is generated
- **Then** the summary reports effective mode, abstention, confidence, and token-delta rollups by verification profile
