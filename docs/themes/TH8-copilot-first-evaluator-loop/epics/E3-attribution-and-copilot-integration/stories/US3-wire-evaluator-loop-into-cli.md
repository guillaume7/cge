---
id: TH8.E3.US3
title: "Wire the evaluator loop into graph context and workflow start"
type: standard
priority: high
size: L
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "`graph context` passes candidate retrieval results through the Context Evaluator and Decision Engine before projecting context to the consumer."
  - AC2: "`graph workflow start` passes candidate kickoff context through the evaluator loop before composing the kickoff envelope."
  - AC3: "The decision envelope (including inline attribution summary) is included in the JSON output of both commands."
  - AC4: "Existing consumers that do not parse decision metadata continue to receive backward-compatible output."
  - AC5: "Family-aware kickoff policies (ADR-016, ADR-017) operate upstream of the evaluator loop; the Decision Engine can further narrow but cannot override a family-level suppression."
depends-on:
  - TH8.E3.US2
---
# TH8.E3.US3 — Wire the evaluator loop into graph context and workflow start

**As a** Copilot CLI agent, **I want** `graph context` and
`graph workflow start` to automatically evaluate and decide on retrieved context,
**so that** I receive scored, attributed context with an explicit decision
rather than raw retrieval results.

## Acceptance Criteria

- [ ] AC1: `graph context` passes candidate retrieval results through the Context Evaluator and Decision Engine before projecting context to the consumer.
- [ ] AC2: `graph workflow start` passes candidate kickoff context through the evaluator loop before composing the kickoff envelope.
- [ ] AC3: The decision envelope (including inline attribution summary) is included in the JSON output of both commands.
- [ ] AC4: Existing consumers that do not parse decision metadata continue to receive backward-compatible output.
- [ ] AC5: Family-aware kickoff policies (ADR-016, ADR-017) operate upstream of the evaluator loop; the Decision Engine can further narrow but cannot override a family-level suppression.

## BDD Scenarios

### Scenario: graph context returns evaluated output with decision envelope
- **Given** a graph workspace with indexed entities
- **And** a task description provided to `graph context`
- **When** the command executes
- **Then** the JSON output includes a `decision` object with outcome, scores, and attribution summary
- **And** the context bundle reflects the selected outcome (full, narrowed, or empty)

### Scenario: graph workflow start uses the evaluator loop
- **Given** a graph workspace with workflow assets installed
- **And** a delegated task provided to `graph workflow start`
- **When** the command executes
- **Then** the kickoff envelope includes a `decision` object from the evaluator loop
- **And** the kickoff context reflects the decision outcome

### Scenario: Backward compatibility for consumers ignoring decision metadata
- **Given** an existing consumer that parses only the context bundle from `graph context`
- **When** `graph context` executes with the evaluator loop active
- **Then** the context bundle field remains in the same JSON location as before
- **And** the consumer can extract context without parsing the new decision fields

### Scenario: Family suppression is respected by the evaluator loop
- **Given** a task classified into a family with an abstain-by-default policy
- **When** `graph workflow start` executes
- **Then** the family policy suppresses context before the evaluator loop runs
- **And** the decision envelope records that the family policy pre-suppressed injection

### Scenario: graph explain and graph diff bypass the evaluator loop
- **Given** a graph workspace
- **When** `graph explain` or `graph diff` is executed
- **Then** the command produces output without invoking the Context Evaluator or Decision Engine

### Scenario: Evaluator loop produces attribution on every invocation
- **Given** a `graph context` invocation
- **When** the command completes
- **Then** an attribution record is persisted to `.graph/attribution/`
- **And** the inline summary appears in the stdout JSON

## Notes

- This story is the primary integration point for the Copilot CLI augmentation
  model (ADR-020).
- The evaluator loop composition follows the boundary rules in components.md:
  Context Evaluator → Decision Engine → Attribution Recorder.
- Only `graph context` and `graph workflow start` go through the evaluator loop;
  other commands (explain, diff, stats, hygiene) bypass it.
