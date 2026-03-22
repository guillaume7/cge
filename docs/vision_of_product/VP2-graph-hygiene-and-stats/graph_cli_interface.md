# Cognitive Graph Engine CLI — VP2 Interface Vision

## Overview

VP2 extends the `graph` CLI with graph-health inspection and safe cleanup
operations.

The new interface goal is not to make the graph prettier for humans. It is to
help agents:

- measure graph quality
- detect disorder
- understand suggested cleanup actions
- explicitly apply selected fixes

This remains a local, repo-scoped, machine-oriented CLI.

## New Commands

### 1. `graph stats`

Return a snapshot of graph structure and health indicators.

```bash
graph stats
```

Expected behavior:

- reports total node count
- reports total relationship count
- reports duplication rate
- reports orphan rate
- reports contradictory fact count
- reports density / clustering indicators
- emits structured output suitable for agent automation

Illustrative output shape:

```json
{
  "schema_version": "v1",
  "command": "stats",
  "status": "ok",
  "result": {
    "snapshot": {
      "nodes": 540,
      "relationships": 1320
    },
    "indicators": {
      "duplication_rate": 0.08,
      "orphan_rate": 0.03,
      "contradictory_facts": 4,
      "density_score": 0.71,
      "clustering_score": 0.64
    }
  }
}
```

### 2. `graph hygiene`

Inspect graph disorder and return suggested fixes.

```bash
graph hygiene
```

Default behavior must be suggest-only.

Expected behavior:

- identifies near-identical duplicate nodes
- identifies orphan nodes
- identifies contradictory facts
- proposes consolidations, prunes, and resolutions
- explains why each suggestion exists
- returns machine-readable candidate actions

Illustrative output shape:

```json
{
  "schema_version": "v1",
  "command": "hygiene",
  "status": "ok",
  "result": {
    "mode": "suggest",
    "suggestions": {
      "duplicate_groups": [],
      "orphan_nodes": [],
      "contradictions": []
    }
  }
}
```

### 3. `graph hygiene --apply`

Apply an explicitly approved hygiene action set.

```bash
graph hygiene --apply --file hygiene-plan.json
```

Expected behavior:

- does not run implicitly by default
- applies only explicit selected actions
- returns structured change summary
- records changes in the graph's revision history

Illustrative applied result:

```json
{
  "schema_version": "v1",
  "command": "hygiene",
  "status": "ok",
  "result": {
    "mode": "apply",
    "applied": {
      "consolidated_duplicates": 3,
      "pruned_orphans": 7,
      "resolved_contradictions": 2
    },
    "revision": {
      "anchor": "..."
    }
  }
}
```

## Hygiene Action Types

VP2 should support at least these action classes:

- `consolidate_duplicate_nodes`
- `prune_orphan_nodes`
- `resolve_contradiction`

Each action should be explainable, attributable, and machine-readable.

## Interaction Principles

1. Suggest mode is the default.
2. Apply mode must be explicit.
3. Stats should be cheap enough to run before retrieval-heavy workflows.
4. Hygiene output should be understandable by agents without free-form prose
   parsing.
5. Hygiene changes should remain compatible with provenance and diff workflows.

## Example Workflow

```bash
# Inspect the current graph state
graph stats

# Ask for graph cleanup suggestions
graph hygiene --output hygiene-suggestions.json

# Review suggested cleanup and apply the selected plan
graph hygiene --apply --file hygiene-plan.json

# Inspect what changed
graph diff --from <before> --to <after>
```

## Design Notes

- `graph stats` is snapshot-based in VP2; trend analysis is not required.
- `graph hygiene` is not a background daemon; it is an explicit agent workflow.
- Contradictions should support resolution, not only reporting.
- The command surface should remain compact: one stats command and one hygiene
  command are preferable to a fragmented command tree in VP2.
