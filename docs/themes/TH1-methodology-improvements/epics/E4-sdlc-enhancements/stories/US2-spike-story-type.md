---
id: TH1.E4.US2
title: "Add spike story type"
agents: [developer, reviewer]
skills: [bdd-stories, backlog-management]
acceptance-criteria:
  - AC1: "Story frontmatter supports type: standard | trivial | spike"
  - AC2: "Spikes produce ADR updates and feasibility reports, not production code"
  - AC3: "Spikes have acceptance criteria but BDD scenarios are optional"
  - AC4: "Product-owner and architect can create spike stories"
  - AC5: "bdd-stories skill documents the spike type and its differences"
depends-on: [TH1.E2.US1]
---

# TH1.E4.US2 — Add Spike Story Type

**As a** product builder, **I want** a spike story type for technical investigations, **so that** risky technical assumptions can be validated before committing the full backlog.

## Acceptance Criteria

- [ ] AC1: Story frontmatter supports `type: standard | trivial | spike`
- [ ] AC2: Spikes produce ADR updates and feasibility reports, not production code
- [ ] AC3: Spikes have acceptance criteria but BDD scenarios are optional
- [ ] AC4: Product-owner and architect can create spike stories
- [ ] AC5: `bdd-stories` skill documents the spike type and its differences

## BDD Scenarios

### Scenario: Spike story has type field
- **Given** a spike story file
- **When** I read the frontmatter
- **Then** it has `type: spike`

### Scenario: Spike produces investigation output
- **Given** a developer agent working on a spike
- **When** it completes the spike
- **Then** the output is an ADR update or feasibility report, not production code

### Scenario: Spike without BDD scenarios
- **Given** the story sizing rules in `bdd-stories` skill
- **When** I read the rules for spike stories
- **Then** BDD scenarios are documented as optional for spikes

### Scenario: Product-owner creates spikes
- **Given** the product-owner agent instructions
- **When** I read what story types it can create
- **Then** `spike` is listed as a valid type
