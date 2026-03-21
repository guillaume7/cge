---
id: TH1.E2.US3
title: "Remove refactorer agent"
agents: [developer, reviewer]
skills: [the-copilot-build-method, code-quality]
acceptance-criteria:
  - AC1: "refactorer.agent.md is moved to .github/agents/archive/"
  - AC2: "Developer agent instructions say to write clean code from the start"
  - AC3: "Epic completion ceremony replaces 'refactor pass' with 'lightweight code quality check by reviewer'"
  - AC4: "refactor.prompt.md is removed or updated to invoke reviewer for code quality"
depends-on: [TH1.E2.US1]
---

# TH1.E2.US3 — Remove Refactorer Agent

**As a** methodology user, **I want** the refactorer removed and clean code written from the start, **so that** we avoid the wasteful cycle of intentionally writing minimal code then paying to clean it up.

## Acceptance Criteria

- [ ] AC1: `refactorer.agent.md` is moved to `.github/agents/archive/`
- [ ] AC2: Developer agent instructions say to write clean code from the start
- [ ] AC3: Epic completion ceremony replaces "refactor pass" with "lightweight code quality check by reviewer"
- [ ] AC4: `refactor.prompt.md` is removed or updated to invoke reviewer for code quality

## BDD Scenarios

### Scenario: Refactorer archived
- **Given** the `.github/agents/archive/` directory
- **When** I list the archived files
- **Then** `refactorer.agent.md` is present

### Scenario: Developer writes clean code
- **Given** the developer agent's constraints section
- **When** I read the coding guidelines
- **Then** it says "write clean, well-structured code" (not "keep implementations minimal")

### Scenario: Epic ceremony has lightweight quality check
- **Given** the orchestrator's epic completion steps
- **When** I read the ceremony workflow
- **Then** it invokes `@reviewer` for a code quality check, not `@refactorer` for a full refactor pass

### Scenario: Anti-patterns updated
- **Given** the anti-patterns list in methodology docs
- **When** I search for refactor references
- **Then** "Never skip the refactor at epic end" is replaced with guidance about code quality review
