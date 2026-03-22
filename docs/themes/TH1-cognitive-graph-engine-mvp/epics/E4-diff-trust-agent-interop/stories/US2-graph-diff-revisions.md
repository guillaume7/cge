---
id: TH1.E4.US2
title: "Compare graph revisions with diff"
type: standard
priority: medium
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "`graph diff` compares two revision anchors and reports added, updated, removed, and retagged entities or relationships."
  - AC2: "Diff results are returned in the native machine-readable response format with revision metadata."
  - AC3: "`graph diff` fails clearly when a requested revision anchor does not exist."
depends-on: [TH1.E2.US3]
---
# TH1.E4.US2 — Compare graph revisions with diff

**As an** agent, **I want** `graph diff` to compare graph revisions, **so that** I can inspect what changed after graph writes, cleanup, or refactoring.

## Acceptance Criteria

- [ ] AC1: `graph diff` compares two revision anchors and reports added, updated, removed, and retagged entities or relationships.
- [ ] AC2: Diff results are returned in the native machine-readable response format with revision metadata.
- [ ] AC3: `graph diff` fails clearly when a requested revision anchor does not exist.

## BDD Scenarios

### Scenario: Diff two revisions after a graph update
- **Given** the graph has two valid revision anchors around a write or cleanup operation
- **When** the agent runs `graph diff --from <older> --to <newer>`
- **Then** the CLI returns the entities and relationships that were added, changed, or removed

### Scenario: Report tag or kind changes
- **Given** an entity changes kind-relevant metadata or normalized tags between revisions
- **When** the agent runs `graph diff`
- **Then** the diff output highlights that metadata change as part of the result set

### Scenario: Reject an unknown revision anchor
- **Given** the agent requests a revision ID that does not exist
- **When** the agent runs `graph diff`
- **Then** the CLI returns a structured error that identifies the missing revision anchor
