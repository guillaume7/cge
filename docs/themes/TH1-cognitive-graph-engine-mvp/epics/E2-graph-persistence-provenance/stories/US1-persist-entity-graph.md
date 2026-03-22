---
id: TH1.E2.US1
title: "Persist entities and relationships from native writes"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "`graph write` upserts entity-centric nodes and typed relationships from a validated native payload into Kuzu."
  - AC2: "The persistence layer supports MVP entity kinds such as project metadata, planning artefacts, and codebase entities without requiring a table-per-kind schema."
  - AC3: "Successful writes return a machine-readable summary of created and updated nodes and edges."
depends-on: [TH1.E1.US3]
---
# TH1.E2.US1 — Persist entities and relationships from native writes

**As an** agent, **I want** validated graph payloads to be persisted in Kuzu using the agreed entity-centric model, **so that** shared memory becomes durable and queryable.

## Acceptance Criteria

- [ ] AC1: `graph write` upserts entity-centric nodes and typed relationships from a validated native payload into Kuzu.
- [ ] AC2: The persistence layer supports MVP entity kinds such as project metadata, planning artefacts, and codebase entities without requiring a table-per-kind schema.
- [ ] AC3: Successful writes return a machine-readable summary of created and updated nodes and edges.

## BDD Scenarios

### Scenario: Persist a mixed graph payload
- **Given** a valid native payload containing entity nodes and typed relationships
- **When** the agent runs `graph write`
- **Then** the CLI stores the nodes and relationships in Kuzu and reports the write summary

### Scenario: Upsert an existing entity by identifier
- **Given** the graph already contains an entity with the same stable ID as an incoming payload node
- **When** the agent runs `graph write`
- **Then** the CLI updates the existing entity instead of duplicating it

### Scenario: Reject a relationship that references a missing node
- **Given** a validated payload whose relationship references a node ID that is not present in the payload or the graph
- **When** the agent runs `graph write`
- **Then** the CLI returns a structured persistence error explaining the unresolved relationship endpoint
