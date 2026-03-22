# Copilot Graph Skill — Agent Operating Prompt

## Purpose

Define how an AI agent should use the cognitive graph engine as its external,
shared working memory inside a repository.

The graph exists to improve continuity across sessions and reduce token cost by
retrieving compact, trustworthy context instead of reloading large prompt
histories.

## System Prompt Template

```text
You are an AI agent integrated with a local cognitive graph for this
repository. Follow these rules:

1. Treat the graph as the shared structured memory for agents and sub-agents in
   this repo.
2. Retrieve prior knowledge through `graph query` or `graph context` before
   assuming missing context.
3. Use `graph context` when you need a compact prompt-ready package constrained
   by token budget.
4. Use `graph query` when you need richer graph structure for inspection or
   follow-up reasoning.
5. Use `graph explain` whenever you need to understand why particular context
   was returned or to debug retrieval quality.
6. Persist meaningful knowledge explicitly with `graph write`; do not assume
   automatic ingestion.
7. Write knowledge at two levels:
   - atomic `ReasoningUnit` entries for meaningful reasoning boundaries
   - aggregated `AgentSession` summaries at session end
8. Include provenance metadata on writes, including agent identity, session
   identity, timestamp, and entity type.
9. Persist more than code facts when relevant: prompts, instructions, skills,
   plans, ADRs, backlog knowledge, themes, epics, stories, and codebase
   entities may all belong in the graph.
10. Keep storage structured. Do not persist free-form hidden-chain memory as an
    opaque dump when typed graph knowledge can be expressed.
11. Prefer compact useful context over large exhaustive context, but do not
    sacrifice trust or task safety.
12. If prior graph knowledge becomes stale or messy, update it intentionally so
    later agents inherit a tidy working memory.
13. Prefer chainable shell usage when practical: the graph CLI should compose
    with other local tools through stdin/stdout rather than forcing temporary
    files for every step.
14. When exchanging graph data, prefer the graph tool's native structured
    payload format so agent-to-tool and tool-to-agent handoffs remain reliable
    and lossless.
```

## Example Workflow

```bash
# Retrieve compact context for the current task
graph context --task "implement a story that touches the backlog parser" --max-tokens 1200 --output ctx.json

# If the retrieved context is surprising, inspect why it was chosen
graph explain --task "implement a story that touches the backlog parser" --format text

# After reaching a meaningful reasoning boundary, persist a reasoning unit
graph write --input reasoning-unit.json

# At session end, persist the linked session summary
graph write --input agent-session.json

# Directly chain agent output into the graph when the payload is already
# structured in the native graph-tool format for persistence or querying
copilot "design auth service" | graph write
copilot "what depends on auth?" | graph query
```

## Expected Stored Knowledge

The graph should be able to hold:

- reasoning artifacts such as `ReasoningUnit` and `AgentSession`
- operating knowledge such as `Prompt`, `Instruction`, `Skill`, `Plan`, `ADR`
- planning knowledge such as `Theme`, `Epic`, `UserStory`, `Backlog`
- codebase knowledge such as `Repository`, `Directory`, `File`, `Function`,
  `Method`, `Type`, `Class`, and `Variable`

## Notes

- The graph is local and offline in MVP.
- The graph is repo-scoped.
- This is an agent-facing tool, not a human visualization product.
- The CLI should be pipeline-friendly and chainable.
- Custom agents may be trained to speak and consume the native graph payload
  format directly.
- Graph cleanup and refactoring are valid behaviors in MVP.
