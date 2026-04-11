# Agent Guide

The active Copilot surface is intentionally small: one planning pair, one
execution lead, and three focused execution subagents.

## Active Agents

| Agent | Phase | Role | Invocation |
| --- | --- | --- | --- |
| `architect` | 2 | Turns approved vision into architecture docs and ADRs | `/plan-product` |
| `product-owner` | 3 | Breaks architecture into themes, epics, stories, and backlog state | `/plan-product` |
| `orchestrator` | 4 | Runs the local autopilot loop and manages backlog state | `/run-autopilot` |
| `developer` | 4 | Implements and tests one story | orchestrator only |
| `reviewer` | 4 | Reviews correctness, security, and conventions | orchestrator only |
| `troubleshooter` | 4 | Fixes failed stories after build or test failures | orchestrator only |

## Skills

| Agent | Build method | BDD stories | Backlog | Code quality | Architecture |
| --- | --- | --- | --- | --- | --- |
| `architect` | ✓ |  |  |  | ✓ |
| `product-owner` | ✓ | ✓ | ✓ |  |  |
| `orchestrator` | ✓ |  | ✓ |  |  |
| `developer` | ✓ | ✓ |  |  |  |
| `reviewer` | ✓ |  |  | ✓ |  |
| `troubleshooter` | ✓ | ✓ |  | ✓ |  |

## Tooling

| Agent | GitHub MCP | Playwright MCP | git | gh |
| --- | --- | --- | --- | --- |
| `architect` | ✓ |  | ✓ |  |
| `product-owner` |  |  | ✓ |  |
| `orchestrator` | ✓ |  | ✓ | ✓ |
| `developer` | ✓ | ✓ | ✓ | ✓ |
| `reviewer` | ✓ |  | ✓ |  |
| `troubleshooter` | ✓ |  | ✓ | ✓ |

## User-facing prompts

| Prompt | Purpose |
| --- | --- |
| `/kickstart-vision` | Align on a new or revised product vision |
| `/plan-product` | Run architecture, then planning |
| `/run-autopilot` | Execute the backlog locally |
