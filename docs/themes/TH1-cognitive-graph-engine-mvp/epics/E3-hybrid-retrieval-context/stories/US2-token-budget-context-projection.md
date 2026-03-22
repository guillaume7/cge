---
id: TH1.E3.US2
title: "Project compact task context"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "`graph context` projects ranked retrieval results into a compact machine-readable context envelope constrained by a caller-supplied token budget."
  - AC2: "The context output preserves critical relationships, summaries, and provenance while excluding low-value graph detail."
  - AC3: "When the available budget is too small to include everything, higher-value context is retained ahead of less relevant detail."
depends-on: [TH1.E3.US1]
---
# TH1.E3.US2 — Project compact task context

**As an** agent, **I want** `graph context` to return only the most useful information that fits my token budget, **so that** I can continue work fluidly without paying to reload excess graph state.

## Acceptance Criteria

- [ ] AC1: `graph context` projects ranked retrieval results into a compact machine-readable context envelope constrained by a caller-supplied token budget.
- [ ] AC2: The context output preserves critical relationships, summaries, and provenance while excluding low-value graph detail.
- [ ] AC3: When the available budget is too small to include everything, higher-value context is retained ahead of less relevant detail.

## BDD Scenarios

### Scenario: Return context that fits the requested budget
- **Given** a task that matches more graph content than can fit into the target context window
- **When** the agent runs `graph context --max-tokens 1200`
- **Then** the CLI returns a context envelope that stays within the requested budget

### Scenario: Preserve the highest-value context when trimming
- **Given** a retrieval result set containing both central entities and peripheral neighbors
- **When** the agent requests a small context budget
- **Then** the CLI keeps the most relevant entities, relationships, and provenance before dropping lower-value detail

### Scenario: Reject an invalid token budget
- **Given** an agent requests `graph context` with a zero or negative token budget
- **When** the command is executed
- **Then** the CLI returns a structured validation error describing the invalid budget
