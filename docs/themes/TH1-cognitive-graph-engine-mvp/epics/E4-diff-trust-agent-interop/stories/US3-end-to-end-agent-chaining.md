---
id: TH1.E4.US3
title: "Verify end-to-end chainable agent workflows"
type: standard
priority: medium
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "End-to-end workflows such as `copilot \"design auth service\" | graph write` and `copilot \"what depends on auth?\" | graph query` are supported by the native graph contract."
  - AC2: "A context retrieval workflow can emit structured context that is consumable by a downstream agent command without ad hoc transformation."
  - AC3: "Workflow validation covers at least one successful chained flow and one structured error flow."
depends-on: [TH1.E3.US3, TH1.E4.US1, TH1.E4.US2]
---
# TH1.E4.US3 — Verify end-to-end chainable agent workflows

**As an** agent, **I want** the graph CLI to work in realistic chained workflows, **so that** trained custom agents can speak and consume native graph payloads directly during multi-step tasks.

## Acceptance Criteria

- [ ] AC1: End-to-end workflows such as `copilot "design auth service" | graph write` and `copilot "what depends on auth?" | graph query` are supported by the native graph contract.
- [ ] AC2: A context retrieval workflow can emit structured context that is consumable by a downstream agent command without ad hoc transformation.
- [ ] AC3: Workflow validation covers at least one successful chained flow and one structured error flow.

## BDD Scenarios

### Scenario: Chain agent output into graph write
- **Given** a custom agent emits valid native graph payload content to stdout
- **When** the output is piped into `graph write`
- **Then** the CLI persists the payload without requiring an intermediate file

### Scenario: Chain graph context into a downstream agent step
- **Given** `graph context` returns a structured native context payload for a task
- **When** that output is piped into a downstream agent command
- **Then** the downstream tool can consume the payload without ad hoc translation

### Scenario: Preserve structured errors in a chained workflow
- **Given** a piped workflow provides invalid native graph payload content
- **When** the receiving graph command processes the input
- **Then** it returns a machine-readable error response that the upstream or downstream tool can inspect programmatically
