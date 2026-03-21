---
id: TH1.E1.US2
title: "Single source of truth for status"
agents: [developer, reviewer]
skills: [bdd-stories, backlog-management]
acceptance-criteria:
  - AC1: "backlog.md (or .yaml) is documented as the sole authoritative status store"
  - AC2: "Story file frontmatter template no longer includes a status field"
  - AC3: "backlog-management skill removes the 'mirror status in story file' instruction"
  - AC4: "Orchestrator agent instructions only read/write status in the backlog file"
depends-on: [TH1.E1.US1]
---

# TH1.E1.US2 — Single Source of Truth for Status

**As a** methodology user, **I want** status to live only in the backlog file, **so that** there's no risk of inconsistent state between backlog and story files.

## Acceptance Criteria

- [ ] AC1: Backlog file is documented as the sole authoritative status store
- [ ] AC2: Story file frontmatter template no longer includes a `status` field
- [ ] AC3: `backlog-management` skill removes the "mirror status in story file" instruction
- [ ] AC4: Orchestrator agent instructions only read/write status in the backlog file

## BDD Scenarios

### Scenario: Story template has no status field
- **Given** the story frontmatter template in `bdd-stories/SKILL.md`
- **When** I read the required frontmatter fields
- **Then** `status` is not listed as a field

### Scenario: Backlog management declares single truth
- **Given** the `backlog-management/SKILL.md` skill file
- **When** I search for the state update protocol
- **Then** it says "write status only to the backlog file" with no mention of mirroring to story files

### Scenario: Orchestrator does not update story file status
- **Given** the orchestrator agent instructions
- **When** I search for status update steps
- **Then** it only mentions updating the backlog file, not individual story files

### Scenario: copilot-instructions.md reflects single truth
- **Given** the workspace instructions in `copilot-instructions.md`
- **When** I read the backlog state file section
- **Then** there's no mention of mirroring status to story frontmatter
