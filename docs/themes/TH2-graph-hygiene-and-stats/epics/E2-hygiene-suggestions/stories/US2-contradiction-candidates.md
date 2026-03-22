---
id: TH2.E2.US2
title: "Detect contradictory facts and propose resolution candidates"
type: standard
priority: high
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "`graph hygiene` detects contradictory fact candidates and reports the conflicting facts in a machine-readable form."
  - AC2: "Each contradiction candidate includes a proposed resolution path rather than passive reporting only."
  - AC3: "If no contradictions exist, the command returns an empty contradiction set without failing."
depends-on: [TH2.E2.US1]
---
# TH2.E2.US2 — Detect contradictory facts and propose resolution candidates

**As an** agent, **I want** contradiction detection and proposed resolution
candidates, **so that** I can keep the shared graph coherent rather than letting
conflicting facts accumulate.

## Acceptance Criteria

- [ ] AC1: `graph hygiene` detects contradictory fact candidates and reports the conflicting facts in a machine-readable form.
- [ ] AC2: Each contradiction candidate includes a proposed resolution path rather than passive reporting only.
- [ ] AC3: If no contradictions exist, the command returns an empty contradiction set without failing.

## BDD Scenarios

### Scenario: Return contradiction candidates with proposed resolutions
- **Given** a repo-local graph contains conflicting facts about the same subject
- **When** an agent runs `graph hygiene`
- **Then** the response includes the contradictory facts and a structured proposed resolution

### Scenario: Return no contradiction candidates for a coherent graph
- **Given** a repo-local graph contains no contradictory facts
- **When** an agent runs `graph hygiene`
- **Then** the response reports an empty contradiction set

### Scenario: Return a structured error when contradiction analysis cannot be completed
- **Given** contradiction analysis cannot evaluate the required graph facts reliably
- **When** an agent runs `graph hygiene`
- **Then** the command returns a structured operational error instead of ambiguous contradiction output
