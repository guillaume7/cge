---
description: "Transform product vision into architecture and implementation plan. Runs architect then product-owner agents sequentially. Use when: planning implementation, generating backlog, creating architecture from vision."
agent: "agent"
tools: [read, edit, search, agent, todo, execute, web, github/github-mcp-server/default]
---

Run the planning pipeline to transform approved vision into architecture and a
delivery backlog.

## Pre-flight: Check for locked artefacts

Before invoking any agent, read `docs/plan/backlog.yaml` (if it exists):
1. Identify themes with `locked: true` — their VP dirs, theme dirs, and ADRs are **immutable**
2. Identify the highest existing VP number and theme number so new work uses the correct next increment
3. Report to the user which themes/VPs are already settled and what the next available numbers are

**Rule**: Never edit VP directories, theme directories, or ADRs that are referenced by a locked theme. New architecture work creates new `ADR-<NNN+1>` documents; new planning creates new `TH<n+1>` themes.

If the user's request is still at the **new VP ideation** stage, or if the target VP has not yet been explicitly aligned with the user, stop and redirect to `/kickstart-vision` behavior first. Do **not** invoke the architect or product-owner to create ADRs, themes, or backlog artefacts yet.

## Pipeline

### Step 1 — Architecture
Invoke the @architect agent to analyze `docs/vision_of_product/` and produce:
- `docs/architecture/` — system design, tech stack, components
- `docs/architecture/adrs/` — architecture decision records

The @architect must **not** modify any ADR that belongs to a locked theme, except to update its `Status` line to `Superseded by ADR-<NNN>` when creating a new ADR that supersedes it.

### Step 2 — Planning
Invoke the @product-owner agent to break the vision + architecture into:
- `docs/themes/TH<n>-<name>/` — theme/epic/story hierarchy
- `docs/plan/backlog.yaml` — YAML dependency graph with all stories

> **Phase gate**: Step 2 is only allowed after Step 1 has produced architecture/ADR output for the agreed VP. Never go directly from a fresh VP idea to story/backlog generation.

After both steps complete, display a summary of:
- Number of themes, epics, and stories created
- Dependency graph overview
- Estimated implementation order
