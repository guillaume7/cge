---
id: TH2.E1.US2
title: "Compute cognitive health indicators from the current graph snapshot"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "`graph stats` includes duplication rate, orphan rate, contradictory fact count, and density/clustering indicators derived from the current graph snapshot."
  - AC2: "Indicator values remain well-formed and deterministic for empty, sparse, and moderately connected graphs."
  - AC3: "Indicator output is machine-readable and scoped to a snapshot rather than historical trend data."
depends-on: [TH2.E1.US1]
---
# TH2.E1.US2 — Compute cognitive health indicators from the current graph snapshot

**As an** agent, **I want** graph-health indicators in `graph stats`, **so that**
I can judge whether the graph is structured enough to trust or chaotic enough to
clean first.

## Acceptance Criteria

- [ ] AC1: `graph stats` includes duplication rate, orphan rate, contradictory fact count, and density/clustering indicators derived from the current graph snapshot.
- [ ] AC2: Indicator values remain well-formed and deterministic for empty, sparse, and moderately connected graphs.
- [ ] AC3: Indicator output is machine-readable and scoped to a snapshot rather than historical trend data.

## BDD Scenarios

### Scenario: Return agreed health indicators for a populated graph
- **Given** a repo-local graph with duplicate-like structure, orphan candidates, and meaningful connectivity
- **When** an agent runs `graph stats`
- **Then** the response includes the agreed health indicators alongside the raw counts

### Scenario: Return safe indicator values for a nearly empty graph
- **Given** a repo-local graph with minimal or no content
- **When** an agent runs `graph stats`
- **Then** the indicator values remain valid machine-readable numbers instead of undefined or malformed output

### Scenario: Return a structured error when the graph snapshot cannot be analyzed
- **Given** the graph store cannot be read consistently for analysis
- **When** an agent runs `graph stats`
- **Then** the command returns a structured operational error rather than misleading indicator values
