# VP2 — Graph Hygiene and Stats

## Vision Summary

Build the second product phase of the Cognitive Graph Engine around one core
idea: a graph that compounds knowledge must also stay **clean, explainable, and
measurable**.

VP1 proved that agents can share a local graph memory, query it, compact it
into context, explain retrieval, and diff revisions. VP2 should make that graph
operationally healthier by helping agents detect and repair graph mess, while
also exposing the graph's structural health through snapshot metrics and
cognitive indicators.

This phase remains local, offline, repo-scoped, and agent-first.

## Product Intent

Once agents begin using a shared graph over time, the graph itself becomes a
maintenance problem:

- duplicate nodes accumulate
- orphaned nodes remain after edits or refactors
- contradictions appear between facts written across sessions
- the graph drifts from structured memory toward noisy memory

If this drift is not controlled, retrieval quality degrades and the graph stops
being a trustworthy substrate. VP2 exists to keep the graph usable by giving
agents explicit hygiene tools and graph-health visibility.

## Primary Outcome

Turn CGE from a memory primitive into a **maintainable memory system**.

VP2 should help an agent answer two new classes of question:

1. **How healthy is this graph right now?**
2. **What should I clean up, and can I safely apply that cleanup?**

## Primary Users

The sole users remain AI agents.

VP2 is for:

- agents that inherit an existing graph and need to assess its quality
- agents that detect duplicates, contradictions, or stale structure during work
- orchestrated flows that want a safer graph before using it for retrieval

Humans may inspect outputs, but the product remains optimized for machine use.

## Core Jobs To Be Done

1. Let an agent inspect graph health through a stable snapshot of graph stats.
2. Let an agent detect duplicate-near-identical nodes that should be
   consolidated.
3. Let an agent detect orphan nodes that no longer contribute useful structure.
4. Let an agent detect contradictory facts and support resolution workflows.
5. Let an agent review hygiene suggestions before changing the graph.
6. Let an agent explicitly apply approved hygiene actions.

## Product Principles

- **Suggest first, apply explicitly**: graph changes should default to
  recommendation mode; mutation requires intentional opt-in such as `--apply`.
- **Safe cleanup over aggressive cleanup**: hygiene should prefer trustworthy
  results over broad destructive behavior.
- **Operational visibility matters**: agents need a compact but meaningful view
  of graph structure and disorder.
- **Cognitive health is measurable**: graph quality should not be treated as
  intuition only; metrics should surface it.
- **Agent-native trust remains central**: suggested fixes and stats should be
  explainable and machine-consumable.
- **Repo-local continuity remains intact**: VP2 must extend the existing local
  memory model, not replace it with hosted infrastructure.

## VP2 Scope

### Included

- A `graph stats` command for snapshot metrics and cognitive indicators
- A `graph hygiene` command for graph cleanup suggestions and explicit apply
  workflows
- Detection of orphan nodes
- Detection of near-identical duplicate nodes
- Detection of contradictory facts
- Resolution flows for contradictions
- Structured outputs for suggestions, stats, and applied fixes
- Hygiene-safe changes that integrate with existing graph revisions and diff
  workflows

### Excluded

- Background autonomous cleanup daemons
- Continuous automatic mutation without explicit approval
- Human visualization dashboards
- Multi-repo or cross-machine graph hygiene in VP2
- Trend analytics as a first-class requirement in VP2

## VP2 Command Surface

VP2 should add:

- `graph stats`
- `graph hygiene`

These commands extend, rather than replace, the VP1 surface.

## Command Intent

### `graph stats`

Return a point-in-time structural snapshot of the graph.

It should expose:

- node count
- relationship count
- duplication rate
- orphan rate
- contradictory fact count
- density / clustering indicators

The output should be machine-readable and suitable for agent decision making.

### `graph hygiene`

Inspect graph quality issues and suggest cleanup actions.

By default, it should run in suggest-only mode and return:

- candidate duplicate groups
- candidate orphan nodes
- candidate contradictions
- proposed resolution or cleanup actions

When explicitly invoked with an apply mode such as `--apply`, it should perform
approved graph cleanup and return a structured summary of what changed.

## Hygiene Expectations

VP2 hygiene should cover at least these cases:

### Duplicate-near-identical nodes

Detect entities that are materially the same concept but were written multiple
times with slightly different titles, aliases, or summaries.

Expected behavior:

- group likely duplicates
- explain why they were grouped
- recommend consolidation targets
- support explicit application of the consolidation

### Orphan nodes

Detect nodes that no longer participate meaningfully in the graph and are safe
cleanup candidates.

Expected behavior:

- identify structurally orphaned nodes
- separate informational orphans from likely garbage where possible
- recommend pruning
- support explicit apply

### Contradictory facts

Detect facts in tension with one another and support resolution.

Expected behavior:

- identify conflicting assertions
- explain the basis of the contradiction
- allow a resolution flow instead of only passive reporting
- preserve enough provenance that future agents can understand what happened

## Graph Stats Expectations

VP2 stats are snapshot-oriented, not trend-oriented.

The product should provide enough information for an agent to decide whether the
graph is:

- relatively tidy and structured
- moderately noisy but still usable
- chaotic enough to justify hygiene before retrieval-heavy work

Stats should be compact, structured, and suitable for automation.

## Cognitive Indicators

VP2 should explicitly model graph health using indicators such as:

- **Duplication rate** — how much of the graph appears redundantly represented
- **Orphan rate** — how much content is disconnected from useful structure
- **Contradictory fact count** — how many unresolved fact conflicts exist
- **Density / clustering indicators** — whether structure suggests coherent
  connected knowledge or fragmented/noisy accumulation

These are not vanity metrics. They are operational signals for whether the
graph is safe to lean on.

## Example End-to-End Workflow

1. An agent receives a repo with an existing local graph.
2. Before retrieving task context, the agent runs `graph stats`.
3. The stats show elevated duplication and orphan rates.
4. The agent runs `graph hygiene` in suggest mode.
5. The CLI returns duplicate groups, orphan candidates, and contradictions with
   proposed resolutions.
6. The agent reviews the suggestions and applies the safe subset explicitly.
7. The graph is updated, and the change can still be inspected using existing
   revision and diff mechanisms.
8. The agent retrieves context from a cleaner graph with higher trust.

## Success Criteria

### Priority 1

Improve the structural health of the shared graph so retrieval remains useful as
the memory grows.

### Priority 2

Give agents a compact operational picture of graph quality before they depend on
it for context and decision making.

## Non-Goals For VP2

- building a human graph monitoring dashboard
- fully autonomous self-healing graph maintenance without explicit approval
- introducing time-series graph analytics as a primary product surface
- expanding to multi-repo federation in this phase

## Guidance For The Architect

The architect should optimize VP2 for:

- safe graph cleanup with explicit apply semantics
- structured metrics that support agent decisions
- contradiction detection and resolution workflows
- reuse of the VP1 persistence, provenance, revision, and diff foundations

The architect should avoid overcomplicating VP2 into a full observability or
data-governance platform.
