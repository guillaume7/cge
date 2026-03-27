---
name: the-copilot-build-method
description: 'The overarching autonomous product development methodology. Covers the 4-phase lifecycle (Vision → Architecture → Planning → Autopilot), VP↔TH mapping, Directory conventions, Definition of Done, agent squad roles, and lifecycle ceremonies. Use when: understanding the methodology, onboarding to the process, checking conventions, verifying Definition of Done.'
---

# The Copilot Build Method

An autonomous product development methodology powered by a squad of specialized AI agents operating through a structured lifecycle.

## Philosophy

- **Vision-first**: Products start as free-form ideas, not code
- **Architecture before implementation**: Design decisions are documented before a single line of code
- **BDD-driven**: Every feature is specified as testable scenarios before implementation
- **Incremental delivery**: Products are built in vision phases (VP<n>) that map to implementation themes (TH<n>), with 1:N mapping for large VPs
- **Autonomous execution**: The orchestrator agent loops the squad through implement → test → review cycles
- **Persistent state**: All progress is tracked in `docs/plan/backlog.yaml` for resumability
- **Ceremony at boundaries**: Epic and theme completions trigger quality gates (integration tests, refactor, release notes, and public-doc hygiene)

## The 4 Phases

### Phase 1 — Vision Design (Human + AI)
- Prompt: `/kickstart-vision`
- Output: `docs/vision_of_product/VP<n>-<name>/`
- Free-form brainstorming canvas — no rigid structure
- Start a new VP conversationally: first reflect your understanding, pitch candidate directions, and ask focused product questions before writing downstream artefacts
- Architecture and planning are blocked until the user explicitly aligns on the VP direction
- Each VP<n> maps 1:1 to a theme TH<n>

### Phase 2 — Architecture (Architect Agent)
- Prompt: `/plan-product` (step 1)
- Output: `docs/architecture/` + `docs/ADRs/`
- System design, tech stack selection, component boundaries
- Every significant decision recorded as an ADR

### Phase 3 — Planning (Product Owner Agent)
- Prompt: `/plan-product` (step 2)
- Output: `docs/themes/TH<n>/` + `docs/plan/backlog.yaml` + `.github/ISSUE_TEMPLATE/TH<n>-E<m>-<slug>.md`
- Vision decomposed into themes → epics → user stories
- Stories are hybrid BDD (acceptance criteria + Given/When/Then)
- Backlog YAML is the dependency graph + status state machine
- One GitHub issue template generated per epic (required for Phase 4B Loom weaving)

### Phase 4A — Local Autopilot Execution (Orchestrator Agent)
- Prompt: `/run-autopilot`
- Loop: implement → test → review per story
- Epic end ceremony: integration tests + refactor + review + changelog
- Theme end ceremony: regression tests + release readiness + README/version updates + release notes + vision revalidation
- Failed stories: troubleshooter loop (max 3 attempts, then escalate)

