---
description: "Run Loom MCP weaving mode to drive server-side GitHub PRs human-out-of-loop until completion. Use when: running /run-loom, operating Loom, weaving PR lifecycle automatically."
agent: "loom-mcp-operator"
tools: [loom/loom_next_step, loom/loom_checkpoint, loom/loom_heartbeat, loom/loom_get_state, loom/loom_abort, github/github-mcp-server/default]
---

## Agents & Skills

| Agent | Skills | Key Tools |
|-------|--------|-----------|
| @loom-mcp-operator | `loom-mcp-loop` | Loom MCP, GitHub MCP |

Begin Loom weaving mode.

## Pre-flight Checks

Before entering the loop, verify:
1. Loom MCP tools are available (`loom_next_step`, `loom_checkpoint`, `loom_heartbeat`, `loom_get_state`, `loom_abort`)
2. GitHub MCP tools for issues/PRs/reviews/merge are available
3. `docs/plan/backlog.yaml` is present if Loom needs local planning context for diagnostics

## MCP Setup

### Loom MCP Server

If Loom is not yet configured as an MCP server, add it to your VS Code MCP configuration (`.vscode/mcp.json` or user settings):

```json
{
  "mcpServers": {
    "loom": {
      "type": "stdio",
      "command": "loom",
      "args": ["mcp"]
    }
  }
}
```

Install the Loom binary:

```bash
# Download and install (Linux/macOS — adjust OS/ARCH as needed)
VERSION=v1.0.0 OS=linux ARCH=amd64
curl -L -o loom "https://github.com/guillaume7/loom/releases/download/${VERSION}/loom-${OS}-${ARCH}"
install -m 0755 loom /usr/local/bin/loom
```

See [https://github.com/guillaume7/loom](https://github.com/guillaume7/loom) for full installation instructions.

### GitHub MCP Server

Ensure the GitHub MCP server is configured (built-in with VS Code Copilot, or add explicitly):

```json
{
  "mcpServers": {
    "github": {
      "type": "http",
      "url": "https://api.githubcopilot.com/mcp/"
    }
  }
}
```

## Environment

Set the required environment variables before running:

```bash
export LOOM_OWNER=your-github-org
export LOOM_REPO=your-target-repo
export LOOM_TOKEN=<your_github_token>   # e.g. output of: gh auth token
```

## Execution

Start the canonical Loom loop and continue until Loom reports `COMPLETE` or transitions to `PAUSED`:
1. Call `loom_next_step`
2. Execute exactly one corresponding GitHub-side workflow step
3. Call `loom_checkpoint` with the canonical action
4. While waiting on async gates, call `loom_heartbeat` every 30 seconds

If Loom and live GitHub state diverge and cannot be reconciled safely, call `loom_abort` and return a concise operator handoff.
