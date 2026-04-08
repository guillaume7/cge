---
description: "Launch the autopilot orchestrator to execute the product backlog autonomously. Use when: running autopilot, starting autonomous development, executing backlog."
agent: "orchestrator"
tools: [read, edit, search, execute, agent, todo, github/github-mcp-server/default]
---

## Repo-local delegated workflow

For most non-trivial delegated subtasks in this repo, use the explicit repo
dogfooding helper:

- kickoff:
  `bash .github/hooks/scripts/repo-delegated-workflow.sh kickoff --task "<delegated task>"`
- handoff:
  `bash .github/hooks/scripts/repo-delegated-workflow.sh handoff --file task-outcome.json`
- direct fallback:
  `graph workflow init`, `graph workflow start`, and `graph workflow finish`
- opt-out:
  `--opt-out` or `CGE_REPO_WORKFLOW_OPTOUT=1`

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

At theme boundaries, do not declare the theme complete until the root
`README.md`, install examples, command tables, and release-version references
match the shipped functionality.
