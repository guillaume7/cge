---
id: TH8.E1.US1
title: "Score candidate context bundles on relevance, consistency, and usefulness"
type: standard
priority: high
size: L
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "The Context Evaluator accepts a list of candidate context items and a task description, and returns a per-candidate score on three dimensions: relevance, consistency, and usefulness."
  - AC2: "Relevance scoring uses text-overlap and structural-neighborhood heuristics to measure how closely a candidate relates to the current task."
  - AC3: "Consistency scoring checks each candidate against other candidates and existing graph state for contradictions or staleness."
  - AC4: "Usefulness scoring estimates whether including the candidate is likely to help task completion rather than add noise."
  - AC5: "Each dimension score is a normalized float between 0.0 and 1.0."
depends-on: []
---
# TH8.E1.US1 — Score candidate context bundles on relevance, consistency, and usefulness

**As a** consuming agent retrieving graph-backed context, **I want** each
candidate context item scored on relevance, consistency, and usefulness, **so
that** I can distinguish helpful context from noise before trusting it.

## Acceptance Criteria

- [ ] AC1: The Context Evaluator accepts a list of candidate context items and a task description, and returns a per-candidate score on three dimensions: relevance, consistency, and usefulness.
- [ ] AC2: Relevance scoring uses text-overlap and structural-neighborhood heuristics to measure how closely a candidate relates to the current task.
- [ ] AC3: Consistency scoring checks each candidate against other candidates and existing graph state for contradictions or staleness.
- [ ] AC4: Usefulness scoring estimates whether including the candidate is likely to help task completion rather than add noise.
- [ ] AC5: Each dimension score is a normalized float between 0.0 and 1.0.

## BDD Scenarios

### Scenario: Score a highly relevant candidate
- **Given** a task description about implementing a new CLI command
- **And** a candidate context item that describes the existing CLI command structure
- **When** the Context Evaluator scores the candidate
- **Then** the relevance score is above 0.7
- **And** all three dimension scores are returned as floats between 0.0 and 1.0

### Scenario: Score an irrelevant candidate
- **Given** a task description about fixing a retrieval bug
- **And** a candidate context item about an unrelated reporting feature
- **When** the Context Evaluator scores the candidate
- **Then** the relevance score is below 0.3

### Scenario: Detect inconsistency with existing graph state
- **Given** a candidate context item that contradicts an entity already present in the graph store
- **When** the Context Evaluator scores the candidate for consistency
- **Then** the consistency score is below 0.4
- **And** the score metadata indicates the conflicting graph entity

### Scenario: Score multiple candidates in a single evaluation call
- **Given** a task description and five candidate context items with varying relevance
- **When** the Context Evaluator scores the candidate bundle
- **Then** each candidate receives independent scores on all three dimensions
- **And** the returned list preserves the input candidate order

### Scenario: Handle an empty candidate list
- **Given** a task description and an empty list of candidate context items
- **When** the Context Evaluator is invoked
- **Then** the evaluator returns an empty scored list without error

## Notes

- The Context Evaluator is an in-process Go component using local heuristics
  only — no LLM calls (ADR-018).
- Scoring dimensions match those defined in ADR-018: relevance, consistency,
  and likely usefulness.
