# VP1 — Cognitive Graph Engine MVP

## Vision Summary

Build a repo-scoped CLI that gives AI agents a shared, persistent, local
graph memory they can read from and write to during work. The graph is not a
human-facing knowledge browser. It is an agent-native substrate designed to
improve task continuity across agent handoffs while reducing token costs by
returning only the compact, trustworthy context needed for the current task.

The CLI must stay local and offline. It should work on a single machine,
inside a repository, and support multiple cooperating agents and sub-agents
sharing the same graph.

## Product Intent

LLM agents repeatedly lose context between sessions, reload too much prompt
material, and pay a large token penalty to recover project understanding. This
product exists to externalize working memory into a structured graph so agents
can compound knowledge over time instead of starting from scratch.

The graph should contain more than code structure. It should capture the
project's operating knowledge: reasoning units, agent sessions, prompts,
skills, instructions, plans, ADRs, themes, epics, user stories, backlog data,
and codebase entities such as files, directories, functions, methods, types,
classes, and variables.

The result should be a compact context package that lets an agent continue a
task fluidly without rehydrating large prompt histories or scanning the whole
repository again.

## Primary Users

The sole users are AI agents.

- Main agents performing tasks in a repository
- Sub-agents collaborating within the same repo-scoped graph
- Future orchestration flows that need shared memory between agent handoffs

Humans may inspect files or outputs, but the CLI is optimized for machine use,
not manual graph exploration.

## Core Jobs To Be Done

1. Let an agent initialize a local shared graph for a repository.
2. Let agents explicitly write structured knowledge into the graph.
3. Let agents retrieve a relevant subgraph for a task using both graph-aware
   and semantic retrieval approaches.
4. Let agents project that subgraph into a compact context package bounded by a
   token budget.
5. Let agents understand why context was returned through explanation,
   provenance, and trust/debug mechanisms.
6. Let agents compare graph states over time and keep the graph tidy through
   updates and refactoring.

## Product Principles

- **Agent-native first**: optimize for machine workflows, not human UX.
- **Local and offline**: MVP must not depend on hosted services.
- **Repo-scoped memory**: each repository owns its graph context.
- **Explicit writes**: agents decide what to persist; MVP does not auto-ingest
  the repo.
- **Structured persistence**: graph data is typed and queryable, not a bag of
  prompt transcripts.
- **Compact context**: context retrieval must reduce token overhead.
- **Trusted retrieval**: every returned context slice should be explainable and
  traceable back to source nodes.
- **Chainable by design**: commands should compose cleanly through stdin/stdout
  so agents can pipe outputs across tools.
- **Native graph protocol**: agents should speak and consume a stable structured
  graph-tool payload format so chaining is lossless and machine-reliable.
- **Compound intelligence**: the graph should let later agents build on earlier
  work instead of repeating it.
- **Non-intrusive support**: the graph augments agents, it does not constrain
  their full internal workflow.

## MVP Scope

### Included

- A Go CLI backed by a local embedded graph database
- Repository-scoped graph initialization
- Explicit graph writes from agents
- Shared graph usage across agents and sub-agents on one machine
- Support for reasoning units and session-level summaries
- Support for project metadata entities and codebase-related entities
- Retrieval through graph-aware and semantic approaches that stay local/offline
- Token-budget-aware context projection
- Explanation and provenance for trust/debugging
- Graph state comparison
- Unix-style stdin/stdout composability for chaining with agent tools
- A native structured input/output format that custom agents can emit and
  consume directly
- Ability to update or rewrite graph content to keep it clean and current

### Excluded

- Human visualization features
- Remote or multi-machine synchronization
- Hosted services as a requirement for core workflows
- Automatic repository ingestion in MVP
- Immutable historical audit trails in MVP

## MVP Command Surface

The MVP command set is:

- `graph init`
- `graph write`
- `graph query`
- `graph context`
- `graph explain`
- `graph diff`

No human-oriented visualization command is required in VP1.

The CLI must also be chainable in shell pipelines so it composes naturally with
agent tooling. Typical target workflows include:

```bash
copilot "design auth service" | graph write
copilot "what depends on auth?" | graph query
```

This chainability is not just convenience. Custom agents are expected to be
trained to "listen" and "speak" in the graph tool's native structured content
format, so the CLI must preserve a stable machine-oriented contract across
stdin, stdout, and file-based I/O.

## Command Intent

### `graph init`

Initialize the repository-scoped graph and any required local storage so agents
in the repo can share the same memory substrate.

### `graph write`

Persist structured nodes, edges, and supporting metadata into the graph.

`graph write` should accept structured input from a file and from standard
input so agents can pipe generation output directly into persistence workflows.

The structured payload format accepted over stdin should be the same native
graph-tool format used for file-based writes.

