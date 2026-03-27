# VP5 — Token-Instrumented Lab

> Status: draft. This VP exists to repair the measurement-integrity gap exposed by VP4 dogfooding.

## Vision Summary

Build the fifth product phase of the Cognitive Graph Engine around one practical
goal: make the experiment lab's token telemetry scientifically honest.

VP4 gave CGE a local evidence-producing workflow lab, but dogfooding exposed a
critical flaw: the ledger and reports were treating placeholder token counts as
if they were authoritative measurements. VP5 exists to close that gap.

The result should be a lab that records **measured**, **partial**, or
**unavailable** execution telemetry explicitly, never guessed totals.

## Product Intent

We are no longer trying to answer only:

- does the graph reduce total token usage?
- does it improve resumability without harming quality?

We must first answer a stricter question:

- are the reported token deltas grounded in real execution data?

Without honest measurement provenance, even a well-designed experiment can
produce misleading conclusions.

VP5 exists to make the lab trustworthy enough that a null result, a positive
result, or a negative result all mean what they appear to mean.

## Core Hypothesis

The narrow VP5 hypothesis is:

> **If the lab records execution-layer token telemetry with explicit provenance
> and completeness, then CGE experiments can distinguish measured efficiency
> effects from missing-data artifacts without fabricating confidence.**

## Primary Outcome

Turn `graph lab` into a workflow-science surface that can say:

1. this token metric was measured end-to-end
2. this token metric is partial and why
3. this run cannot support token comparison yet

## Primary Users

VP5 is for:

- maintainers running local CGE experiments and needing defensible evidence
- agents or orchestrators that can emit execution usage in outcome payloads
- reviewers deciding whether a reported token delta is scientifically usable

## Core Jobs To Be Done

1. Let a run ingest authoritative usage from the execution layer when available.
2. Let the run ledger preserve measurement provenance and completeness state.
3. Let reports exclude incomplete token telemetry from strict token comparisons.
4. Let the system keep quality and resumability analysis available even when
   token telemetry is partial or unavailable.
5. Let future providers plug into the same ledger contract without rewriting the
   reporting model.

## Product Principles

- **Never fabricate token totals**: guessed numbers are worse than missing data.
- **Missingness is data**: partial or unavailable telemetry must be recorded
  explicitly and surfaced in reports.
- **Provider-agnostic ledger, narrow first collector**: the schema should not
  assume one runtime forever, but the first implementation can target one path.
- **Execution-layer evidence over post-hoc inference**: usage should come from
  outcome artifacts or runtime responses, not from string lengths or heuristics.
- **Quality analysis still matters**: incomplete token telemetry must not erase
  useful quality and resumability evidence.

## VP5 Scope

### Included

- a provider-agnostic run-telemetry contract with explicit completeness states
- ingestion of execution usage from delegated outcome payload metadata
- removal of synthetic token generation from `graph lab run`
- report warnings and limitations for incomplete token telemetry
- a Copilot-CLI-first integration slice for lab runs

### Excluded

- broad multi-provider runtime integrations in the first slice
- hosted telemetry backends
- hidden fallbacks that silently estimate token totals
- changing the evaluation/judgment separation introduced in VP4

## Command Intent

### `graph lab run`

VP5 should allow a single run to carry a real delegated outcome payload whose
metadata includes execution usage. If such usage is present, the ledger records
it as measured telemetry. If it is absent, the ledger records an explicit
partial or unavailable status instead of synthetic numbers.

### `graph lab report`

VP5 should make token-comparison summaries conditional on telemetry
completeness. A report may still compare quality, success, and resumability
while warning that token metrics are incomplete.

## Success Criteria

- A run record can distinguish `complete`, `partial`, and `unavailable` token
  telemetry.
- `graph lab run` no longer writes fabricated token totals.
- `graph lab report` does not treat missing token values as zero.
- The repo can ingest a delegated outcome payload carrying execution usage and
  persist those values into the run ledger.
