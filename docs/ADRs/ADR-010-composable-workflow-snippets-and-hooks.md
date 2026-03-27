# ADR-010: Use composable workflow snippets and transparent wrapper/hooks

## Status
Proposed

## Context

The current adoption gap is not only graph initialization. The sharper problem is
that prompts, skills, and instructions do not reliably route non-trivial
delegated work through the graph.

VP3 needs a way to make graph-backed kickoff and handoff feel natural without:

- taking over the repository's metadata wholesale
- hiding behavior behind opaque background automation
- forcing every repo to accept the same rigid workflow layout

## Decision

Use **composable workflow snippets** plus **small transparent wrapper/hooks** as
the primary integration mechanism.

The workflow integration should:

1. install or refresh prompt/skill/instruction snippets that encourage explicit
   workflow commands
2. use small wrappers/hooks only where they reduce repeated manual steps without
   obscuring what will happen
3. preserve local overrides explicitly instead of overwriting them silently
4. keep workflow behavior inspectable and easy to opt out of

## Consequences

### Positive
- Targets the real adoption problem: wiring existing agent metadata to use the graph
- Fits the repo-first dogfooding strategy with lower adoption risk
- Makes later extraction for other repos easier because the pieces are modular
- Avoids the trust cost of hidden automation

### Negative
- Snippet-based integration can become fragmented if not curated carefully
- Hook/wrapper behavior still needs strong documentation and refresh rules
- Adoption may be slower than with a rigid one-shot template takeover

### Risks
- Risk: wrappers/hooks may add more ceremony than the tokens they save
  - Mitigation: keep them thin, optional, and benchmark the delegated workflow
    against a no-graph baseline

## Alternatives Considered

### Rigid repo-wide metadata takeover
- Pros: stronger standardization, fewer local choices
- Cons: invasive, harder to preserve repo conventions, higher correction cost
- Rejected because: VP3 should solve adoption with the lightest viable touch

### Purely manual documentation with no wrappers/hooks
- Pros: maximum transparency, minimum automation complexity
- Cons: unlikely to make graph usage natural enough during real delegated work
- Rejected because: the product goal is natural usage, not only written guidance

### Hidden automatic interception of sub-agent launches
- Pros: graph usage might happen more consistently
- Cons: opaque behavior, weak trust, difficult debugging
- Rejected because: VP3 must remain explicit and inspectable
