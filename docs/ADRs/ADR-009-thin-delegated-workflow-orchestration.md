# ADR-009: Add a thin delegated-workflow orchestration layer over existing graph primitives

## Status
Proposed

## Context

VP3 is intentionally narrowed to one golden path:

- graph-backed kickoff for non-trivial delegated subtasks
- graph-backed handoff/writeback for those subtasks
- benchmark evidence that this path reduces recovery-token cost without harming
  task quality

The existing system already has the core primitives needed to support that path:

- graph initialization
- graph persistence and revisions
- retrieval, context projection, and explanation
- graph stats and hygiene

The question is whether VP3 should introduce a larger workflow platform or a
thin orchestration layer that composes the existing primitives.

## Decision

Implement VP3 as a **thin delegated-workflow orchestration layer** that adds:

- `graph workflow init`
- `graph workflow start`
- `graph workflow finish`

and composes the existing graph primitives behind those flows.

The workflow layer should:

1. stay focused on non-trivial delegated subtasks
2. assemble kickoff and handoff envelopes from existing retrieval, stats,
   hygiene, write, and revision capabilities
3. remain explicit and inspectable
4. avoid becoming a general-purpose workflow engine in VP3

## Consequences

### Positive
- Reuses the strongest existing graph capabilities instead of duplicating them
- Keeps VP3 proportional to the narrowed delegated-subtask hypothesis
- Makes the benchmark easier to interpret because the new layer is small and
  explicit
- Preserves a clear path to future expansion if the golden path proves valuable

### Negative
- Adds orchestration glue that must be maintained carefully
- May expose seams between existing primitives that were previously acceptable
- Risks under-serving broader workflow use cases in the short term

### Risks
- Risk: the thin layer still becomes bloated if every adjacent workflow concern
  is pulled into VP3
  - Mitigation: keep the contract centered on delegated kickoff/handoff only and
    reject unrelated workflow expansion during VP3

## Alternatives Considered

### Broad workflow platform in VP3
- Pros: could cover more agent lifecycle scenarios immediately
- Cons: higher complexity, weaker focus, harder to benchmark honestly
- Rejected because: VP3 must prove one delegated-subtask golden path first

### No new workflow layer; rely on ad hoc composition only
- Pros: zero new orchestration code
- Cons: fails to solve the current pain of prompts/skills/instructions not using
  the graph naturally
- Rejected because: VP3 exists specifically to make that workflow explicit and
  easy to adopt
