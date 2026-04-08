---
id: TH8.E4.US2
title: "Down-rank low-confidence graph entities during retrieval"
type: standard
priority: medium
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "The Context Evaluator can flag graph entities as low-confidence during scoring."
  - AC2: "The Retrieval Engine deprioritizes low-confidence entities so they rank lower in candidate lists."
  - AC3: "Down-ranked entities remain in the graph store; they are not deleted or modified."
  - AC4: "Down-ranking is visible in the attribution record so lab analysis can track its effect."
depends-on:
  - TH8.E4.US1
---
# TH8.E4.US2 — Down-rank low-confidence graph entities during retrieval

**As a** consuming agent, **I want** stale or low-confidence graph entities
deprioritized during retrieval, **so that** the context I receive reflects
current, trustworthy knowledge rather than graph drift.

## Acceptance Criteria

- [ ] AC1: The Context Evaluator can flag graph entities as low-confidence during scoring.
- [ ] AC2: The Retrieval Engine deprioritizes low-confidence entities so they rank lower in candidate lists.
- [ ] AC3: Down-ranked entities remain in the graph store; they are not deleted or modified.
- [ ] AC4: Down-ranking is visible in the attribution record so lab analysis can track its effect.

## BDD Scenarios

### Scenario: Deprioritize a stale entity
- **Given** a graph entity with outdated provenance that scores low on consistency
- **And** a newer entity covering the same topic that scores higher
- **When** the Retrieval Engine returns ranked candidates
- **Then** the stale entity appears lower in the ranking than the newer entity
- **And** the stale entity is still present in the results (not removed)

### Scenario: Down-ranking does not delete graph state
- **Given** a low-confidence entity flagged by the evaluator
- **When** retrieval completes
- **Then** the entity remains unchanged in the Kuzu store
- **And** only its ranking position is affected

### Scenario: Attribution record shows down-ranking
- **Given** a retrieval pass where two entities are down-ranked
- **When** the attribution record is generated
- **Then** the per-candidate fates include a `down-ranked` indicator for those entities
- **And** the reason references the low consistency or staleness signal

### Scenario: High-confidence entities are unaffected
- **Given** a graph entity with recent provenance and high consistency scores
- **When** the Retrieval Engine returns ranked candidates
- **Then** the entity's ranking is based solely on its relevance and usefulness scores
- **And** no down-ranking indicator appears in its candidate fate

### Scenario: All entities are low-confidence
- **Given** a retrieval pass where every candidate entity is flagged as low-confidence
- **When** the evaluator and Decision Engine process the bundle
- **Then** the Decision Engine may select `abstain` or `minimal` based on the bundle confidence
- **And** the attribution record explains that all candidates were low-confidence

## Notes

- Down-ranking is preferred over deletion because it is reversible and preserves
  the graph as a record (ADR-022 §3).
- Existing hygiene workflows (ADR-007) can still prune entities that the
  evaluator persistently flags as low-confidence.
