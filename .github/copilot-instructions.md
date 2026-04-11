# Copilot Workspace Guide

## Lifecycle

This repository uses a single local execution path:

| Phase | Prompt | Primary output |
| --- | --- | --- |
| Vision | `/kickstart-vision` | `docs/vision_of_product/VP<n>-<slug>/` |
| Architecture | `/plan-product` | `docs/architecture/` + `docs/architecture/adrs/` |
| Planning | `/plan-product` | `docs/themes/TH<n>-<slug>/` + `docs/plan/backlog.yaml` |
| Execution | `/run-autopilot` | Local implement → test → review loop |

## Active Agents

| Agent | Role |
| --- | --- |
| `architect` | Produces architecture docs and ADRs |
| `product-owner` | Produces themes, stories, and backlog state |
| `orchestrator` | Runs the autopilot loop |
| `developer` | Implements one story |
| `reviewer` | Reviews correctness, security, and conventions |
| `troubleshooter` | Fixes failed stories |

## Docs Layout

- `docs/vision_of_product/` — vision phases
- `docs/architecture/` — architecture overview, components, tech stack, setup
- `docs/architecture/adrs/` — architecture decision records
- `docs/themes/` — implementation themes, epics, and stories
- `docs/plan/` — backlog, session log, changelog, release notes

## Core State

- `docs/plan/backlog.yaml` is the single source of truth for execution state.
- `docs/plan/session-log.md` is the short rolling execution log.

## Repo-local delegated workflow

For non-trivial delegated subtasks in this repository:

- inspect `.graph/workflow/assets/` when you need the installed workflow assets
- start with `bash .github/hooks/scripts/repo-delegated-workflow.sh kickoff --task "<task>"`
  or explicit `graph workflow init` plus `graph workflow start`
- finish with `bash .github/hooks/scripts/repo-delegated-workflow.sh handoff --file task-outcome.json`
  or explicit `graph workflow finish --file task-outcome.json`
- honor opt-outs with `--opt-out` or `CGE_REPO_WORKFLOW_OPTOUT=1`

## Skills

| Topic | Skill |
| --- | --- |
| Lifecycle and conventions | `the-copilot-build-method` |
| Architecture decisions | `architecture-decisions` |
| Planning and backlog state | `backlog-management` |
| Story format | `bdd-stories` |
| Code review | `code-quality` |

## Rules

- Always read and write execution state through `docs/plan/backlog.yaml`.
- Stay in vision work until the user aligns on the VP direction.
- Do not edit artefacts that belong to a locked theme.
- Do not skip troubleshooting for failed stories.
- Do not close a theme with stale public docs.
