---
id: TH1.E1.US2
title: "Add a chainable command surface and repo discovery"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "The CLI exposes `init`, `write`, `query`, `context`, `explain`, and `diff` commands with consistent exit-code behavior."
  - AC2: "Commands resolve the active repo graph from the current working directory without requiring a global graph path."
  - AC3: "Commands that accept task or payload input support stdin in addition to explicit flags or files, enabling shell pipelines."
depends-on: [TH1.E1.US1]
---
# TH1.E1.US2 — Add a chainable command surface and repo discovery

**As an** agent, **I want** a consistent command surface that resolves the active repo graph and accepts piped input, **so that** I can compose graph workflows naturally in shell pipelines.

## Acceptance Criteria

- [ ] AC1: The CLI exposes `init`, `write`, `query`, `context`, `explain`, and `diff` commands with consistent exit-code behavior.
- [ ] AC2: Commands resolve the active repo graph from the current working directory without requiring a global graph path.
- [ ] AC3: Commands that accept task or payload input support stdin in addition to explicit flags or files, enabling shell pipelines.

## BDD Scenarios

### Scenario: Resolve the active graph from the repo
- **Given** a repository that contains a valid `.graph/` workspace
- **When** an agent runs `graph query --task "what depends on auth?"`
- **Then** the command opens the graph for the current repository without requiring extra path configuration

### Scenario: Accept piped input for a chainable command
- **Given** a command that emits structured graph-compatible content to stdout
- **When** the agent pipes it into `graph write` or `graph query`
- **Then** the receiving command reads from stdin and processes the input successfully

### Scenario: Reject graph commands when the workspace is missing
- **Given** a repository where `.graph/` has not been initialized
- **When** an agent runs a graph command other than `graph init`
- **Then** the CLI returns a clear error that the repo graph must be initialized first
