---
id: TH1.E1.US1
title: "Introduce qualified story IDs"
agents: [developer, reviewer]
skills: [bdd-stories, backlog-management]
acceptance-criteria:
  - AC1: "All story IDs in backlog use fully-qualified dot-notation TH<n>.E<m>.US<l>"
  - AC2: "All depends-on references use fully-qualified IDs"
  - AC3: "backlog-management skill documents qualified ID format as the canonical schema"
  - AC4: "bdd-stories skill shows qualified ID in frontmatter template"
  - AC5: "copilot-instructions.md examples use qualified IDs"
depends-on: []
---

# TH1.E1.US1 — Introduce Qualified Story IDs

**As a** methodology user, **I want** all story IDs to be fully qualified (TH1.E1.US1), **so that** cross-epic dependencies are unambiguous and never collide.

## Acceptance Criteria

- [ ] AC1: All story IDs in backlog use fully-qualified dot-notation `TH<n>.E<m>.US<l>`
- [ ] AC2: All `depends-on` references use fully-qualified IDs
- [ ] AC3: `backlog-management` skill documents qualified ID format as the canonical schema
- [ ] AC4: `bdd-stories` skill shows qualified ID in frontmatter template
- [ ] AC5: `copilot-instructions.md` examples use qualified IDs

## BDD Scenarios

### Scenario: Backlog schema uses qualified IDs
- **Given** the backlog YAML schema in `backlog-management/SKILL.md`
- **When** I read the story ID examples
- **Then** all IDs follow the pattern `TH<n>.E<m>.US<l>` (e.g., `TH1.E1.US1`)

### Scenario: Cross-epic dependency is unambiguous
- **Given** two epics E1 and E2 each with a story US1
- **When** E2.US2 declares `depends-on: [TH1.E1.US1]`
- **Then** the dependency clearly refers to E1's US1, not E2's US1

### Scenario: Story frontmatter uses qualified ID
- **Given** the story file template in `bdd-stories/SKILL.md`
- **When** I read the `id:` field in the frontmatter example
- **Then** it shows `id: TH1.E1.US1` (qualified format)

### Scenario: copilot-instructions.md examples updated
- **Given** the example backlog structure in `copilot-instructions.md`
- **When** I read story entries in the example YAML
- **Then** all story IDs and depends-on values use qualified format
