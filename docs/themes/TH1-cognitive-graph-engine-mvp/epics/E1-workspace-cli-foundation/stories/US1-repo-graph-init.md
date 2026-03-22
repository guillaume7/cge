---
id: TH1.E1.US1
title: "Initialize a repo-scoped graph workspace"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "`graph init` creates a repo-local `.graph/` workspace containing config, Kuzu storage, and index directories."
  - AC2: "`graph init` records repository identity and schema version metadata so later commands can open the same graph reliably."
  - AC3: "`graph init` is safe to re-run and reports the existing workspace instead of corrupting it."
depends-on: []
---
# TH1.E1.US1 — Initialize a repo-scoped graph workspace

**As an** agent, **I want** to initialize a repo-local graph workspace, **so that** every later graph command has a deterministic shared memory home inside the repository.

## Acceptance Criteria

- [ ] AC1: `graph init` creates a repo-local `.graph/` workspace containing config, Kuzu storage, and index directories.
- [ ] AC2: `graph init` records repository identity and schema version metadata so later commands can open the same graph reliably.
- [ ] AC3: `graph init` is safe to re-run and reports the existing workspace instead of corrupting it.

## BDD Scenarios

### Scenario: Initialize a new repository graph
- **Given** a repository without a `.graph/` workspace
- **When** the agent runs `graph init`
- **Then** the CLI creates the expected workspace layout and initialization metadata

### Scenario: Re-run initialization on an existing graph
- **Given** a repository that already contains a valid `.graph/` workspace
- **When** the agent runs `graph init` again
- **Then** the CLI keeps the existing workspace intact and returns a successful idempotent result

### Scenario: Fail outside a repository root
- **Given** the command is run from a directory that is not inside a repository
- **When** the agent runs `graph init`
- **Then** the CLI returns a clear error explaining that repo-scoped initialization could not determine a repository root
