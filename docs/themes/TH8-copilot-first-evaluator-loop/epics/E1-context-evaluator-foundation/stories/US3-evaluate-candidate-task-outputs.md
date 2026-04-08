---
id: TH8.E1.US3
title: "Evaluate candidate task outputs for iterative critique"
type: standard
priority: medium
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "The Context Evaluator exposes an `evaluateOutput` method that scores a candidate task output against the original task description."
  - AC2: "Output evaluation uses the same three dimensions (relevance, consistency, usefulness) and composite confidence as context evaluation."
  - AC3: "Output evaluation can compare the candidate output against a prior output to detect quality regression."
  - AC4: "The evaluator returns a structured result that the Decision Engine can use to select continue, backtrack, or revise outcomes."
depends-on:
  - TH8.E1.US2
---
# TH8.E1.US3 — Evaluate candidate task outputs for iterative critique

**As a** consuming agent in an iterative generate/critique loop, **I want** the
evaluator to score candidate task outputs, **so that** I can decide whether to
accept, revise, or backtrack before trusting the output.

## Acceptance Criteria

- [ ] AC1: The Context Evaluator exposes an `evaluateOutput` method that scores a candidate task output against the original task description.
- [ ] AC2: Output evaluation uses the same three dimensions (relevance, consistency, usefulness) and composite confidence as context evaluation.
- [ ] AC3: Output evaluation can compare the candidate output against a prior output to detect quality regression.
- [ ] AC4: The evaluator returns a structured result that the Decision Engine can use to select continue, backtrack, or revise outcomes.

## BDD Scenarios

### Scenario: Score a high-quality task output
- **Given** a task description requesting a specific code change
- **And** a candidate output that addresses the task completely
- **When** the evaluator scores the output
- **Then** the composite confidence is above the continue threshold
- **And** the result includes per-dimension scores

### Scenario: Detect quality regression between iterations
- **Given** a prior output with composite confidence 0.75
- **And** a new candidate output with lower relevance and usefulness
- **When** the evaluator scores the new output with the prior output as baseline
- **Then** the result indicates quality regression
- **And** the composite confidence is lower than the prior output's score

### Scenario: Score a partial output
- **Given** a task description with multiple requirements
- **And** a candidate output that addresses only one requirement
- **When** the evaluator scores the output
- **Then** the usefulness score reflects partial completion
- **And** the composite confidence is moderate

### Scenario: Evaluate output without prior baseline
- **Given** a task description and a first-iteration candidate output
- **And** no prior output exists for comparison
- **When** the evaluator scores the output
- **Then** the result includes dimension scores and composite confidence
- **And** no regression indicator is present

### Scenario: Handle empty output
- **Given** a task description and an empty candidate output
- **When** the evaluator scores the output
- **Then** all dimension scores are 0.0
- **And** the composite confidence is 0.0

## Notes

- Output evaluation supports the iterative critique/revise cycle described in
  the VP8 vision (ADR-018 §4).
- The evaluator does not generate revised outputs — it only scores them. The
  consuming agent decides what to do with the score.
