---
id: TH7.E2.US1
title: "Tighten verification thresholds and token budgets"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "Verification sub-profiles use stricter confidence thresholds than write-producing tasks."
  - AC2: "Verification sub-profiles use smaller default token budgets than implementation-oriented kickoff where appropriate."
  - AC3: "Threshold and budget outcomes are exposed machine-readably for later rerun attribution."
depends-on: [TH7.E1.US3]
---
# TH7.E2.US1 — Tighten verification thresholds and token budgets

**As a** maintainer trying to make audits cheap and precise, **I want**
verification tasks to use stricter thresholds and smaller budgets, **so that**
borderline audit evidence does not expand into costly full kickoff briefs.

## Acceptance Criteria

- [ ] AC1: Verification sub-profiles use stricter confidence thresholds than write-producing tasks.
- [ ] AC2: Verification sub-profiles use smaller default token budgets than implementation-oriented kickoff where appropriate.
- [ ] AC3: Threshold and budget outcomes are exposed machine-readably for later rerun attribution.

## BDD Scenarios

### Scenario: Use a stricter threshold for verification than for implementation
- **Given** a verification task and an implementation task with similarly mixed evidence
- **When** workflow start evaluates kickoff confidence
- **Then** the verification task downgrades sooner than the implementation task

### Scenario: Cap token budget for a stats audit
- **Given** a stats-oriented verification task with several candidate entities
- **When** workflow start projects kickoff context
- **Then** the projected verification brief uses the smaller verification-specific token cap

### Scenario: Expose the chosen threshold and budget outcome
- **Given** a verification task whose kickoff was evaluated
- **When** the machine-readable result is returned
- **Then** the advisory output reflects the stricter verification calibration outcome
