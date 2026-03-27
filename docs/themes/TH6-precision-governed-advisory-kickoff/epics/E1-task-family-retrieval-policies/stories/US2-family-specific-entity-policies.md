---
id: TH6.E1.US2
title: "Enforce family-specific entity allowlists and suppressions"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "Each kickoff family applies an explicit entity-type allowlist and suppression set before context projection."
  - AC2: "The selected family policy is visible in workflow-start output so a maintainer can inspect which entity classes were allowed or suppressed."
  - AC3: "Entity classes already known to cause contamination, including workflow-history artifacts outside the selected policy, are filtered before they reach the kickoff brief."
depends-on: [TH6.E1.US1]
---
# TH6.E1.US2 — Enforce family-specific entity allowlists and suppressions

**As a** maintainer tuning kickoff quality, **I want** each task family to apply
its own entity policy, **so that** graph context is filtered for relevance before
the agent sees it.

## Acceptance Criteria

- [ ] AC1: Each kickoff family applies an explicit entity-type allowlist and suppression set before context projection.
- [ ] AC2: The selected family policy is visible in workflow-start output so a maintainer can inspect which entity classes were allowed or suppressed.
- [ ] AC3: Entity classes already known to cause contamination, including workflow-history artifacts outside the selected policy, are filtered before they reach the kickoff brief.

## BDD Scenarios

### Scenario: Apply implementation-friendly entity policy
- **Given** a write-producing task is classified successfully
- **When** `graph workflow start` builds the kickoff candidate set
- **Then** the workflow uses the write-producing allowlist and suppresses unrelated entity classes before projection

### Scenario: Apply audit-specific filtering
- **Given** a verification or audit task is classified successfully
- **When** `graph workflow start` builds the kickoff candidate set
- **Then** the workflow uses the verification/audit allowlist instead of the write-producing policy

### Scenario: Suppress contamination-prone workflow history
- **Given** a candidate set contains entity types outside the selected family policy
- **When** the family policy is enforced
- **Then** those entity types are removed before the kickoff brief is assembled

