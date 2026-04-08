---
id: TH8.E1.US2
title: "Compute composite confidence from evaluation dimensions"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "The Context Evaluator produces a single composite confidence score per candidate from the three dimension scores."
  - AC2: "Dimension weights are configurable and default to equal weighting."
  - AC3: "The composite confidence score is a normalized float between 0.0 and 1.0."
  - AC4: "The evaluator also computes a bundle-level composite confidence that summarizes the overall quality of the surviving candidate set."
depends-on:
  - TH8.E1.US1
---
# TH8.E1.US2 — Compute composite confidence from evaluation dimensions

**As a** Decision Engine consumer, **I want** a single composite confidence
score per candidate and per bundle, **so that** threshold-driven outcome
selection has a clear numeric input.

## Acceptance Criteria

- [ ] AC1: The Context Evaluator produces a single composite confidence score per candidate from the three dimension scores.
- [ ] AC2: Dimension weights are configurable and default to equal weighting.
- [ ] AC3: The composite confidence score is a normalized float between 0.0 and 1.0.
- [ ] AC4: The evaluator also computes a bundle-level composite confidence that summarizes the overall quality of the surviving candidate set.

## BDD Scenarios

### Scenario: Compute composite with equal weights
- **Given** a candidate scored at relevance=0.9, consistency=0.6, usefulness=0.8
- **And** dimension weights are set to equal (default)
- **When** the evaluator computes the composite confidence
- **Then** the composite confidence is approximately 0.77

### Scenario: Compute composite with custom weights
- **Given** a candidate scored at relevance=0.9, consistency=0.6, usefulness=0.8
- **And** dimension weights are configured as relevance=0.5, consistency=0.3, usefulness=0.2
- **When** the evaluator computes the composite confidence
- **Then** the composite confidence reflects the weighted combination

### Scenario: Compute bundle-level confidence
- **Given** three candidates with composite confidence scores of 0.9, 0.5, and 0.3
- **When** the evaluator computes the bundle-level composite
- **Then** the bundle-level confidence summarizes the surviving set quality
- **And** the bundle-level score is between 0.0 and 1.0

### Scenario: Single candidate bundle
- **Given** one candidate with composite confidence 0.85
- **When** the evaluator computes the bundle-level composite
- **Then** the bundle-level confidence equals the single candidate's composite

### Scenario: All candidates score zero
- **Given** three candidates that each score 0.0 on all dimensions
- **When** the evaluator computes composite scores
- **Then** all per-candidate composites are 0.0
- **And** the bundle-level composite is 0.0

## Notes

- The composite confidence is the primary input to the Decision Engine (ADR-019).
- Weight configuration uses the same local config pattern as existing threshold
  settings.