Writes should support both:

- **Reasoning-unit granularity** for atomic task progress
- **Agent-session granularity** for end-of-session rollups

MVP writes are explicit and should include provenance metadata such as agent ID,
session ID, timestamp, and entity type.

### `graph query`

Return the relevant structured subgraph for a task or concept. This is the
general retrieval command and may return richer graph structure than the
token-optimized context projection.

`graph query` should support task input through flags and through standard
input so it can participate naturally in shell pipelines.

Its output should be machine-consumable in a native structured graph format so
downstream agents or tools can continue processing without lossy translation.

### `graph context`

Project task-relevant graph data into a compact context package suitable for
feeding into an agent prompt. The output should balance token minimization with
safe retrieval quality.

### `graph explain`

Explain why a query or context request returned particular nodes, edges, or
artifacts. This is a trust and debugging tool for agents, not a human graph
viewer.

### `graph diff`

Show meaningful changes between graph states so agents can inspect how memory
evolved over time and debug graph refactoring or task progression.

## Information That Must Be Representable

VP1 should be able to represent and connect at least these classes of things:

- `ReasoningUnit`
- `AgentSession`
- `Skill`
- `Agent`
- `Prompt`
- `Instruction`
- `Plan`
- `ADR`
- `Theme`
- `Epic`
- `UserStory`
- `Backlog`
- `Repository`
- `Directory`
- `File`
- `Function`
- `Method`
- `Type`
- `Class`
- `Variable`

The architect should refine the exact modeling approach, but the vision expects
all of these to be first-class citizens of the shared memory graph.

## Retrieval Expectations

Context retrieval is central to the product. It should:

- find the smallest useful subgraph for the task
- balance recall and compactness instead of optimizing only one
- support both structural graph retrieval and semantic retrieval
- stay local/offline in MVP
- preserve provenance so agents can verify what they were given
- work well for handoff scenarios between agents and sub-agents

The output of `graph context` should be shaped for prompt consumption, not just
raw graph inspection.

## Chainability Expectations

Shell composability is a first-class MVP requirement.

The architect should assume:

- commands can read meaningful input from stdin when appropriate
- commands can emit machine-consumable output to stdout by default
- file-based I/O is supported where useful, but piping must feel natural
- agent workflows should be able to combine `copilot`, `graph`, and other local
  tools without glue code
- the same native structured format should work consistently across stdin,
  stdout, and file payloads

Illustrative flows:

```bash
copilot "design auth service" | graph write
copilot "what depends on auth?" | graph query
graph context --task "continue auth work" --max-tokens 1200 | copilot
```

## Write Expectations

The graph is written by agents, for agents.

MVP should assume:

- no automatic repo crawling is required
- agents explicitly decide what knowledge deserves persistence
- writes can update or rewrite prior graph content when cleanup is needed
- graph hygiene and refactoring are legitimate workflows
- custom agents may be trained specifically to emit and consume native
  graph-tool structured payloads

This means the graph is a living working memory, not a frozen audit ledger.

## Trust, Debugging, and Safety Expectations

Agents need to know whether returned context can be trusted.

VP1 should therefore provide:

- provenance linking context back to source nodes or sessions
- explanation of retrieval paths or reasons for inclusion
- enough detail to debug false positives, stale knowledge, or missing context
- predictable behavior around task-scoped context packaging

The product should help an agent answer:
"Why did I receive this context, and what source knowledge does it come from?"

## Success Criteria

### Priority 1

Improve task continuity and fluidity across agent sessions and handoffs.

### Priority 2

Reduce token overhead required to resume work or reason about a repo task.

## Example End-to-End Workflow

1. An agent initializes a graph in a repository.
2. The agent writes project entities such as prompts, instructions, plans, or
   codebase entities.
3. During work, the agent writes a `ReasoningUnit` when it reaches a meaningful
   reasoning boundary.
4. At the end of the session, the agent writes an `AgentSession` summary linked
   to the relevant reasoning units and project artifacts.
5. A later agent asks for context for a task.
6. The CLI returns a compact, provenance-aware context package.
7. If the result looks surprising, the agent uses `graph explain`.
8. If the graph contains stale or messy knowledge, an agent updates or
   refactors it and can inspect the change with `graph diff`.

## Non-Goals For VP1

- building a general-purpose visual graph browser
- supporting remote collaboration across machines
- replacing the agent's internal reasoning loop
- fully solving long-term immutable graph history
- auto-modeling every repository artifact without explicit agent intent

## Guidance For The Architect

The architect should optimize for the product's actual value:

- shared memory continuity for agents
- compact and trustworthy context retrieval
- local/offline operation
- explicit structured writes
- support for graph cleanup and evolution

The architect should not overfit the MVP toward human interfaces or hosted
services.
