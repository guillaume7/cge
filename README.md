# Copilot Autopilot

A template repository for **AI-driven autonomous product development** using VS Code Copilot agents.

## What is this?

A squad of specialized Copilot agents that collaborate through a structured lifecycle to take a product from **vision** to **working software** — autonomously.

## Development Lifecycle

```
┌─────────────────────────────────────────────────────────────────┐
│  Phase 1: VISION          Human + AI brainstorm                 │
│  /kickstart-vision        → docs/vision_of_product/VP<n>/       │
├─────────────────────────────────────────────────────────────────┤
│  Phase 2: ARCHITECTURE    Architect agent                       │
│  /plan-product            → docs/architecture/ + docs/ADRs/     │
├─────────────────────────────────────────────────────────────────┤
│  Phase 3: PLANNING        Product Owner agent                   │
│  /plan-product            → docs/themes/TH<n>/ + backlog.yaml  │
│                           → .github/ISSUE_TEMPLATE/ (per epic)  │
├─────────────────────────────────────────────────────────────────┤
│  Phase 4A: LOCAL AUTOPILOT Orchestrator loops the squad          │
│  /run-autopilot           implement → test → review → repeat    │
│                           epic end: integration + review          │
│                           theme end: full test suite + release    │
├─────────────────────────────────────────────────────────────────┤
│  Phase 4B: LOOM WEAVING   Loom MCP drives GitHub server-side    │
│  /run-loom                create issue → assign @copilot →      │
│                           poll PR → gate → merge → repeat        │
│                           (requires loom binary as MCP server)   │
└─────────────────────────────────────────────────────────────────┘
```

## Quick Start

1. **Design your product vision**
   - Run `/kickstart-vision` in Copilot Chat
   - Brainstorm freely — capture ideas in `docs/vision_of_product/VP1-mvp/`
   - Add more phases: `VP2-<feature>/`, `VP3-<feature>/`, etc.

2. **Generate architecture and plan**
   - Run `/plan-product` in Copilot Chat
   - Architect agent produces `docs/architecture/` and `docs/ADRs/`
   - Product Owner agent breaks vision into themes/epics/stories and builds `docs/plan/backlog.yaml`
   - Product Owner also creates `.github/ISSUE_TEMPLATE/TH<n>-E<m>-<slug>.md` — one per epic, so Loom (Phase 4B) can open GitHub issues that `@copilot` implements directly

