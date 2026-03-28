---
id: TH7.E1.US2
title: "Apply verification-specific suppressions and allowlists"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "Stats-oriented audits suppress workflow-finish and unrelated workflow artifacts by default."
  - AC2: "Workflow-verification tasks suppress unrelated graph-health and contradiction-heavy context unless explicitly relevant."
  - AC3: "The kickoff result returns only verification-policy-compliant entities for the selected verification sub-profile."
depends-on: [TH7.E1.US1]
---
# TH7.E1.US2 — Apply verification-specific suppressions and allowlists

**As a** maintainer trying to stop audit contamination, **I want** each
verification sub-profile to filter different entity types, **so that** the
kickoff brief only carries evidence aligned with the audit task.

## Acceptance Criteria

- [ ] AC1: Stats-oriented audits suppress workflow-finish and unrelated workflow artifacts by default.
- [ ] AC2: Workflow-verification tasks suppress unrelated graph-health and contradiction-heavy context unless explicitly relevant.
- [ ] AC3: The kickoff result returns only verification-policy-compliant entities for the selected verification sub-profile.

## BDD Scenarios

### Scenario: Suppress workflow artifacts during a stats audit
- **Given** a stats-oriented verification task
- **When** workflow start builds the kickoff context
- **Then** workflow-finish and unrelated workflow-management artifacts are excluded from the brief

### Scenario: Suppress graph-health clutter during workflow verification
- **Given** a workflow-verification task
- **When** workflow start builds the kickoff context
- **Then** unrelated graph-health and contradiction-heavy context is excluded from the brief

### Scenario: Keep only policy-aligned evidence for a general verification task
- **Given** a general evidence audit task
- **When** workflow start projects the kickoff context
- **Then** the returned entities conform to the selected verification profile instead of leaking unrelated families
