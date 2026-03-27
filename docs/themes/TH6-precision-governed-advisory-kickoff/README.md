# TH6 — Precision-Governed Advisory Kickoff

## Theme Goal

Turn VP6 into an implementable backlog that makes `graph workflow start`
family-aware, confidence-gated, provenance-rich, and explicitly suppressible so
graph kickoff stays net-positive for the task families where it already works.

## Scope

This theme covers:

- task-family classification for delegated workflow start
- family-specific entity allowlists and suppressions
- no-kickoff default behavior for reporting and synthesis tasks
- family-aware confidence thresholds and abstention decisions
- one-line inclusion reasons for kickoff entities
- explicit no-kickoff and minimal-kickoff controls
- graceful degradation when the repo is sparse or the task text is ambiguous

## Out of Scope

- broad LLM-as-judge evaluation hardening
- new hosted services or remote policy engines
- expanding graph injection into reporting and synthesis by default
- replacing the existing workflow start / finish contract

## Epics

1. **TH6.E1 — Task-Family Retrieval Policies**
   Classify workflow-start requests into kickoff families and apply
   family-specific allowlists, suppressions, and abstention defaults.

2. **TH6.E2 — Confidence, Provenance, and Abstention**
   Add family-aware confidence thresholds, per-entity inclusion reasons, and
   low-confidence pull-on-demand recommendations.

3. **TH6.E3 — Freedom-Preserving Kickoff Controls**
   Add explicit no-kickoff and minimal-kickoff controls while preserving the
   machine-readable workflow contract and graceful degradation paths.

## Dependency Flow

```text
E1 → E2 → E3
```

## Success Signal

A maintainer can run `graph workflow start` on a delegated task and receive a
kickoff result that is clearly family-scoped, confidence-aware, explainable,
and suppressible, with reporting/synthesis tasks abstaining by default instead
of receiving contamination-prone graph context.