### Phase 4B — Loom Weaving (Loom MCP Operator)
- Prompt: `/run-loom`
- Alternative to Phase 4A; requires the [Loom](https://github.com/guillaume7/loom) binary installed and configured as an MCP server
- The Loom Go binary drives a deterministic FSM: creates GitHub issues → assigns `@copilot` → polls for PRs → gates merges → merges approved PRs
- The `loom-mcp-operator` agent drives Loom MCP tools (`loom_next_step`, `loom_checkpoint`, `loom_heartbeat`, `loom_get_state`, `loom_abort`) and executes one GitHub action per checkpoint
- Sub-agents `loom-gate`, `loom-debug`, and `loom-merge` handle specialized merge gating, CI failure diagnosis, and PR merging respectively
- State persists in a local SQLite database (managed by Loom) — survives VS Code restarts and machine reboots

## VP ↔ TH Mapping Convention

One vision phase can produce **one or more** themes (1:N). Theme numbering is sequential and independent of VP numbering.

## New VP Start Gate

When a user introduces a **new VP idea**:

1. Stay in **Phase 1** first.
2. Explain your understanding of the product intent in your own words.
3. Pitch candidate product directions, trade-offs, and open questions back to the user.
4. Get explicit user alignment on the VP direction.
5. Only then proceed to Phase 2 (architecture) and Phase 3 (theme/backlog planning).

Do **not** create themes, epics, user stories, backlog entries, or issue templates while the product is still in this vision-alignment step.

| Vision Phase | Theme(s) | Relationship |
|:---|:---|:---|
| `VP1-mvp/` | `TH1-<name>/` | 1:1 (simple case) |
| `VP1-mvp/` | `TH1-<name>/`, `TH2-<name>/` | 1:N (large vision phase) |
| `VP2-<feat>/` | `TH3-<name>/` | Sequential numbering continues |

## Definition of Done

### Story Done
1. Code compiles / lints clean
2. All BDD scenario tests pass (if applicable — trivial/spike stories may have fewer or no BDD tests)
3. All acceptance criteria verified
4. Build artifacts produce successfully
5. Code review agent approves (trivial stories: lightweight self-review only, skip full reviewer)
6. Relevant documentation updated

### Epic Done (Story DoD + ceremony)

Ceremony scales with epic size:

**Small epic (≤3 stories)**:
1. All stories `done`
2. Run full test suite across epic stories
3. Brief changelog entry

**Large epic (4+ stories)**:
1. All stories `done`
2. Integration test suite passes across all epic stories
3. Reviewer performs lightweight code quality check
4. Orchestrator generates full epic changelog entry

### Theme Done (Epic DoD + ceremony)
1. All epics `done`
2. Full test suite passes (all tests across all epics)
3. Release readiness: artifacts build, no `failed` stories, and public docs are current:
   - root `README.md` reflects the shipped command surface and user-visible behavior
   - install snippets and examples match the release workflow/artifacts
   - version and release-tag references are updated (for example `v0.2.0`)
4. If `docs/architecture/deployment.md` exists, verify deployment readiness (CI/CD, health checks, rollback)
5. If vision includes NFRs (performance, scalability targets), verify they are covered by test results
6. Orchestrator produces theme release notes
7. Product-owner revalidates theme against `docs/vision_of_product/VP<n>/`
8. **Archive issue templates**: move completed theme's epic templates from `.github/ISSUE_TEMPLATE/TH<n>-*.md` → `.github/ISSUE_TEMPLATE/archive/` to keep the active template set clean for the next theme
9. **User checkpoint**: orchestrator pauses and presents a demo summary to the user:
   - User can **accept** (proceed to next theme), **reject** (rework), or **amend** vision for next VP
   - Vision is frozen only for the theme currently in execution — future VPs can be updated at checkpoints
10. **Lock the theme**: after user accepts, orchestrator sets `locked: true` on the theme in `docs/plan/backlog.yaml` — all associated VP directory, theme directory, story files, and ADRs are now immutable. Note: issue templates were already archived in step 8; archiving is a separate operational concern from locking.

## Naming Conventions

| Entity | Pattern | Example |
|:---|:---|:---|
| Vision Phase | `VP<n>-<slug>/` | `VP1-mvp/` |
| Theme | `TH<n>-<slug>/` | `TH1-core-platform/` |
| Epic | `E<m>-<slug>/` | `E1-user-auth/` |
| User Story | `US<l>-<slug>.md` | `US1-login-form.md` |
| ADR | `ADR-<NNN>-<slug>.md` | `ADR-001-database-choice.md` |

## Agent Squad Roles

| Agent | Phase | Responsibility |
|:---|:---|:---|
| orchestrator | 4A | Local autopilot loop, sequencing, state management |
| product-owner | 3 | Vision → themes/epics/stories + backlog |
| architect | 2 | Vision → architecture + ADRs |
| developer | 4A | Implements + tests one user story per session |
| reviewer | 4A | Code review: correctness, security, conventions |
| troubleshooter | 4A | Diagnoses + fixes failed stories |
| loom-mcp-operator | 4B | Drives Loom MCP tools in the master session |
| loom-orchestrator | 4B | End-to-end FSM driver with gate/debug/merge handoffs |
| loom-gate | 4B | Read-only pre-merge checks (CI, review, draft, conflicts) |
| loom-debug | 4B | CI failure diagnosis; posts structured debug comment |
| loom-merge | 4B | Merge-only agent: calls `merge_pull_request` and returns JSON |

## Recommended Tools per Agent

Each agent has a defined set of MCP servers and CLI tools it should use. Configure these in your VS Code MCP settings before running the autopilot.

### MCP Servers

#### GitHub MCP (`github/github-mcp-server/default`)
Required by: **all agents**. For loom-gate, loom-merge, and loom-debug it is their sole tool; all other agents use it alongside CLI tools or other MCP servers.

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

Use for: searching repositories and code; reading PR diffs; checking CI status; posting comments.

#### Loom MCP (`loom/*`)
Required by: **loom-mcp-operator**, **loom-orchestrator** (Phase 4B only).

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

Tools: `loom_next_step`, `loom_checkpoint`, `loom_heartbeat`, `loom_get_state`, `loom_abort`.

#### Playwright MCP (`playwright`)
Required by: **developer** (for UI/browser end-to-end tests).

```json
{
  "mcpServers": {
    "playwright": {
      "type": "stdio",
      "command": "npx",
      "args": ["@playwright/mcp@latest"]
    }
  }
}
```

Use for: driving a real browser for BDD scenario tests; taking screenshots to verify visual output.

### CLI Tools

| CLI | Agents | Key Commands |
|:----|:-------|:-------------|
| **git** | orchestrator, product-owner, architect, developer, reviewer, troubleshooter | `git status/diff/log/blame/commit` |
| **gh** | orchestrator, product-owner, developer, troubleshooter | `gh run view --log`, `gh pr list`, `gh issue list`, `gh auth token` |

> **Note**: The `gh` CLI must be authenticated (`gh auth login`) before running any agent that needs to read CI logs or manage issues.

### Agent ↔ Tool Summary

| Agent | GitHub MCP | Loom MCP | Playwright MCP | git CLI | gh CLI |
|:------|:----------:|:--------:|:--------------:|:-------:|:------:|
| orchestrator | ✓ | | | ✓ | ✓ |
| product-owner | ✓ | | | ✓ | ✓ |
| architect | ✓ | | | ✓ | |
| developer | ✓ | | ✓ | ✓ | ✓ |
| reviewer | ✓ | | | ✓ | |
| troubleshooter | ✓ | | | ✓ | ✓ |
| loom-mcp-operator | ✓ | ✓ | | | |
| loom-orchestrator | ✓ | ✓ | | | |
| loom-gate | ✓ | | | | |
| loom-debug | ✓ | | | | |
| loom-merge | ✓ | | | | |

## Anti-Patterns

- Never hardcode state in agent memory — read/write `docs/plan/backlog.yaml`
- Never jump from a new VP idea straight to architecture artefacts or theme/backlog planning before user alignment on the VP direction
- Never skip the troubleshooter — failed stories must be fixed before epic completion
- Never modify vision docs during Phase 4 for the **theme currently in execution** — future VPs can be amended at user checkpoints
- Never implement multiple stories in one agent session
- Never skip the code quality review at epic end
- Never mark a theme complete while `README.md`, install examples, or release-version references still describe the previous release
- Never leave a completed theme's issue templates in `.github/ISSUE_TEMPLATE/` — archive them to `ISSUE_TEMPLATE/archive/` at theme boundary so Loom only sees the current theme's epics

## Immutability Policy

Once a specification artifact is **settled** (its theme is `done` and the user checkpoint is accepted), it is **locked** and must not be modified. Later work always **extends history** by creating new artifacts with incremented numbers.

### What is locked and when

| Artifact | Locked when | Locked marker |
|:---|:---|:---|
| Vision phase `VP<n>-*/` | Corresponding theme is `done` and user-accepted | `locked: true` on the theme in `backlog.yaml` |
| Theme `TH<n>-*/` (stories, epics) | Theme status transitions to `done` and user-accepted | `locked: true` on the theme in `backlog.yaml` |
| ADR `ADR-<NNN>-*.md` | Its theme is `done` and user-accepted | Status changes from `Accepted` to `Superseded by ADR-<NNN>` only via a new ADR |

### Rules for extending locked artifacts

- **Vision**: Create `VP<n+1>-<slug>/` instead of editing `VP<n>-*/`
- **Architecture / ADRs**: Create a new `ADR-<NNN+1>` with status `Accepted` that sets the old ADR's `Status` line to `Superseded by ADR-<NNN+1>` — do not edit the body, decision, or consequences of the old ADR
- **Themes**: Create `TH<n+1>-<slug>/` with new epics and stories; reference the new vision phase in the new theme's `vision-ref`
- **Backlog**: Append new theme entries to `backlog.yaml` — never delete or rewrite entries for locked themes

### Checking lock status before planning

Before creating or editing any specification artifact, agents **must** read `docs/plan/backlog.yaml` and identify which themes have `locked: true`. Any VP, ADR, or theme directory referenced by a locked theme is off-limits for modification (except the single `Status:` line of a superseded ADR).
