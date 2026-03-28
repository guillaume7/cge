---
id: TH7.E2.US2
title: "Add verification-specific contamination reason codes"
type: standard
priority: medium
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "Workflow start emits verification-specific advisory reason codes when stats or workflow-verification tasks are downgraded due to likely contamination."
  - AC2: "Reason codes distinguish between sparse aligned evidence and off-profile contamination."
  - AC3: "The reason-code set is stable enough to be aggregated by the experiment harness."
depends-on: [TH7.E2.US1]
---
# TH7.E2.US2 — Add verification-specific contamination reason codes

**As a** maintainer reading rerun artifacts, **I want** verification downgrade
paths to use explicit reason codes, **so that** we can tell whether a task
minimized because evidence was sparse or because the wrong evidence dominated.

## Acceptance Criteria

- [ ] AC1: Workflow start emits verification-specific advisory reason codes when stats or workflow-verification tasks are downgraded due to likely contamination.
- [ ] AC2: Reason codes distinguish between sparse aligned evidence and off-profile contamination.
- [ ] AC3: The reason-code set is stable enough to be aggregated by the experiment harness.

## BDD Scenarios

### Scenario: Emit a contamination reason code for a stats audit
- **Given** a stats audit whose highest-ranked evidence is mostly workflow-specific
- **When** workflow start evaluates advisory state
- **Then** the result records a verification contamination reason code

### Scenario: Emit a sparse-evidence reason code for a broad audit
- **Given** a general verification task with too little aligned evidence
- **When** workflow start evaluates advisory state
- **Then** the result records a sparse-aligned-evidence reason code instead of a contamination code

### Scenario: Preserve stable reason codes for later aggregation
- **Given** multiple verification tasks are evaluated across a rerun batch
- **When** the experiment harness reads advisory output
- **Then** the reason-code field is stable enough to roll up by frequency and family
