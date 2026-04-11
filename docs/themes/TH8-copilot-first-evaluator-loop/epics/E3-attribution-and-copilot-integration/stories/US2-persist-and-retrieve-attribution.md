---
id: TH8.E3.US2
title: "Persist and retrieve attribution records in the local workspace"
type: standard
priority: medium
size: M
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "Attribution records are persisted to `.graph/attribution/` as individual JSON files named by a unique attribution ID."
  - AC2: "The Attribution Recorder exposes a `loadAttribution(attributionId)` method that returns a single record."
  - AC3: "The Attribution Recorder exposes a `listAttributions(filter)` method that lists records filtered by time range, outcome type, or task context."
  - AC4: "Persisted records follow the same workspace lifecycle as lab artifacts and can be pruned."
depends-on:
  - TH8.E3.US1
---
# TH8.E3.US2 — Persist and retrieve attribution records in the local workspace

**As a** lab experiment analyst, **I want** attribution records persisted to the
local workspace and retrievable by ID or filter, **so that** I can aggregate
decision evidence across runs.

## Acceptance Criteria

- [ ] AC1: Attribution records are persisted to `.graph/attribution/` as individual JSON files named by a unique attribution ID.
- [ ] AC2: The Attribution Recorder exposes a `loadAttribution(attributionId)` method that returns a single record.
- [ ] AC3: The Attribution Recorder exposes a `listAttributions(filter)` method that lists records filtered by time range, outcome type, or task context.
- [ ] AC4: Persisted records follow the same workspace lifecycle as lab artifacts and can be pruned.

## BDD Scenarios

### Scenario: Persist an attribution record to the workspace
- **Given** the Attribution Recorder generates a record for a `minimal` decision
- **When** the record is persisted
- **Then** a JSON file appears under `.graph/attribution/` with the record's attribution ID as the filename
- **And** the file content matches the generated record

### Scenario: Load a persisted record by ID
- **Given** an attribution record with ID `attr-20260401-001` has been persisted
- **When** `loadAttribution("attr-20260401-001")` is called
- **Then** the returned record matches the original persisted content

### Scenario: List records filtered by outcome type
- **Given** five persisted attribution records: two `continue`, two `abstain`, one `minimal`
- **When** `listAttributions` is called with filter `outcome=abstain`
- **Then** exactly two records are returned

### Scenario: List records filtered by time range
- **Given** attribution records from three different dates
- **When** `listAttributions` is called with a time-range filter covering only one date
- **Then** only records from that date are returned

### Scenario: Handle missing attribution ID gracefully
- **Given** no attribution record with ID `attr-nonexistent` exists
- **When** `loadAttribution("attr-nonexistent")` is called
- **Then** the method returns a not-found error

## Notes

- Storage location `.graph/attribution/` matches the component specification
  in components.md (Attribution Recorder §20).
- Old records can be pruned using the same lifecycle rules as lab artifacts
  (ADR-021 §risk mitigation).
