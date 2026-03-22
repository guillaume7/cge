# TH1 — Cognitive Graph Engine MVP

## Theme Goal

Turn VP1 into an implementable MVP backlog for a repo-scoped, local, agent-only
graph CLI that improves continuity across sessions while reducing token overhead
through compact, trustworthy context retrieval.

## Scope

This theme covers:

- repository-local graph bootstrap
- chainable CLI surface for `init`, `write`, `query`, `context`, `explain`, and `diff`
- native structured graph payloads for stdin, stdout, and files
- Kuzu-backed graph persistence with provenance
- hybrid retrieval using graph structure plus local text relevance
- context projection, explanation, and diff support

## Out of Scope

- human-oriented visualization
- remote synchronization
- hosted services
- automatic repository ingestion
- immutable historical audit trails

## Epics

1. **TH1.E1 — Workspace and CLI Foundation**
   Establish the repo-local graph workspace, command surface, and native payload
   validation needed by every downstream capability.

2. **TH1.E2 — Graph Persistence and Provenance**
   Implement Kuzu-backed graph writes, provenance rules, and revision anchors so
   the graph becomes a durable shared memory substrate.

3. **TH1.E3 — Hybrid Retrieval and Context**
   Build `graph query`, `graph context`, and `graph explain` on top of hybrid
   structural plus text-relevance retrieval.

4. **TH1.E4 — Diff, Trust, and Agent Interoperability**
   Complete the MVP with consistent machine-readable command contracts, graph
   diffing, and end-to-end chainable agent workflows.

## Dependency Flow

```text
E1 → E2 → E3 → E4
```

## Success Signal

An agent can initialize a repo graph, pipe native graph payloads into
`graph write`, retrieve compact trustworthy context for a task, understand why
it was returned, diff graph changes over time, and chain these commands with
other local tools without glue code.
