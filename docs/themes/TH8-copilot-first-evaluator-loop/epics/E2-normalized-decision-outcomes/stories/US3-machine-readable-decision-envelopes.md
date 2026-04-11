---
id: TH8.E2.US3
title: "Compose machine-readable decision envelopes"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "Every decision is returned as a structured JSON envelope containing the selected outcome, evaluator scores, and the surviving context bundle (when outcome is continue or minimal)."
  - AC2: "The envelope includes the composite confidence score and per-candidate dimension scores."
  - AC3: "The envelope is emittable to stdout so that consuming agents can parse it from CLI output."
  - AC4: "When the outcome is abstain, the envelope contains the outcome and scores but no context bundle."
  - AC5: "When the outcome is backtrack, the envelope contains the outcome, the current and prior scores, and no context bundle."
depends-on:
  - TH8.E2.US2
---
# TH8.E2.US3 — Compose machine-readable decision envelopes

**As a** consuming agent, **I want** every decision returned as a structured
envelope, **so that** I can programmatically inspect the outcome, scores, and
surviving context without parsing unstructured text.

## Acceptance Criteria

- [ ] AC1: Every decision is returned as a structured JSON envelope containing the selected outcome, evaluator scores, and the surviving context bundle (when outcome is continue or minimal).
- [ ] AC2: The envelope includes the composite confidence score and per-candidate dimension scores.
- [ ] AC3: The envelope is emittable to stdout so that consuming agents can parse it from CLI output.
- [ ] AC4: When the outcome is abstain, the envelope contains the outcome and scores but no context bundle.
- [ ] AC5: When the outcome is backtrack, the envelope contains the outcome, the current and prior scores, and no context bundle.

## BDD Scenarios

### Scenario: Envelope for a continue outcome
- **Given** the Decision Engine selects `continue` with three surviving candidates
- **When** the decision envelope is composed
- **Then** the JSON envelope contains `"outcome": "continue"`, composite confidence, per-candidate scores, and the full context bundle

### Scenario: Envelope for a minimal outcome
- **Given** the Decision Engine selects `minimal` with one surviving candidate out of four
- **When** the decision envelope is composed
- **Then** the JSON envelope contains `"outcome": "minimal"` and only the highest-scored candidate in the context bundle

### Scenario: Envelope for an abstain outcome
- **Given** the Decision Engine selects `abstain`
- **When** the decision envelope is composed
- **Then** the JSON envelope contains `"outcome": "abstain"`, the evaluator scores, and no context bundle

### Scenario: Envelope for a backtrack outcome
- **Given** the Decision Engine selects `backtrack` with current score 0.4 and prior score 0.7
- **When** the decision envelope is composed
- **Then** the JSON envelope contains `"outcome": "backtrack"`, both scores, and no context bundle

### Scenario: Envelope is valid JSON on stdout
- **Given** any decision outcome
- **When** the envelope is emitted to stdout
- **Then** the output is valid JSON parseable by standard JSON libraries

## Notes

- The decision envelope schema is defined in ADR-019 §3.
- Attribution records (ADR-021) will be added to the envelope in E3.
