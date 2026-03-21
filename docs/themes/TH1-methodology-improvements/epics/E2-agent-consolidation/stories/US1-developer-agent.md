---
id: TH1.E2.US1
title: "Create developer agent from implementer + tester"
agents: [developer, reviewer]
skills: [the-copilot-build-method, bdd-stories]
acceptance-criteria:
  - AC1: "A new developer.agent.md exists that combines implement + test responsibilities"
  - AC2: "developer agent implements code AND writes tests in a single session"
  - AC3: "implementer.agent.md and tester.agent.md are moved to .github/agents/archive/"
  - AC4: "orchestrator.agent.md delegates to @developer instead of @implementer + @tester"
  - AC5: "README.md and copilot-instructions.md agent tables reflect the new squad"
depends-on: [TH1.E1.US1]
---

# TH1.E2.US1 — Create Developer Agent from Implementer + Tester

**As a** methodology user, **I want** a single developer agent that implements and tests each story, **so that** context is preserved between implementation and testing, reducing delegation overhead.

## Acceptance Criteria

- [ ] AC1: A new `developer.agent.md` exists that combines implement + test responsibilities
- [ ] AC2: Developer agent implements code AND writes tests in a single session
- [ ] AC3: `implementer.agent.md` and `tester.agent.md` are moved to `.github/agents/archive/`
- [ ] AC4: `orchestrator.agent.md` delegates to `@developer` instead of `@implementer` + `@tester`
- [ ] AC5: `README.md` and `copilot-instructions.md` agent tables reflect the new squad

## BDD Scenarios

### Scenario: Developer agent file exists with merged instructions
- **Given** the `.github/agents/` directory
- **When** I list the agent files
- **Then** `developer.agent.md` exists with instructions covering both implementation and testing

### Scenario: Orchestrator story loop uses developer
- **Given** the orchestrator's core loop for processing a story
- **When** I read the delegation steps
- **Then** it delegates to `@developer` (one call), not `@implementer` then `@tester` (two calls)

### Scenario: Old agents archived
- **Given** the `.github/agents/archive/` directory
- **When** I list the archived files
- **Then** `implementer.agent.md` and `tester.agent.md` are present

### Scenario: Developer agent output includes both implementation and test reports
- **Given** the developer agent's output format
- **When** I read the required report structure
- **Then** it includes both "Files Changed" (implementation) and "Test Results" sections
