---
id: TH1.E5.US3
title: "Add size/complexity field to stories"
agents: [developer, reviewer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "Story frontmatter supports optional size: S | M | L"
  - AC2: "Product-owner assigns size during planning"
  - AC3: "bdd-stories skill documents sizing guidance"
depends-on: [TH1.E1.US1]
---

# TH1.E5.US3 — Add Size/Complexity Field to Stories

**As a** product builder, **I want** stories to have an estimated size, **so that** the orchestrator and users can anticipate which stories will be long-running.

## Acceptance Criteria

- [ ] AC1: Story frontmatter supports optional `size: S | M | L`
- [ ] AC2: Product-owner assigns size during planning
- [ ] AC3: `bdd-stories` skill documents sizing guidance

## BDD Scenarios

### Scenario: Size field in frontmatter
- **Given** the story frontmatter template
- **When** I read the optional fields
- **Then** `size` is listed with values `S | M | L`

### Scenario: Product-owner sets size
- **Given** the product-owner agent instructions
- **When** I read the story creation steps
- **Then** it mentions estimating story size

### Scenario: Size relates to existing rules
- **Given** the existing sizing rules (2-6 ACs, 3-8 scenarios)
- **When** I read the size field guidance
- **Then** S/M/L correlates with AC and scenario counts
