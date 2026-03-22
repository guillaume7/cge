# Cognitive Graph Engine CLI — Interface Vision

## Overview

The `graph` CLI provides a local, repo-scoped, shared graph memory for AI
agents. Its job is to persist structured project knowledge and return compact,
trustworthy context slices that help agents continue work across sessions and
handoffs.

This is an agent-first CLI. It is not designed as a general human graph
browser, and MVP does not include visualization features.

## Core Behavior

The CLI should let agents:

- initialize a repository-local graph
- write structured knowledge explicitly
- retrieve task-relevant graph data
- project that data into a token-budgeted context package
- explain why retrieval returned specific graph elements
- compare graph states over time
- compose commands through stdin/stdout in shell pipelines
- exchange data through a native machine-oriented graph payload format

The MVP stays local and offline.

## Commands

### 1. `graph init`

Initialize the graph for the current repository.

```bash
graph init [--path ./graph]
```

Expected outcome:

- prepares repository-scoped graph storage
- makes the graph available to all agents working in the repo

### 2. `graph write`

Write structured nodes, edges, and metadata to the graph.

```bash
graph write --input <json_file>
```

Writes are explicit and agent-driven. MVP should support storing:

- reasoning units
- session summaries
- project metadata
- planning artifacts
- codebase entities

The command should be chainable. Besides file input, it should support reading
its payload from standard input so workflows like `copilot "design auth
service" | graph write` are natural.

The stdin contract should use the same native structured graph payload format as
file-based writes so custom agents can emit it directly.

Each write should carry provenance metadata such as:

- `agent_id`
- `session_id`
- `timestamp`
- `entity_type`

Illustrative payload:

```json
{
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-42",
    "timestamp": "2026-03-21T14:00:00Z"
  },
  "nodes": [
    {
      "id": "ru-123",
      "type": "ReasoningUnit",
      "properties": {
        "task": "Implement graph context projection",
        "summary": "Need compact context with provenance"
      }
    }
  ],
  "edges": []
}
```

### 3. `graph query`

Return a relevant structured subgraph for a task or concept.

```bash
graph query --task "prepare context for backlog refactor" --format json
```

Expected behavior:

- combines graph-aware retrieval and semantic retrieval
- returns richer structured data than `graph context` when needed
- stays offline in MVP
- can participate in shell pipelines using stdin for task input and stdout for
  machine-readable output
- should return a stable structured output format suitable for direct
  consumption by downstream agents

### 4. `graph context`

Return a prompt-ready compact context package for an agent.

```bash
graph context --task "prepare context for backlog refactor" --max-tokens 1200 --output ctx.json
```

Expected behavior:

- balances token minimization with safe retrieval quality
- includes enough provenance for trust and debugging
- produces a machine-friendly format for prompt injection into an agent

### 5. `graph explain`

Explain why a query or context request returned specific results.

```bash
graph explain --task "prepare context for backlog refactor" --format text
```

Expected behavior:

- surfaces why nodes or edges were selected
- helps debug stale, noisy, or missing context
- makes retrieval behavior more trustworthy for agents

### 6. `graph diff`

Compare graph states to inspect how shared memory changed.

```bash
graph diff --from <state-a> --to <state-b>
```

Expected behavior:

- highlights meaningful graph changes
- helps inspect graph cleanup or refactoring
- supports debugging of agent-written memory evolution

## Supported Knowledge Domains

The graph should be able to represent at least:

- reasoning artifacts: `ReasoningUnit`, `AgentSession`
- project operating knowledge: `Prompt`, `Instruction`, `Skill`, `Plan`, `ADR`
- planning knowledge: `Theme`, `Epic`, `UserStory`, `Backlog`
- codebase knowledge: `Repository`, `Directory`, `File`, `Function`, `Method`,
  `Type`, `Class`, `Variable`

## Design Principles

1. Graph memory is a first-class substrate for agent continuity.
2. Explicit structured writes beat implicit hidden memory.
3. Compact context matters, but not at the expense of trust.
4. Provenance and explanation are MVP features, not afterthoughts.
5. The graph is allowed to evolve and be cleaned up over time.
6. The CLI should be chainable with other local agent tools.
7. The CLI should expose a stable native graph payload contract for agent
   interoperability.

## Example Workflow

```bash
# initialize repo graph
graph init

# agent writes a reasoning unit
graph write --input reasoning-unit.json

# direct tool chaining should also work
copilot "design auth service" | graph write
copilot "what depends on auth?" | graph query

# later agent retrieves compact context
graph context --task "continue work on query ranking" --max-tokens 1200 --output ctx.json

# if results seem odd, inspect why
graph explain --task "continue work on query ranking" --format text

# session closes with summary write
graph write --input agent-session.json

# compare graph states after cleanup
graph diff --from before-cleanup --to after-cleanup
```
