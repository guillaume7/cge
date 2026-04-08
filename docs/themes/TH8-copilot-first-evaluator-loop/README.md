# TH8 — Copilot-First Evaluator Loop and Decision Harness

## Theme Goal

Turn VP8 into an implementable backlog that builds the missing evaluator loop,
decision engine, attribution layer, evaluated memory discipline, and
harness-aware lab conditions so CGE becomes a useful Copilot-first augmentation
layer instead of a graph-centric memory tool.

## Scope

This theme covers:

- a Context Evaluator that scores candidate context bundles and candidate task
  outputs for relevance, consistency, and likely usefulness (ADR-018)
- a Decision Engine that selects one of five normalized outcomes — continue,
  minimal, abstain, backtrack, write — based on evaluator confidence (ADR-019)
- attribution-first decision records that explain every evaluator-loop decision
  and persist evidence for lab analysis (ADR-021)
- Copilot CLI integration: wiring the evaluator loop into `graph context` and
  `graph workflow start` without breaking backward compatibility (ADR-020)
- evaluated graph memory discipline: gating workflow-mediated writes on evaluator
  confidence and down-ranking stale graph state during retrieval (ADR-022)
- harness-aware lab conditions and token-decline measurement including
  over-abstention detection (ADR-023)

## Out of Scope

- hosted control planes or mandatory remote services
- LLM-based scoring for the evaluator (local heuristics only in VP8)
- graph schema changes or new node/relationship types
- broad human UI/dashboard work
- generic multi-agent platform ambitions
- replacing the existing raw `graph write` primitive

## Epics

1. **TH8.E1 — Context Evaluator Foundation**
   Build the in-process Go component that scores candidate context and candidate
   outputs on relevance, consistency, and usefulness dimensions.

2. **TH8.E2 — Normalized Decision Outcomes**
   Translate evaluator confidence scores into the five normalized outcomes with
   configurable thresholds and machine-readable decision envelopes.

3. **TH8.E3 — Attribution and Copilot CLI Integration**
   Generate, persist, and retrieve attribution records and wire the full
   evaluator loop into the existing `graph context` and `graph workflow start`
   CLI commands.

4. **TH8.E4 — Evaluated Graph Memory Discipline**
   Gate workflow-mediated memory writes on evaluator confidence, down-rank stale
   graph state during retrieval, and preserve raw write backward compatibility.

5. **TH8.E5 — Harness-Aware Lab Conditions**
   Extend the experiment lab to support harness-aware conditions, surface
   token-decline as a primary metric, detect over-abstention, and aggregate
   attribution records in lab reports.

## Dependency Flow

```text
E1 → E2 → E3
            ↘
       E2 → E4
            ↘
       E3 + E4 → E5
```

E3 and E4 both depend on E2 and may be worked in parallel. E5 depends on both
E3 and E4.

## Success Signal

Lab experiments run through `graph lab run` with harness-aware conditions show a
measurable decline in token consumption when using the full CGE evaluator loop
compared to without-harness and graph-only baselines, without over-abstention
or quality regression. Every decision path can explain why context was injected,
minimized, rejected, or persisted.
