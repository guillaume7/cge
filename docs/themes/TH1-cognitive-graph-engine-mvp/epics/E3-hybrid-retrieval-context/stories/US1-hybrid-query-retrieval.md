---
id: TH1.E3.US1
title: "Build hybrid query retrieval"
type: standard
priority: high
size: L
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "`graph query` combines structural graph traversal with a local text-relevance index to retrieve task-relevant candidates."
  - AC2: "The retrieval pipeline ranks and merges candidates into a machine-readable result set with provenance references."
  - AC3: "The local text-relevance implementation works fully offline and can be rebuilt from persisted graph data."
depends-on: [TH1.E2.US2]
---
# TH1.E3.US1 — Build hybrid query retrieval

**As an** agent, **I want** `graph query` to combine graph structure and local text relevance, **so that** I can find useful knowledge even when task phrasing does not exactly match stored entity names.

## Acceptance Criteria

- [ ] AC1: `graph query` combines structural graph traversal with a local text-relevance index to retrieve task-relevant candidates.
- [ ] AC2: The retrieval pipeline ranks and merges candidates into a machine-readable result set with provenance references.
- [ ] AC3: The local text-relevance implementation works fully offline and can be rebuilt from persisted graph data.

## BDD Scenarios

### Scenario: Retrieve candidates using graph structure and text relevance
- **Given** the graph contains related entities, tags, and textual summaries about an authentication subsystem
- **When** the agent runs `graph query --task "what depends on auth?"`
- **Then** the CLI returns a ranked result set that reflects both direct graph relationships and relevant text matches

### Scenario: Return relevant results when wording differs from stored titles
- **Given** stored entities describe "authentication" while the task uses the shorter term "auth"
- **When** the agent runs `graph query`
- **Then** the local text-relevance ranking still surfaces the relevant graph entities

### Scenario: Fail clearly when the text index is unavailable
- **Given** the graph store exists but the text index is missing or corrupted
- **When** the agent runs `graph query`
- **Then** the CLI returns a clear error or rebuild instruction instead of silently dropping the text-relevance portion of retrieval
