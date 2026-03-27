---
id: TH4.E3.US1
title: "Add separated evaluation scoring with blinding support"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "Evaluation scores are stored as separate records under `.graph/lab/evaluations/<run-id>.json`, linked to run IDs but outside the run record."
  - AC2: "The evaluation step can present run outcomes in a condition-blind format that strips condition metadata so evaluators judge outputs on merit."
  - AC3: "Both automated rubric-based and human-sourced evaluation records are supported with an explicit evaluator identity field."
depends-on: [TH4.E2.US2]
---
# TH4.E3.US1 — Add separated evaluation scoring with blinding support

**As an** experiment evaluator, **I want** to score run outcomes separately from
execution and optionally blind to condition, **so that** quality judgments are
not biased by knowing which condition produced the output.

## Acceptance Criteria

- [ ] AC1: Evaluation scores are stored as separate records under `.graph/lab/evaluations/<run-id>.json`, linked to run IDs but outside the run record.
- [ ] AC2: The evaluation step can present run outcomes in a condition-blind format that strips condition metadata so evaluators judge outputs on merit.
- [ ] AC3: Both automated rubric-based and human-sourced evaluation records are supported with an explicit evaluator identity field.

## BDD Scenarios

### Scenario: Store an evaluation record separately from the run record
- **Given** a completed run exists in the run ledger with run ID `run-001`
- **When** an evaluator submits scores for success, quality, and resumability
- **Then** an evaluation record is written to `.graph/lab/evaluations/run-001.json` with the scores, evaluator identity, and timestamp, and the original run record remains unchanged

### Scenario: Present run outcomes in condition-blind format
- **Given** a completed run exists with a known experimental condition
- **When** the evaluation step presents the run for blinded scoring
- **Then** the presented output includes task description and outcome artifacts but strips the condition ID, workflow mode, and any condition-revealing metadata

### Scenario: Reject an evaluation for a nonexistent run ID
- **Given** no run record exists for run ID `run-999`
- **When** an evaluator attempts to submit scores for `run-999`
- **Then** the evaluation step returns a structured error identifying the missing run record

### Scenario: Support re-evaluation under an improved rubric
- **Given** a run has already been evaluated once
- **When** a new evaluation is submitted with a different evaluator identity or rubric version
- **Then** the new evaluation record is accepted and linked to the same run ID alongside the previous evaluation
