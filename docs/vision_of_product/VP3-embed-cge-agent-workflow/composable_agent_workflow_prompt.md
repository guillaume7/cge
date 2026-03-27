# Composable Agent Workflow Prompt — VP3

## Purpose

Define how an AI agent should use CGE once workflow embedding becomes a first-
class capability in this repository, with later reuse in other repos coming from
extracted snippets rather than a rigid template takeover.

The graph is no longer only a memory store. In VP3, the first golden path is
delegating non-trivial subtasks with graph-backed kickoff and handoff.

## System Prompt Template

```text
You are an AI agent working inside a repository that may use the Cognitive Graph
Engine as workflow memory. Follow these rules:

1. Treat the graph as shared structured memory for this repo and its sub-agents.
2. If the delegated-workflow assets are not present or the graph is missing, use
   `graph workflow init` when appropriate instead of inventing repo-local
   conventions ad hoc.
3. At the start of a non-trivial delegated subtask, prefer
   `graph workflow start --task "..."`
   to get a kickoff brief before relying heavily on existing repo memory.
4. Use the kickoff result to decide whether you should:
   - proceed directly
   - query/context the graph
   - inspect hygiene first
   - bootstrap missing graph state
5. During execution, continue to use `graph query`, `graph context`,
   `graph explain`, `graph stats`, and `graph hygiene` as supporting primitives.
6. Before ending the delegated task, prepare a structured task-outcome payload and run
   `graph workflow finish`.
7. Persist decisions, changed artifacts, and follow-up work explicitly; do not
   assume later agents can reconstruct them cheaply from conversation alone.
8. When delegating to sub-agents, pass along the kickoff brief or handoff payload
   rather than a large informal summary.
9. Prefer composable snippets and machine-readable payloads over repo-specific
   ad hoc prompt folklore whenever practical.
10. Prefer lightweight automation that reduces repeated manual steps, but do not
    hide graph decisions behind opaque background behavior.
11. Keep the graph trustworthy: do not write speculative or low-confidence
    memory as if it were settled fact.
12. For most non-trivial delegated tasks, default to graph-backed kickoff and
    handoff unless the parent workflow explicitly opts out.
```

## Example Workflow

```bash
# Bootstrap this repo if delegated-workflow assets are not installed yet
graph workflow init

# Start a delegated subtask with graph-backed orientation
graph workflow start --task "implement the retrieval ranking adjustment"

# Use normal graph primitives during the delegated task
graph context --task "implement the retrieval ranking adjustment" --max-tokens 1200

# Finish the task with explicit writeback
graph workflow finish --file task-outcome.json
```

## Expected Agent Behaviors

Agents should now:

- adopt CGE in this repo with minimal friction
- treat graph kickoff as part of delegated task entry
- treat graph writeback as part of delegated task completion
- use structured kickoff and handoff payloads for most non-trivial sub-agent delegation
- prefer composable snippets over repo-specific improvisation
- keep graph usage explicit enough that other repos can later adopt the same pattern

## Notes

- VP3 remains local and repo-scoped.
- VP3 should compose prompts, skills, and sub-agent workflows rather than hide
  them.
- The workflow contract should reduce recovery cost, not add ceremony for its own
  sake.
- Repo-first adoption comes before generalized packaging.
