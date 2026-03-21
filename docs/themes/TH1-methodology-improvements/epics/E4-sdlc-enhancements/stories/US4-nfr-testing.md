---
id: TH1.E4.US4
title: "Add NFR testing guidance"
agents: [developer, reviewer]
skills: [bdd-stories, the-copilot-build-method]
acceptance-criteria:
  - AC1: "bdd-stories skill documents how to express NFRs as acceptance criteria"
  - AC2: "Developer agent can write NFR-related tests when ACs include performance targets"
  - AC3: "Theme completion ceremony mentions NFR verification when NFRs exist in vision"
depends-on: [TH1.E2.US1]
---

# TH1.E4.US4 — Add NFR Testing Guidance

**As a** product builder, **I want** non-functional requirements (performance, scalability) to be testable, **so that** NFRs from the vision are verified, not just functional behavior.

## Acceptance Criteria

- [ ] AC1: `bdd-stories` skill documents how to express NFRs as acceptance criteria
- [ ] AC2: Developer agent can write NFR-related tests when ACs include performance targets
- [ ] AC3: Theme completion ceremony mentions NFR verification when NFRs exist in vision

## BDD Scenarios

### Scenario: NFR as acceptance criterion
- **Given** the `bdd-stories` skill's AC writing guidelines
- **When** I read the guidance for non-functional criteria
- **Then** it shows examples like "responds within 200ms under 100 concurrent users"

### Scenario: Developer writes NFR tests
- **Given** a story with an AC like "API responds within 200ms"
- **When** the developer implements the story
- **Then** the test suite includes a test that verifies the response time constraint

### Scenario: Theme ceremony checks NFRs
- **Given** the vision docs include NFRs (performance, scalability targets)
- **When** the orchestrator runs the theme completion ceremony
- **Then** it mentions verifying NFRs as part of release readiness
