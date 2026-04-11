# Copilot CLI Harness Augmentation — VP8

## Intent

VP8 should augment the Copilot CLI harness rather than compete with it.

The host workflow already knows how to:

- take user requests
- route work through tools and sub-agents
- preserve session context
- execute repository tasks

CGE should add the layer that is currently missing:

- candidate-context retrieval
- evaluation and critique
- decision and backtracking discipline
- attributed memory updates

## Host Model

The preferred VP8 shape is:

1. **Copilot CLI remains the host runtime**
2. **CGE provides local retrieval, scoring, and memory capabilities**
3. **The evaluator loop decides how strongly CGE should influence the task**

This keeps VP8 narrow enough to prove value before broadening into a more generic
agent runtime.

## Component Roles

### Copilot CLI harness

- owns the main interaction loop with the user
- owns tool execution and repository work
- remains the primary execution shell for autonomous agents

### CGE retrieval layer

- retrieves candidate graph-backed context
- may combine graph memory with other local signals
- should prefer relevance over raw breadth

### Evaluator layer

- scores the candidate context and candidate outputs
- checks whether the graph is helping, neutral, or harmful
- provides a basis for continue/backtrack/minimize/abstain decisions

### Decision layer

- determines whether to inject, trim, reject, or revise guidance
- decides when memory should be updated
- prevents graph drift from becoming automatic product truth

### Memory layer

- preserves useful context across sessions
- stores attribution and trust signals, not just raw artifacts
- supports continuity without taking over the entire loop

## Preferred Workflow Shape

```text
task -> retrieve candidates -> evaluate -> decide -> act -> update memory -> verify in lab
```

This is intentionally different from the older product shape:

```text
task -> retrieve graph context -> assume usefulness -> act
```

## What VP8 Should Improve

VP8 should improve the Copilot CLI workflow by making it more disciplined about:

- when graph-backed context is injected
- how much graph-backed context is injected
- whether current memory is still trustworthy
- whether a current attempt should be revised or rejected
- how decisions are explained to later experiments and later agents

## Guardrails

- Do not require hosted infrastructure for the core path.
- Do not make Copilot CLI a thin wrapper around CGE.
- Do not reintroduce graph-centric drift by treating stored memory as
  automatically correct.
- Do not add so much control-loop ceremony that token savings disappear.

## Success Signal

The Copilot-hosted path should be measurably better with CGE than without it,
especially on token consumption and context relevance, while preserving useful
task outcomes.
