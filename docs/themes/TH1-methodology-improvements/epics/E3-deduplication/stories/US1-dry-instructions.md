---
id: TH1.E3.US1
title: "DRY copilot-instructions.md"
agents: [developer, reviewer]
skills: [the-copilot-build-method]
acceptance-criteria:
  - AC1: "copilot-instructions.md is a concise entry point, not a content dump"
  - AC2: "Naming conventions, status values, DoD, story format are referenced from skills (not repeated)"
  - AC3: "copilot-instructions.md references each skill by name for its canonical topic"
  - AC4: "README.md is simplified to human-facing content only (no agent instructions)"
depends-on: [TH1.E2.US1]
---

# TH1.E3.US1 — DRY copilot-instructions.md

**As a** template maintainer, **I want** copilot-instructions.md to reference skills instead of duplicating their content, **so that** a change to any convention requires updating exactly one file.

## Acceptance Criteria

- [ ] AC1: `copilot-instructions.md` is a concise entry point, not a content dump
- [ ] AC2: Naming conventions, status values, DoD, story format are referenced from skills (not repeated)
- [ ] AC3: `copilot-instructions.md` references each skill by name for its canonical topic
- [ ] AC4: `README.md` is simplified to human-facing content only (no agent instructions)

## BDD Scenarios

### Scenario: copilot-instructions.md is concise
- **Given** the file `copilot-instructions.md`
- **When** I count the number of lines
- **Then** it is significantly shorter than the current version (no duplicated tables or templates)

### Scenario: Skills are the canonical source
- **Given** a convention like "naming conventions"
- **When** I search for it across all files
- **Then** the detailed definition appears only in the relevant skill file, not in `copilot-instructions.md`

### Scenario: Entry point references skills
- **Given** `copilot-instructions.md`
- **When** I read it
- **Then** each major topic says "See skill: `<skill-name>` for details"

### Scenario: README is human-only
- **Given** the root `README.md`
- **When** I read it
- **Then** it describes the project for human users and does not repeat agent-facing instructions
