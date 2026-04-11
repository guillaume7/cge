# VP8 — CGE 2 New Architecture

> Status: discussion draft only. This VP reframes CGE around a Copilot-first
> control loop and should not yet be treated as settled architecture or backlog
> scope.

## Vision Summary

Build the eighth product phase of the Cognitive Graph Engine around one practical
goal: turn CGE into a useful augmentation layer for the Copilot CLI harness
instead of treating the graph as the product by itself.

The current product drifted away from its intent. The graph database became less
reliable over time, graph-backed behavior diverged, and the system lacks the
evaluator loop needed to decide whether retrieved context or generated guidance
is actually helping. VP8 exists to correct that failure.

The new direction is **Copilot-first, extensible later**:

- CGE should improve autonomous agents operating through Copilot CLI
- graph memory should remain important, but as one subsystem among several
- evaluation and decision-making should become first-class instead of implicit
- the system should stay local-first and implemented in Go

## Product Intent

The original CGE bet too heavily on memory structure without building the loop
that keeps that memory trustworthy and task-useful.

That shows up in two concrete failures:

- the graph store can diverge and become less reliable over time
- task execution lacks an explicit evaluator/critique loop that can reject,
  minimize, backtrack, or refine poor context and poor outputs

VP8 exists to make CGE useful in the actual Copilot CLI workflow:

- orient an agent with narrower, better-scored context
- reduce irrelevant context injection
- preserve strong session-to-session continuity
- expose clear attribution for why guidance was injected, minimized, or rejected
- prove value through lab experiments instead of intuition

## Core Hypothesis

The narrow VP8 hypothesis is:

> **If CGE becomes a Copilot-first control loop with explicit evaluation,
> decision, and attribution, then graph-backed task execution will use fewer
> tokens, inject less irrelevant context, and produce more trustworthy continuity
> than the current graph-centric approach.**

If VP8 cannot demonstrate those improvements in the lab, the product should
simplify again instead of adding more subsystems.

## Primary Outcome

Turn CGE from a graph-led memory tool into a **local task-quality harness** for
autonomous agents working through Copilot CLI.

At the end of VP8, the product should be able to:

1. retrieve candidate context without assuming retrieval is automatically correct
2. score candidate context and candidate outputs before trusting them
3. decide whether to continue, backtrack, minimize, abstain, or write memory
4. update memory with stronger attribution and tighter trust signals
5. show, through lab evidence, that the harness reduces token consumption while
   keeping or improving useful task outcomes

## Primary Users

The primary users are autonomous agents operating through Copilot CLI.

VP8 is specifically for:

- agents entering a task and needing compact, better-scored context
- agents that need a critique loop before treating generated output as good
- maintainers running lab experiments to verify whether the CGE harness is
  actually improving token efficiency and task usefulness

Humans remain supervisors and reviewers, but VP8 is still optimized for machine
workflows first.

## Core Jobs To Be Done

1. Let an agent retrieve candidate context without overcommitting to noisy graph
   memory.
2. Let an agent evaluate relevance, consistency, and likely usefulness before
   acting on retrieved context.
3. Let an agent iteratively generate, critique, and revise task guidance instead
   of trusting the first pass.
4. Let the harness decide when to continue, backtrack, minimize, abstain, or
   persist new memory.
5. Let graph memory stay useful as a subsystem without letting stale graph state
   dominate the entire product.
6. Let experiments explain why graph-backed guidance helped, regressed, or was
   suppressed.
7. Let session-to-session continuity improve without requiring large prompt
   reconstruction.

## Product Principles

- **Evaluation before trust**: no context source, including the graph, should be
  treated as self-validating.
- **Copilot-first host model**: VP8 should improve the Copilot CLI harness before
  it broadens into a more generic runtime.
- **Graph as subsystem, not sovereign**: memory matters, but it should no longer
  dictate the product shape by itself.
- **Local-first by default**: the core loop should not require hosted services.
- **Go-native implementation**: the implementation direction should stay in Go.
- **Attribution is load-bearing**: the harness must explain why guidance was
  injected, minimized, rejected, or persisted.
- **Cheap correction beats confident drift**: backtracking, abstaining, or
  minimizing are valid outcomes when confidence is weak.
- **Evidence over folklore**: lab results, especially token consumption and task
  outcome evidence, should drive whether VP8 is considered successful.

## VP8 Scope

### Included

- evaluation and scoring as a first-class subsystem
- decision/backtracking behavior around context and output quality
- context retrieval that can be narrowed, scored, and suppressed
- memory writes and updates with stronger attribution and trust signals
- Copilot CLI workflow integration as the primary host environment
- experiment/lab support to verify token and usefulness improvements

### Excluded

- hosted control planes or mandatory remote services
- graph-first product positioning where memory alone defines the system
- broad human UI/dashboard work
- uncontrolled growth into a generic multi-agent platform in VP8
- treating every graph artifact as trustworthy just because it exists

## Product Surface Intent

VP8 should keep the existing product recognizable while changing what is
considered load-bearing.

### Copilot CLI integration

The Copilot CLI harness should become the main environment where CGE proves its
value. CGE should help decide:

1. what context to retrieve
2. whether that context is good enough to trust
3. whether the current attempt should continue or backtrack
4. what memory should be updated afterward

### Existing graph surfaces

Existing graph retrieval and write surfaces can remain important, but they should
act as supporting primitives inside a broader control loop rather than standing
in for the product.

### Lab and verification surfaces

VP8 should make experiment outputs rich enough to compare:

- token consumption with and without the CGE harness
- quality or usefulness of resulting task guidance
- attribution for why a retrieval or decision path succeeded or regressed

## Success Criteria

- lab experiments show a clear decline in token consumption when using the CGE
  harness
- graph-backed context injection becomes narrower and less irrelevant
- the product can explain why a decision path continued, minimized, abstained, or
  backtracked
- session continuity improves without depending on large prompt restarts
- graph reliability improves because memory is evaluated and updated under a more
  disciplined loop instead of silently drifting

## Non-Goals For VP8

- proving a fully generic agent runtime before the Copilot-first host model works
- expanding graph schema complexity as a substitute for evaluator quality
- building a hosted memory service
- optimizing for human graph browsing over agent task quality

## Guidance For The Architect

The architect should optimize VP8 for:

- a Go-native, local-first control loop
- explicit evaluator and decision stages
- better-scored retrieval and memory update discipline
- clear Copilot CLI integration points
- experiment design that can demonstrate token decline without hiding quality
  regressions
- a product shape where graph memory is important but no longer the sole center
  of gravity

The architect should avoid treating the graph database as inherently trustworthy
or overfitting the design around storage concerns while the evaluator loop
remains weak.

## Resolved Vision Direction

- Anchor VP8 in a **hybrid, Copilot-first** framing.
- Treat autonomous agents operating through Copilot CLI as the primary users.
- Make the missing evaluator loop the main product failure to correct.
- Treat graph memory as one subsystem among several equal or near-equal parts.
- Keep context retrieval, evaluation/scoring, decision/backtracking, memory
  updates, and lab verification in scope.
- Keep the implementation local-first and in Go.
- Use token-decline evidence in lab runs as the main success signal.

## Open Questions

- What is the smallest command or API surface that exposes the evaluator loop
  without turning VP8 into a brand-new platform all at once?
- How should the harness score output quality in a way that is credible for both
  coding and verification-oriented tasks?
- Which existing graph assets should be trusted, downgraded, or rewritten during
  the transition into the VP8 loop?
