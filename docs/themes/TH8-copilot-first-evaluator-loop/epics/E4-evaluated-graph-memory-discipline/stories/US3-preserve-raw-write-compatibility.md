---
id: TH8.E4.US3
title: "Preserve raw graph write backward compatibility"
type: standard
priority: medium
size: S
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "Direct `graph write` commands continue to persist entities without mandatory evaluation."
  - AC2: "Raw writes do not produce attribution records or decision envelopes."
  - AC3: "The coexistence of raw writes and evaluated writes is documented in CLI help and workspace README."
depends-on:
  - TH8.E4.US2
---
# TH8.E4.US3 — Preserve raw graph write backward compatibility

**As an** agent using the existing `graph write` command, **I want** raw writes
to continue working without mandatory evaluation, **so that** existing
integrations are not broken by the VP8 evaluator loop.

## Acceptance Criteria

- [ ] AC1: Direct `graph write` commands continue to persist entities without mandatory evaluation.
- [ ] AC2: Raw writes do not produce attribution records or decision envelopes.
- [ ] AC3: The coexistence of raw writes and evaluated writes is documented in CLI help and workspace README.

## BDD Scenarios

### Scenario: Raw write persists without evaluation
- **Given** a valid entity payload
- **When** `graph write` is invoked with the payload
- **Then** the entity is persisted to the Kuzu store immediately
- **And** no Context Evaluator or Decision Engine is invoked

### Scenario: Raw write produces no attribution record
- **Given** a `graph write` invocation
- **When** the write completes
- **Then** no file is created under `.graph/attribution/`

### Scenario: Evaluated write and raw write coexist
- **Given** an entity written via `graph write` (raw) and another written via `workflow finish` (evaluated)
- **When** both entities are queried from the graph
- **Then** both entities are present in the store
- **And** the workflow-mediated entity has an associated attribution record while the raw entity does not

## Notes

- Raw writes are an explicit opt-out from evaluation discipline, not the
  recommended default for VP8 workflow paths (ADR-022 §4).
- This story is intentionally small — it primarily validates that the new
  evaluated write path does not regress existing behavior.