3. **Launch autopilot** *(choose one)*

   **Option A — Local Autopilot** (no extra tools required):
   - Run `/run-autopilot` in Copilot Chat in "Autopilot" mode
   - Orchestrator executes stories locally: implement → test → review
   - Session state persists in `docs/plan/backlog.yaml` — resume anytime

   **Option B — Loom Weaving** (server-side, human-out-of-loop):

   Install the [Loom](https://github.com/guillaume7/loom) binary and register it as an MCP server:

   ```bash
   # Download and install (Linux/macOS — adjust OS/ARCH as needed)
   VERSION=v1.0.0 OS=linux ARCH=amd64
   curl -L -o loom "https://github.com/guillaume7/loom/releases/download/${VERSION}/loom-${OS}-${ARCH}"
   install -m 0755 loom /usr/local/bin/loom
   ```

   Add to your VS Code MCP config (`.vscode/mcp.json` or user settings):

   ```json
   { "mcpServers": { "loom": { "type": "stdio", "command": "loom", "args": ["mcp"] } } }
   ```

   Set credentials and run:

   ```bash
   export LOOM_OWNER=your-github-org
   export LOOM_REPO=your-repo
   export LOOM_TOKEN=<your_github_token>   # e.g. output of: gh auth token
   ```

   Then run `/run-loom` in Copilot Chat. Loom drives GitHub server-side — creates issues, assigns `@copilot`, polls PRs, gates and merges. Loom maintains its own SQLite database for weaving runtime state (current story, PR, FSM position); `docs/plan/backlog.yaml` remains the single source of truth for planning, story statuses, and resume.

## Recommended MCP & CLI Tools

The agent squad uses a set of MCP servers and CLI tools. Configure them once for all agents to work optimally.

### Required for all modes

| Tool | Purpose | Setup |
|------|---------|-------|
| **GitHub MCP** | PRs, CI checks, code search, issue management | Built-in with VS Code Copilot; or add to `.vscode/mcp.json`: `{"mcpServers":{"github":{"type":"http","url":"https://api.githubcopilot.com/mcp/"}}}` |
| **git CLI** | Commits, diffs, file history | Pre-installed on most systems — `git --version` to verify |
| **gh CLI** | CI log retrieval, issue/PR management | [cli.github.com](https://cli.github.com) — run `gh auth login` after install |

### Required for UI projects (developer agent)

| Tool | Purpose | Setup |
|------|---------|-------|
| **Playwright MCP** | Browser automation for end-to-end UI tests | Add to `.vscode/mcp.json`: `{"mcpServers":{"playwright":{"type":"stdio","command":"npx","args":["@playwright/mcp@latest"]}}}` |

### Required for Loom weaving (Phase 4B only)

| Tool | Purpose | Setup |
|------|---------|-------|
| **Loom MCP** | Drives the server-side PR weaving FSM | See Option B above and [github.com/guillaume7/loom](https://github.com/guillaume7/loom) |

> See [`.github/agents/README.md`](.github/agents/README.md) for the full per-agent tool matrix, and the `the-copilot-build-method` skill for MCP server configuration snippets.

## Agent Squad

| Agent | Role | Phase |
|-------|------|-------|
| **orchestrator** | Local autopilot loop: sequencing, parallelism, state management | 4A |
| **product-owner** | Vision → themes → epics → BDD stories | 3 |
| **architect** | Vision → architecture, tech stack, ADRs | 2 |
| **developer** | Implements + tests one user story per session | 4A |
| **reviewer** | Code review: correctness, security, conventions | 4A |
| **troubleshooter** | Diagnoses and fixes failed stories | 4A |
| **loom-mcp-operator** | Drives Loom MCP tools to weave PRs server-side | 4B |

> Phase 4B also includes `loom-orchestrator`, `loom-gate`, `loom-debug`, and `loom-merge` sub-agents. See [`.github/agents/README.md`](.github/agents/README.md) for the full squad.

## Directory Structure

```
docs/
├── vision_of_product/    # Free-form product vision (VP<n> → TH<n>)
├── architecture/         # System design + tech stack
├── ADRs/                 # Architecture Decision Records
├── themes/               # TH<n>/epics/E<m>/stories/US<l>.md
└── plan/
    ├── backlog.yaml      # Core state file (pure YAML dependency graph)
    └── session-log.md    # Autopilot session history

.github/
├── agents/               # 11 specialized agents (6 core + 5 loom)
├── prompts/              # Lifecycle prompts (/kickstart-vision, /plan-product,
│                         #   /run-autopilot, /run-loom, /review, /troubleshoot)
├── skills/               # 6 reusable skills (the-copilot-build-method,
│                         #   bdd-stories, backlog-management, code-quality,
│                         #   architecture-decisions, loom-mcp-loop)
├── ISSUE_TEMPLATE/       # One issue template per epic (generated by /plan-product)
│   │                     #   TH<n>-E<m>-<slug>.md — used by Loom (Phase 4B)
│   └── archive/          # Completed-theme templates (moved here at theme boundary)
├── hooks/                # Session lifecycle automation
└── copilot-instructions.md
```

## Key Conventions

- **VP<n> ↔ TH<n>**: Vision phases map 1:N to implementation themes
- **1 story = 1 developer session**: Stories are sized for single-agent execution
- **Backlog is truth**: `docs/plan/backlog.yaml` is the only state file the orchestrator trusts
- **Hybrid BDD**: Stories contain acceptance criteria + Given/When/Then scenarios
- **Language-agnostic**: Architect agent chooses tech stack based on your vision
- **Two execution modes**: `/run-autopilot` runs locally; `/run-loom` offloads to GitHub server-side via the Loom binary

## Using as a Template

### From GitHub (recommended)

```bash
gh repo create my-product --template guillaume-galp/copilotautopilot --public --clone && cd my-product
```

### Without GitHub CLI

```bash
git clone --depth 1 https://github.com/guillaume-galp/copilotautopilot.git my-product \
  && cd my-product && rm -rf .git && git init && git add . \
  && git commit -m "Initial commit from copilotautopilot template"
```

Then open the folder in VS Code and run `/kickstart-vision` in Copilot Chat to start designing your product.

## License

MIT
