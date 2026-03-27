---
id: TH6.E2.US3
title: "Recommend pull-on-demand when kickoff confidence is low"
type: standard
priority: medium
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "When workflow start abstains because kickoff confidence is low, the output includes a machine-readable recommendation to use pull-on-demand graph retrieval instead of pushed kickoff context."
  - AC2: "The abstention recommendation explains whether the trigger was low confidence, family policy, or both."
  - AC3: "Low-confidence abstention does not prevent later graph retrieval by the delegated agent."
depends-on: [TH6.E2.US1, TH6.E2.US2]
---
# TH6.E2.US3 — Recommend pull-on-demand when kickoff confidence is low

**As a** delegated agent starting a task, **I want** low-confidence kickoff to
recommend pull-on-demand instead of forcing graph context, **so that** I can
fetch graph help later if I actually need it.

## Acceptance Criteria

- [ ] AC1: When workflow start abstains because kickoff confidence is low, the output includes a machine-readable recommendation to use pull-on-demand graph retrieval instead of pushed kickoff context.
- [ ] AC2: The abstention recommendation explains whether the trigger was low confidence, family policy, or both.
- [ ] AC3: Low-confidence abstention does not prevent later graph retrieval by the delegated agent.

## BDD Scenarios

### Scenario: Recommend pull-on-demand after low-confidence abstention
- **Given** workflow start abstains because confidence is below the selected family threshold
- **When** the result is returned
- **Then** the output includes a machine-readable recommendation to pull graph context later if needed

### Scenario: Distinguish low-confidence abstention from family-default abstention
- **Given** one task abstains due to low confidence and another abstains due to reporting-family policy
- **When** workflow start returns the results
- **Then** each result explains which abstention reason applied

### Scenario: Preserve later graph access after abstention
- **Given** a delegated task started with a low-confidence abstention result
- **When** the delegated agent later decides to query the graph directly
- **Then** the graph retrieval surface remains available and unchanged

