---
id: TH7.E2.US3
title: "Minimize workflow-verification kickoff when workflow artifacts dominate"
type: standard
priority: medium
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "Workflow-verification tasks downgrade to minimal or abstain when workflow artifacts are present but not specific enough to the audit request."
  - AC2: "Workflow-verification tasks keep only the smallest sufficient evidence set for kickoff when confidence is borderline."
  - AC3: "The downgrade preserves machine-readable next-step guidance for pulling more graph evidence on demand."
depends-on: [TH7.E2.US2]
---
# TH7.E2.US3 — Minimize workflow-verification kickoff when workflow artifacts dominate

**As a** maintainer verifying workflow behavior, **I want** workflow-related
audits to minimize or abstain when the retrieved workflow context is too broad,
**so that** the audit is not swamped by generic workflow history.

## Acceptance Criteria

- [ ] AC1: Workflow-verification tasks downgrade to minimal or abstain when workflow artifacts are present but not specific enough to the audit request.
- [ ] AC2: Workflow-verification tasks keep only the smallest sufficient evidence set for kickoff when confidence is borderline.
- [ ] AC3: The downgrade preserves machine-readable next-step guidance for pulling more graph evidence on demand.

## BDD Scenarios

### Scenario: Downgrade a broad workflow-verification brief
- **Given** a workflow-verification task whose retrieved workflow evidence is broad but weakly aligned
- **When** workflow start evaluates the kickoff
- **Then** the result minimizes or abstains instead of emitting a full injected brief

### Scenario: Keep the smallest sufficient workflow evidence set
- **Given** a workflow-verification task with a few strongly aligned workflow entities
- **When** workflow start projects the kickoff
- **Then** the result retains only the smallest sufficient evidence set instead of a broad workflow history

### Scenario: Preserve next-step guidance after workflow downgrade
- **Given** a workflow-verification task that downgraded to minimal or abstain
- **When** the result is returned
- **Then** the output still tells the caller how to pull additional graph evidence on demand
