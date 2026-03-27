# TH3 — Graph-Backed Delegated Workflow

## Theme Goal

Turn VP3 into an implementable backlog that makes graph-backed kickoff and
handoff the normal path for non-trivial delegated subtasks in this repository.

## Scope

This theme covers:

- `graph workflow init` for repo-local bootstrap, workflow assets, and baseline
  graph seeding
- `graph workflow start` for delegated-subtask readiness checks, retrieval, and
  kickoff brief generation
- `graph workflow finish` for structured writeback, revision-aware persistence,
  and next-agent handoff
- composable prompt, skill, instruction, and wrapper/hook assets that make the
  delegated workflow natural in this repo
- local benchmark scenarios and machine-readable reports that compare with-graph
  and without-graph delegated workflows

## Out of Scope

- broad workflow automation beyond delegated-subtask kickoff and handoff
- hidden background daemons or silent graph mutation
- hosted telemetry or observability platforms for benchmark collection
- multi-repo graph federation
- generic reusable packaging before the repo-first workflow is proven here

## Epics

1. **TH3.E1 — Workflow Bootstrap and Assets**
   Add `graph workflow init`, baseline graph seeding, and composable workflow
   asset installation so this repo can adopt the delegated workflow cleanly.

2. **TH3.E2 — Delegated Kickoff and Handoff**
   Add `graph workflow start` and `graph workflow finish` so delegated subtasks
   begin and end with structured graph-backed envelopes.

3. **TH3.E3 — Benchmark and Repo Dogfooding**
   Add benchmark surfaces plus repo-local workflow verification so VP3 proves the
   delegated-workflow hypothesis with evidence instead of intuition.

## Dependency Flow

```text
E1 → E2 → E3
```

## Success Signal

An agent in this repo can initialize workflow support once, start a non-trivial
subtask with a compact graph-backed kickoff brief, finish with structured
writeback and handoff, and compare that path against a non-graph baseline using
local benchmark evidence.
