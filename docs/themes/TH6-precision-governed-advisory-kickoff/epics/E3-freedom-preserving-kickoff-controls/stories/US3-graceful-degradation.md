---
id: TH6.E3.US3
title: "Gracefully degrade kickoff on sparse repos and ambiguous tasks"
type: standard
priority: medium
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "When the repo graph is sparse, workflow start degrades to minimal or abstained kickoff instead of manufacturing confidence."
  - AC2: "When task classification remains ambiguous, workflow start returns an explicit ambiguous-task result rather than over-projecting graph context."
  - AC3: "Graceful degradation still preserves machine-readable next-step guidance for the caller."
depends-on: [TH6.E3.US2]
---
# TH6.E3.US3 — Gracefully degrade kickoff on sparse repos and ambiguous tasks

**As a** maintainer using CGE on imperfect repositories and task prompts, **I
want** workflow start to degrade gracefully, **so that** uncertainty produces
honest guidance instead of noisy context.

## Acceptance Criteria

- [ ] AC1: When the repo graph is sparse, workflow start degrades to minimal or abstained kickoff instead of manufacturing confidence.
- [ ] AC2: When task classification remains ambiguous, workflow start returns an explicit ambiguous-task result rather than over-projecting graph context.
- [ ] AC3: Graceful degradation still preserves machine-readable next-step guidance for the caller.

## BDD Scenarios

### Scenario: Degrade gracefully on a sparse repo graph
- **Given** the repository has only a small amount of graph knowledge
- **When** workflow start evaluates a delegated task
- **Then** the result chooses minimal or abstained kickoff instead of pretending a full kickoff is well-supported

### Scenario: Degrade gracefully on an ambiguous task
- **Given** the delegated task text is too ambiguous to classify confidently
- **When** workflow start evaluates the task
- **Then** the result returns the ambiguous-task state and avoids projecting a noisy kickoff brief

### Scenario: Return next-step guidance after degradation
- **Given** workflow start degrades to minimal or abstained kickoff
- **When** the result is returned
- **Then** the machine-readable output still provides a next-step recommendation for the caller

