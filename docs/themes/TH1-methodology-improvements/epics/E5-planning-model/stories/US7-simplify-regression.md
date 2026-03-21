---
id: TH1.E5.US7
title: "Simplify regression testing"
agents: [developer, reviewer]
skills: [the-copilot-build-method]
acceptance-criteria:
  - AC1: "Theme completion testing is renamed to 'full test suite verification'"
  - AC2: "No pretense of a special testing mode — it's just running all tests"
  - AC3: "Developer agent testing modes reflect the simplification"
depends-on: [TH1.E2.US1]
---

# TH1.E5.US7 — Simplify Regression Testing

**As a** methodology user, **I want** theme-level regression testing simplified, **so that** the methodology doesn't pretend a full test run is a special testing mode.

## Acceptance Criteria

- [ ] AC1: Theme completion testing is renamed to "full test suite verification"
- [ ] AC2: No pretense of a special testing mode — it's just running all tests
- [ ] AC3: Developer agent testing modes reflect the simplification

## BDD Scenarios

### Scenario: Renamed in methodology docs
- **Given** the `the-copilot-build-method` skill
- **When** I read the theme completion steps
- **Then** it says "full test suite verification" not "regression testing"

### Scenario: Developer has simplified test modes
- **Given** the developer agent instructions
- **When** I read the test modes
- **Then** there are only 2: story testing and integration testing (at epic level)

### Scenario: Theme completion just runs everything
- **Given** the orchestrator's theme completion ceremony
- **When** I read the testing step
- **Then** it says "run the full test suite" without a special mode
