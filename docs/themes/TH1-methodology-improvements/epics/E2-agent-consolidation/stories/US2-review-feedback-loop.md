---
id: TH1.E2.US2
title: "Add review feedback loop to orchestrator"
agents: [developer, reviewer]
skills: [the-copilot-build-method]
acceptance-criteria:
  - AC1: "Orchestrator loop handles REQUEST_CHANGES by delegating back to developer with feedback"
  - AC2: "Rework cycle runs max 2 iterations before escalating to user"
  - AC3: "Troubleshooter is reserved for runtime/build failures only, not review feedback"
depends-on: [TH1.E2.US1]
---

# TH1.E2.US2 — Add Review Feedback Loop to Orchestrator

**As a** methodology user, **I want** the orchestrator to handle review change requests by re-delegating to the developer, **so that** review feedback gets addressed instead of being routed to the troubleshooter.

## Acceptance Criteria

- [ ] AC1: Orchestrator loop handles `REQUEST_CHANGES` by delegating back to developer with feedback
- [ ] AC2: Rework cycle runs max 2 iterations before escalating to user
- [ ] AC3: Troubleshooter is reserved for runtime/build failures only, not review feedback

## BDD Scenarios

### Scenario: Reviewer requests changes
- **Given** the reviewer returns `REQUEST_CHANGES` with a list of issues
- **When** the orchestrator processes the review result
- **Then** it delegates back to `@developer` with the review feedback (not to `@troubleshooter`)

### Scenario: Rework iteration limit
- **Given** the developer has already reworked a story twice
- **When** the reviewer still returns `REQUEST_CHANGES` on the third review
- **Then** the orchestrator escalates to the user instead of looping again

### Scenario: Troubleshooter reserved for failures
- **Given** the orchestrator's error handling section
- **When** I read when the troubleshooter is invoked
- **Then** it's only for `failed` stories (build errors, test failures), not review feedback

### Scenario: Successful rework
- **Given** the reviewer returns `REQUEST_CHANGES` on first review
- **When** the developer addresses the feedback and the reviewer approves on second review
- **Then** the story proceeds to `done` status
