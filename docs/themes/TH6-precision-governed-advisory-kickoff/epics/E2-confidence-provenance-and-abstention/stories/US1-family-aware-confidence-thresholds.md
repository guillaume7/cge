---
id: TH6.E2.US1
title: "Compute family-aware kickoff confidence and thresholding"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "Workflow start computes a kickoff-confidence signal using the selected task family and the retrieved candidate evidence."
  - AC2: "Family-specific thresholds determine whether the outcome is inject, minimal, or abstain."
  - AC3: "The workflow-start output exposes the confidence outcome in a machine-readable form."
depends-on: [TH6.E1.US3]
---
# TH6.E2.US1 — Compute family-aware kickoff confidence and thresholding

**As a** maintainer trying to keep kickoff net-positive, **I want** workflow
start to compute family-aware confidence, **so that** the system can step back
when retrieval quality is weak.

## Acceptance Criteria

- [ ] AC1: Workflow start computes a kickoff-confidence signal using the selected task family and the retrieved candidate evidence.
- [ ] AC2: Family-specific thresholds determine whether the outcome is inject, minimal, or abstain.
- [ ] AC3: The workflow-start output exposes the confidence outcome in a machine-readable form.

## BDD Scenarios

### Scenario: Inject when confidence is high
- **Given** a write-producing task retrieves strong, policy-compliant graph matches
- **When** workflow start evaluates kickoff confidence
- **Then** the result chooses the inject outcome and records the confidence state

### Scenario: Use minimal kickoff for borderline evidence
- **Given** a diagnostic task retrieves some relevant graph context but not enough for a full brief
- **When** workflow start evaluates kickoff confidence
- **Then** the result chooses the minimal outcome instead of full injection

### Scenario: Abstain when confidence is below threshold
- **Given** an ambiguous task retrieves weak or conflicting graph evidence
- **When** workflow start evaluates kickoff confidence
- **Then** the result abstains and exposes the low-confidence decision machine-readably

