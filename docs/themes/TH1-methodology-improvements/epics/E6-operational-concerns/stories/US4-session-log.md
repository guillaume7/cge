---
id: TH1.E6.US4
title: "Session log cleanup"
agents: [developer, reviewer]
skills: [backlog-management]
acceptance-criteria:
  - AC1: "Session log is either removed or replaced with a bounded structured format"
  - AC2: "If kept: max N recent entries, older ones pruned"
  - AC3: "Git history is mentioned as the natural audit trail"
depends-on: [TH1.E1.US3]
---

# TH1.E6.US4 — Session Log Cleanup

**As a** methodology user, **I want** the session log to either be bounded or removed, **so that** it doesn't grow without limit and duplicate backlog information.

## Acceptance Criteria

- [ ] AC1: Session log is either removed or replaced with a bounded structured format
- [ ] AC2: If kept: max N recent entries, older ones pruned
- [ ] AC3: Git history is mentioned as the natural audit trail

## BDD Scenarios

### Scenario: Session log removed (option A)
- **Given** the `backlog-management` skill
- **When** I search for session log references
- **Then** there's no mention of session-log.md

### Scenario: Session log bounded (option B)
- **Given** the `backlog-management` skill
- **When** I read the session log section
- **Then** it specifies a maximum number of entries and rotation policy

### Scenario: Git as audit trail
- **Given** the methodology documentation
- **When** I search for audit/history guidance
- **Then** it mentions git commit history as the primary audit trail
