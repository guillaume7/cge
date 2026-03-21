---
id: TH1.E5.US1
title: "Relax VP-to-TH mapping to 1:N"
agents: [developer, reviewer]
skills: [the-copilot-build-method, backlog-management]
acceptance-criteria:
  - AC1: "VP:TH mapping is documented as 1:N (one VP can produce multiple themes)"
  - AC2: "vision-ref field in backlog schema accepts a list of VP references"
  - AC3: "Naming conventions updated to reflect flexible mapping"
depends-on: [TH1.E1.US1]
---

# TH1.E5.US1 — Relax VP-to-TH Mapping to 1:N

**As a** product planner, **I want** one vision phase to map to multiple themes, **so that** cross-cutting concerns and large feature sets aren't artificially crammed into one theme.

## Acceptance Criteria

- [ ] AC1: VP:TH mapping documented as 1:N (not 1:1)
- [ ] AC2: `vision-ref` field accepts a string or list of VP paths
- [ ] AC3: Naming conventions updated to reflect flexible mapping

## BDD Scenarios

### Scenario: One VP maps to multiple themes
- **Given** a vision phase VP1 with many features
- **When** the product-owner decomposes it
- **Then** it can create TH1 and TH2 both referencing VP1

### Scenario: vision-ref accepts list
- **Given** the backlog YAML schema
- **When** I read the `vision-ref` field definition
- **Then** it accepts a string or a list of VP paths

### Scenario: Theme numbering is sequential
- **Given** VP1 produces TH1 and TH2
- **When** VP2 produces another theme
- **Then** it becomes TH3 (sequential, not tied to VP numbering)
