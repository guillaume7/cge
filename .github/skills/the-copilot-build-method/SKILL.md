---
name: the-copilot-build-method
description: 'The overarching autonomous product development methodology. Covers the 4-phase lifecycle (Vision → Architecture → Planning → Autopilot), VP↔TH mapping, Directory conventions, Definition of Done, agent squad roles, and lifecycle ceremonies. Use when: understanding the methodology, onboarding to the process, checking conventions, verifying Definition of Done.'
---

# The Copilot Build Method

This repository uses a lean, local-first product workflow built around six
agents and one execution path.

## Phases

| Phase | Prompt | Output |
| --- | --- | --- |
| Vision | `/kickstart-vision` | `docs/vision_of_product/VP<n>-<slug>/` |
| Architecture | `/plan-product` | `docs/architecture/` + `docs/architecture/adrs/` |
| Planning | `/plan-product` | `docs/themes/TH<n>-<slug>/` + `docs/plan/backlog.yaml` |
| Execution | `/run-autopilot` | local implement → test → review loop |

## Core principles

- Vision first: do not plan implementation before the user aligns on the VP.
- Architecture before code: significant technical choices go into ADRs.
- BDD-driven planning: stories need explicit acceptance criteria and scenarios.
- Persistent state: execution status lives in `docs/plan/backlog.yaml`.
- Release hygiene: a theme is not done while public docs lag behind reality.

## Agent roles

| Agent | Responsibility |
| --- | --- |
| `architect` | architecture docs and ADRs |
| `product-owner` | themes, epics, stories, backlog |
| `orchestrator` | execution sequencing and theme closure |
| `developer` | one-story implementation and tests |
| `reviewer` | correctness, security, and convention review |
| `troubleshooter` | failed-story diagnosis and repair |

## Definition of done

### Story
1. Acceptance criteria are covered.
2. Relevant tests pass.
3. Build or lint expectations still hold.
4. Documentation is updated when behavior changes.

### Epic
1. All stories are done.
2. Epic-level validation has run.
3. `docs/plan/CHANGELOG.md` captures the delivered change.

### Theme
1. All epics are done.
2. Full validation passes.
3. `README.md`, install guidance, and release references match the shipped state.
4. `docs/plan/RELEASE-<theme-id>.md` exists.
5. The user accepts the checkpoint and the theme becomes `locked: true`.

## Naming conventions

| Entity | Pattern |
| --- | --- |
| Vision phase | `VP<n>-<slug>/` |
| Theme | `TH<n>-<slug>/` |
| Epic | `E<m>-<slug>/` |
| User story | `US<l>-<slug>.md` |
| ADR | `ADR-<NNN>-<slug>.md` |

## Locked artefacts

Once a theme is user-accepted and marked `locked: true`, its vision files,
theme files, and ADR bodies are frozen. Extend history with new VP, theme, or
ADR numbers instead of rewriting settled work. The only allowed edit to a
locked ADR is changing its `Status` line when a newer ADR supersedes it.
