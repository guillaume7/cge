---
id: TH2.E2.US3
title: "Return machine-readable hygiene suggestion plans"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "`graph hygiene` suggest mode returns duplicate, orphan, and contradiction suggestions inside a stable machine-readable plan envelope."
  - AC2: "Each suggested action includes an action type, target identifiers, and explanation fields suitable for downstream agent review."
  - AC3: "Structured stdout output is equivalent to file output for the same hygiene suggestion result."
depends-on: [TH2.E2.US2]
---
# TH2.E2.US3 — Return machine-readable hygiene suggestion plans

**As an** agent, **I want** `graph hygiene` to return a stable suggestion plan,
**so that** I can review, transform, and later apply cleanup actions without
parsing ad hoc prose.

## Acceptance Criteria

- [ ] AC1: `graph hygiene` suggest mode returns duplicate, orphan, and contradiction suggestions inside a stable machine-readable plan envelope.
- [ ] AC2: Each suggested action includes an action type, target identifiers, and explanation fields suitable for downstream agent review.
- [ ] AC3: Structured stdout output is equivalent to file output for the same hygiene suggestion result.

## BDD Scenarios

### Scenario: Return a structured hygiene suggestion plan
- **Given** a repo-local graph contains one or more hygiene candidates
- **When** an agent runs `graph hygiene`
- **Then** the command returns a structured suggestion plan with machine-readable action details

### Scenario: Return an empty but valid plan when no hygiene work is needed
- **Given** a repo-local graph contains no cleanup candidates
- **When** an agent runs `graph hygiene`
- **Then** the command returns a valid empty suggestion plan rather than special-case output

### Scenario: Return matching stdout and file plans
- **Given** `graph hygiene` supports both stdout output and file output
- **When** an agent executes both modes for the same graph snapshot
- **Then** the serialized suggestion plan is equivalent in both places
