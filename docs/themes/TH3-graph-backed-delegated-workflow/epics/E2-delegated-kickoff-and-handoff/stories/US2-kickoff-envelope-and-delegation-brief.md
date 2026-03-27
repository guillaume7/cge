---
id: TH3.E2.US2
title: "Produce compact kickoff envelopes and delegation briefs"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "`graph workflow start` returns a machine-readable kickoff envelope containing task details, graph-state summary, compact context, and a prompt-ready delegation brief."
  - AC2: "Kickoff context respects an explicit token budget and prioritizes high-value graph context over low-signal detail."
  - AC3: "When retrieval finds little or no useful context, the kickoff envelope returns explicit guidance rather than fabricating a rich brief."
depends-on: [TH3.E2.US1]
---
# TH3.E2.US2 — Produce compact kickoff envelopes and delegation briefs

**As an** agent delegating work, **I want** a compact kickoff envelope and delegation
brief, **so that** I can hand a sub-agent useful structured orientation instead of a
large ad hoc summary.

## Acceptance Criteria

- [ ] AC1: `graph workflow start` returns a machine-readable kickoff envelope containing task details, graph-state summary, compact context, and a prompt-ready delegation brief.
- [ ] AC2: Kickoff context respects an explicit token budget and prioritizes high-value graph context over low-signal detail.
- [ ] AC3: When retrieval finds little or no useful context, the kickoff envelope returns explicit guidance rather than fabricating a rich brief.

## BDD Scenarios

### Scenario: Produce a kickoff envelope for a well-covered delegated task
- **Given** the graph contains relevant context for the delegated task
- **When** an agent runs `graph workflow start --task "implement delegated workflow finish" --max-tokens 1200`
- **Then** the command returns a machine-readable kickoff envelope with compact context and a prompt-ready delegation brief

### Scenario: Trim low-value detail to respect the token budget
- **Given** the graph contains more relevant detail than the requested token budget can carry
- **When** an agent runs `graph workflow start --task "implement delegated workflow finish" --max-tokens 600`
- **Then** the command prioritizes the most useful context and returns a bounded kickoff envelope

### Scenario: Return an explicit low-context kickoff result when retrieval is sparse
- **Given** the delegated task has little or no useful context in the graph yet
- **When** an agent runs `graph workflow start --task "investigate a new benchmark scenario"`
- **Then** the command returns a machine-readable kickoff result that explains the low-context state and recommends the next best action
