---
description: "Brainstorm and design the product vision interactively. Use when: starting a new product, defining MVP, brainstorming features, defining vision phases."
agent: "ask"
---

## Agents & Skills

| Agent | Skills |
|-------|--------|
| Interactive (ask) | `the-copilot-build-method` |

Let's design the product vision together. I'll help you brainstorm and capture ideas in `docs/vision_of_product/`.

## Pre-flight: Check for existing locked artefacts

Before creating any VP directory, read `docs/plan/backlog.yaml` (if it exists) and identify:
- Which VP directories already exist under `docs/vision_of_product/`
- Which themes have `locked: true` in the backlog (their referenced VP dirs are **immutable**)

**Rule**: Never edit a VP directory whose corresponding theme is `locked: true`. Instead, create the next `VP<n+1>-<slug>/` directory to capture new vision work.

If locked VPs exist, inform the user which vision phases are already settled and propose the next available VP number for new ideas.

## Structure

Each vision phase maps to one or more implementation themes (1:N):
- `VP1-mvp/` → `TH1` — your minimum viable product
- `VP2-<feature>/` → `TH2`, `TH3` — larger phases can produce multiple themes
- Theme numbering is sequential and independent of VP numbering

## What to capture in each VP<n> directory

Create markdown files covering:
- **Problem statement**: What pain point does this solve?
- **Target users**: Who are the primary users/personas?
- **Core features**: What must this phase deliver?
- **Success criteria**: How do we know this phase is done?
- **Constraints**: Budget, timeline, technology, compliance
- **Open questions**: Unknowns to resolve before implementation

## Let's start

For a **new VP**, do not jump straight to architecture or themes.

First:
1. Restate your understanding of the product intent in your own words
2. Pitch one or more candidate directions or framing options
3. Ask focused questions that reduce product ambiguity
4. Wait for user alignment before writing downstream artefacts beyond the VP discussion draft

Then ask:
1. What is the product you want to build?
2. What problem does it solve?
3. Who is it for?

I'll help you structure the answers into VP directories as we go.
