# TH3.E1 — Workflow Bootstrap and Assets

## Epic Goal

Make graph-backed delegated workflow easy to adopt in this repo by adding a
workflow bootstrap command, baseline graph seeding, and composable workflow
assets that remain transparent and refreshable.

## Stories

- `TH3.E1.US1` — Add `graph workflow init` and a workflow asset manifest
- `TH3.E1.US2` — Seed baseline repo graph knowledge during workflow init
- `TH3.E1.US3` — Install and refresh composable workflow assets while preserving repo overrides

## Done When

- `graph workflow init` can bootstrap or refresh repo-local delegated-workflow support
- standard repo artifacts can seed baseline graph knowledge without inventing a second memory store
- prompt, skill, instruction, and wrapper/hook assets are installed transparently and remain inspectable
