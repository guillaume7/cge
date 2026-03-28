# TH7 — Verification-Calibrated Audit Kickoff

## Theme Goal

Turn VP7 into an implementable backlog that narrows and disciplines
verification-family kickoff behavior so stats audits and workflow verification
stop erasing the gains achieved by TH6.

## Scope

This theme covers:

- verification/audit sub-profile routing
- stats-specific and workflow-verification-specific suppressions
- tighter verification-family confidence thresholds and token budgets
- verification-family minimal/abstain downgrade paths
- targeted rerun attribution capture for workflow-start and baseline prompt
  surfaces
- a verification-focused rerun gate before any broader full rerun

## Out of Scope

- broad redesign of write-producing, reporting, or troubleshooting families
- a full decision-grade rerun before the targeted verification gate passes
- hosted experiment telemetry or remote policy infrastructure
- replacing the existing workflow-start machine-readable contract

## Epics

1. **TH7.E1 — Verification-Specific Kickoff Policies**
   Split verification/audit work into narrower profiles and apply the right
   suppressions and entity policies for each one.

2. **TH7.E2 — Verification Confidence and Downgrade Discipline**
   Tighten verification-family thresholds, token discipline, and downgrade logic
   so borderline audit tasks minimize or abstain instead of over-injecting.

3. **TH7.E3 — Verification Rerun Attribution and Gate**
   Capture richer per-run attribution and use a targeted rerun to decide whether
   the full campaign budget is justified.

## Dependency Flow

```text
E1 → E2 → E3
```

## Success Signal

A maintainer can run `graph workflow start` on verification-oriented tasks and
see a narrower, more honest kickoff policy that keeps stats audits and
workflow-verification near neutral or better, with enough rerun attribution to
justify whether a full rerun should happen next.
