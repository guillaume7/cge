---
id: TH1.E3.US2
title: "Make agents thin, skills canonical"
agents: [developer, reviewer]
skills: [the-copilot-build-method]
acceptance-criteria:
  - AC1: "Each agent .agent.md file contains only identity, constraints, output format, and skill references"
  - AC2: "Process instructions (step-by-step how-to) live only in skill files"
  - AC3: "No agent file exceeds 50 lines (excluding YAML frontmatter)"
  - AC4: "All reusable knowledge is in skills, not duplicated in agents"
depends-on: [TH1.E3.US1]
---

# TH1.E3.US2 — Make Agents Thin, Skills Canonical

**As a** template maintainer, **I want** agent files to be thin wrappers that load skills, **so that** process changes are made in one place (skills) instead of updating every agent file.

## Acceptance Criteria

- [ ] AC1: Each agent `.agent.md` file contains only identity, constraints, output format, and skill references
- [ ] AC2: Process instructions (step-by-step how-to) live only in skill files
- [ ] AC3: No agent file exceeds 50 lines (excluding YAML frontmatter)
- [ ] AC4: All reusable knowledge is in skills, not duplicated in agents

## BDD Scenarios

### Scenario: Agent file is thin
- **Given** any agent `.agent.md` file
- **When** I count total lines excluding YAML frontmatter
- **Then** it is 50 lines or fewer

### Scenario: Process knowledge lives in skills
- **Given** a process instruction like "how to write BDD scenarios"
- **When** I search across agent files
- **Then** it is not found in any agent file (only in the skill file)

### Scenario: Agent declares skills
- **Given** the developer agent file
- **When** I read its skill references
- **Then** it lists which skills to load (e.g., `bdd-stories`, `code-quality`)

### Scenario: No duplication between agent and skill
- **Given** the reviewer agent's security checklist
- **When** I compare it to the `code-quality` skill's security checklist
- **Then** the agent file does not duplicate the checklist (it references the skill)
