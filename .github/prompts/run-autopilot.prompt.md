---
description: "Launch the autopilot orchestrator to execute the product backlog autonomously. Use when: running autopilot, starting autonomous development, executing backlog."
agent: "orchestrator"
tools: [read, edit, search, execute, agent, todo, github/github-mcp-server/default]
---

## Agents & Skills

| Agent | Skills | Key Tools |
|-------|--------|-----------|
| @orchestrator | `the-copilot-build-method`, `backlog-management` | GitHub MCP, git CLI, gh CLI |
| @developer | `the-copilot-build-method`, `bdd-stories` | GitHub MCP, Playwright MCP, git CLI, gh CLI |
| @reviewer | `the-copilot-build-method`, `code-quality` | GitHub MCP, git CLI |
| @troubleshooter | `the-copilot-build-method`, `bdd-stories`, `code-quality` | GitHub MCP, gh CLI, git CLI |
| @product-owner | `the-copilot-build-method`, `bdd-stories`, `backlog-management` | GitHub MCP, gh CLI, git CLI |

Begin autonomous execution of the product backlog.

## Pre-flight Checks

Before starting the loop, verify:
1. `docs/plan/backlog.yaml` exists and contains valid YAML with at least one theme
2. `docs/architecture/` exists with tech stack and component definitions
3. `docs/themes/` contains at least one theme with epics and stories
4. Check `docs/plan/backlog.yaml` for any `in-progress` stories — if found, trigger crash recovery (assess and continue, reset, or escalate)

## Execution

Start the autopilot loop as defined in your orchestrator instructions. Process stories in dependency order, running the full cycle (implement → test → review) for each.

Report progress after each story completion.
