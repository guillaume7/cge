---
id: TH1.E2.US2
title: "Store reasoning units and agent sessions with provenance"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "The graph persistence layer stores `ReasoningUnit` and `AgentSession` records with required provenance metadata."
  - AC2: "Persisted reasoning units can be linked to sessions and related project entities through typed relationships."
  - AC3: "Writes fail clearly when provenance metadata is incomplete for reasoning or session records."
depends-on: [TH1.E2.US1]
---
# TH1.E2.US2 — Store reasoning units and agent sessions with provenance

**As an** agent, **I want** reasoning units and session summaries to be persisted with provenance, **so that** later agents can trust where retrieved knowledge came from.

## Acceptance Criteria

- [ ] AC1: The graph persistence layer stores `ReasoningUnit` and `AgentSession` records with required provenance metadata.
- [ ] AC2: Persisted reasoning units can be linked to sessions and related project entities through typed relationships.
- [ ] AC3: Writes fail clearly when provenance metadata is incomplete for reasoning or session records.

## BDD Scenarios

### Scenario: Persist a reasoning unit linked to a session
- **Given** a valid payload containing a `ReasoningUnit`, an `AgentSession`, and a relationship connecting them
- **When** the agent runs `graph write`
- **Then** the graph stores both records and their linkage with provenance intact

### Scenario: Link reasoning to a project artefact
- **Given** a reasoning unit payload that references a prompt, ADR, or code entity already in the graph
- **When** the agent runs `graph write`
- **Then** the resulting graph preserves the typed connection between the reasoning and that artefact

### Scenario: Reject a reasoning unit without session provenance
- **Given** a `ReasoningUnit` payload missing its required `session_id`
- **When** the agent runs `graph write`
- **Then** the CLI returns a structured error indicating that provenance is incomplete
