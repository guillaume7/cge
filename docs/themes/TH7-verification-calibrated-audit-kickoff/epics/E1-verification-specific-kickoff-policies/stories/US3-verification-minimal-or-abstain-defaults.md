---
id: TH7.E1.US3
title: "Introduce verification-safe minimal and abstain defaults"
type: standard
priority: medium
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "Verification sub-profiles can default to minimal or abstain when policy-aligned evidence is sparse."
  - AC2: "Stats-oriented audits can prefer a minimal or abstained kickoff instead of forcing a full injected brief."
  - AC3: "The machine-readable advisory state explains whether a verification task injected, minimized, or abstained and why."
depends-on: [TH7.E1.US2]
---
# TH7.E1.US3 — Introduce verification-safe minimal and abstain defaults

**As a** maintainer using CGE for audits, **I want** verification tasks to
prefer minimal or abstained kickoff when evidence is weak, **so that** the
graph does not dominate an audit with low-value context.

## Acceptance Criteria

- [ ] AC1: Verification sub-profiles can default to minimal or abstain when policy-aligned evidence is sparse.
- [ ] AC2: Stats-oriented audits can prefer a minimal or abstained kickoff instead of forcing a full injected brief.
- [ ] AC3: The machine-readable advisory state explains whether a verification task injected, minimized, or abstained and why.

## BDD Scenarios

### Scenario: Downgrade a sparse stats audit to minimal
- **Given** a stats audit with only weak policy-aligned graph evidence
- **When** workflow start evaluates kickoff policy
- **Then** the result chooses minimal instead of full injection

### Scenario: Abstain from a verification task with no aligned evidence
- **Given** a verification task whose profile-aligned evidence is missing
- **When** workflow start evaluates kickoff policy
- **Then** the result abstains and exposes the reason machine-readably

### Scenario: Explain the verification downgrade outcome
- **Given** a verification task that minimized or abstained
- **When** the kickoff result is returned
- **Then** the advisory output records the effective mode and verification-specific reason codes
