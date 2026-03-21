---
id: TH1.E5.US2
title: "Add priority field to stories"
agents: [developer, reviewer]
skills: [bdd-stories, backlog-management]
acceptance-criteria:
  - AC1: "Story frontmatter supports optional priority: high | medium | low (default: medium)"
  - AC2: "Orchestrator prefers higher-priority eligible stories when selecting work"
  - AC3: "backlog-management skill documents priority-based selection"
depends-on: [TH1.E1.US1]
---

# TH1.E5.US2 — Add Priority Field to Stories

**As a** product builder, **I want** stories to have a priority field, **so that** the orchestrator can pick the highest-value eligible story when multiple choices exist.

## Acceptance Criteria

- [ ] AC1: Story frontmatter supports optional `priority: high | medium | low` (default: `medium`)
- [ ] AC2: Orchestrator prefers higher-priority eligible stories when selecting work
- [ ] AC3: `backlog-management` skill documents priority-based selection

## BDD Scenarios

### Scenario: Priority field in frontmatter
- **Given** the story frontmatter template in `bdd-stories` skill
- **When** I read the optional fields
- **Then** `priority` is listed with values `high | medium | low` and default `medium`

### Scenario: Orchestrator priority selection
- **Given** two eligible stories: one `high` priority and one `medium` priority
- **When** the orchestrator selects the next story
- **Then** it picks the `high` priority story first

### Scenario: Default priority
- **Given** a story with no explicit priority field
- **When** the orchestrator evaluates its priority
- **Then** it treats it as `medium`
