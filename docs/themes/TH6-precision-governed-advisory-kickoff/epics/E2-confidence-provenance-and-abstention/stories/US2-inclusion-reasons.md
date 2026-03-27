---
id: TH6.E2.US2
title: "Return one-line inclusion reasons for kickoff entities"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "Each entity included in a kickoff brief carries a short machine-readable reason describing why it survived family policy and ranking."
  - AC2: "The inclusion reason references concrete task or graph evidence rather than generic labels such as 'high rank'."
  - AC3: "Minimal and full kickoff modes use the same inclusion-reason contract."
depends-on: [TH6.E2.US1]
---
# TH6.E2.US2 — Return one-line inclusion reasons for kickoff entities

**As an** agent receiving a kickoff brief, **I want** each entity to explain why
it was included, **so that** I can calibrate whether the graph context deserves
trust.

## Acceptance Criteria

- [ ] AC1: Each entity included in a kickoff brief carries a short machine-readable reason describing why it survived family policy and ranking.
- [ ] AC2: The inclusion reason references concrete task or graph evidence rather than generic labels such as "high rank".
- [ ] AC3: Minimal and full kickoff modes use the same inclusion-reason contract.

## BDD Scenarios

### Scenario: Explain a code-reference inclusion
- **Given** a kickoff brief includes a code-reference entity
- **When** workflow start returns the brief
- **Then** the entity includes a one-line reason tied to the task or graph evidence that caused its inclusion

### Scenario: Explain a decision-document inclusion
- **Given** a kickoff brief includes an architectural decision entity
- **When** workflow start returns the brief
- **Then** the entity includes a one-line reason describing the decision's relevance to the delegated task

### Scenario: Keep inclusion reasons stable in minimal mode
- **Given** workflow start returns a minimal kickoff brief
- **When** the brief includes entities
- **Then** each included entity still carries the same inclusion-reason contract as full kickoff mode

