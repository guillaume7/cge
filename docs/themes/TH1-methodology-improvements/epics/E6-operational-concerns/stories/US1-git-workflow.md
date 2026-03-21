---
id: TH1.E6.US1
title: "Add git workflow guidance"
agents: [developer, reviewer]
skills: [architecture-decisions]
acceptance-criteria:
  - AC1: "architecture-decisions skill includes a git workflow section"
  - AC2: "Convention: one commit per story with conventional commit message referencing qualified story ID"
  - AC3: "Optional branch-per-epic strategy documented"
  - AC4: "Orchestrator instructions mention committing after story completion"
depends-on: [TH1.E2.US1]
---

# TH1.E6.US1 — Add Git Workflow Guidance

**As a** product builder, **I want** the methodology to include git workflow guidance, **so that** stories produce clean, traceable commits.

## Acceptance Criteria

- [ ] AC1: `architecture-decisions` skill includes a git workflow section
- [ ] AC2: Convention: one commit per story with conventional commit message referencing qualified story ID
- [ ] AC3: Optional branch-per-epic strategy documented
- [ ] AC4: Orchestrator instructions mention committing after story completion

## BDD Scenarios

### Scenario: Commit convention documented
- **Given** the `architecture-decisions` skill
- **When** I read the git workflow section
- **Then** it specifies: one commit per story, message format `feat(TH1.E1.US1): <description>`

### Scenario: Branch strategy optional
- **Given** the git workflow section
- **When** I read the branching guidance
- **Then** it suggests optional branch-per-epic as a pattern

### Scenario: Orchestrator commits after story
- **Given** the orchestrator's story completion step
- **When** a story transitions to `done`
- **Then** the instructions mention creating a commit with the story ID
