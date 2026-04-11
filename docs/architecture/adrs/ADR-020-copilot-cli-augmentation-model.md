# ADR-020: Copilot CLI augmentation model

## Status
Proposed

## Context

Through VP1–VP7, CGE has been positioned as a standalone graph-memory CLI. It
provides graph persistence, retrieval, workflow orchestration, and experiment
infrastructure, but the product interface assumes that consuming agents invoke
`graph` commands as a self-contained toolchain.

VP8 reframes CGE around one practical observation: the primary execution
environment for autonomous agents is the Copilot CLI harness. That harness
already owns user interaction, tool routing, session management, and repository
work. CGE should augment that harness rather than compete with it or stand
apart from it.

The VP8 vision states:

> CGE should add the layer that is currently missing: candidate-context
> retrieval, evaluation and critique, decision and backtracking discipline,
> attributed memory updates.

This ADR establishes the product integration model for VP8.

## Decision

Adopt a **Copilot CLI augmentation model** where:

1. **Copilot CLI remains the host runtime**: Copilot CLI owns the main
   interaction loop with the user, tool execution, repository work, and session
   management. CGE does not replicate or replace any of those responsibilities.

2. **CGE provides local capabilities**: CGE provides four capabilities that the
   Copilot CLI harness consumes:
   - **retrieval** — graph-backed and text-relevance candidate context
   - **evaluation** — scoring of candidate context and candidate outputs
   - **decision** — outcome selection (continue/minimal/abstain/backtrack/write)
   - **memory** — attributed persistence of useful state across sessions

3. **CLI command interface**: the existing `graph` CLI commands remain the
   integration surface. Copilot CLI agents invoke `graph context`,
   `graph workflow start`, `graph workflow finish`, and similar commands as
   tool calls. No daemon, socket, or API server is introduced.

4. **No platform ambition in VP8**: CGE does not become a generic agent runtime
   or multi-agent platform. It stays a local augmentation layer that proves
   value through one host environment before broadening.

5. **Existing surfaces preserved**: all existing `graph` commands continue to
   work. VP8 enhances `graph context` and `workflow start` with evaluation and
   decision stages but does not break backward compatibility for consumers
   that do not use the new decision metadata.

## Consequences

### Positive
- Clear product boundary: CGE adds quality-control intelligence, Copilot CLI
  owns execution
- The product proves value in one real environment before generalizing
- No new infrastructure (daemon, API server) is needed
- Existing integrations continue to work

### Negative
- CGE's value depends on the consuming agent correctly invoking its commands;
  agents that skip `graph context` or ignore decision metadata do not benefit
- The product cannot control or observe the full Copilot CLI session lifecycle
- VP8 value is bounded by the capabilities available through CLI-based tool
  invocation

### Risks
- Risk: Copilot CLI changes its tool-calling interface and breaks CGE
  integration
  - Mitigation: CGE's interface is stdin/stdout JSON; this is stable regardless
    of how the host invokes it
- Risk: the augmentation model is too passive and CGE is ignored in practice
  - Mitigation: lab experiments must demonstrate measurable improvement; if the
    augmentation model does not show value, the product should simplify rather
    than add ceremony
- Risk: VP8 scope creeps toward building a generic agent platform
  - Mitigation: this ADR explicitly limits VP8 to one host environment

## Alternatives Considered

### Build CGE as a standalone agent runtime
- Pros: full control over the agent lifecycle
- Cons: duplicates Copilot CLI capabilities, violates VP8 "augment not replace"
  principle
- Rejected because: VP8 should prove value in the existing workflow before
  building a new one

### Run CGE as a background daemon with an API
- Pros: lower latency, richer session management
- Cons: adds deployment complexity, breaks local-first CLI simplicity
- Rejected because: CLI-based invocation is sufficient for VP8 and aligns with
  the existing product shape

### Integrate CGE as a Copilot CLI plugin or extension
- Pros: tighter integration, automatic invocation
- Cons: couples CGE to Copilot CLI internals, reduces portability
- Rejected because: loose coupling through CLI tool calls is more portable and
  easier to test
