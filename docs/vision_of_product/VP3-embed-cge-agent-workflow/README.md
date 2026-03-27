# VP3 — Embed CGE into Agent Workflow

> Status: discussion draft only. Do not derive architecture, ADRs, themes, or backlog items from this VP until the user explicitly confirms the vision direction.

## Vision Summary

Build the third product phase of the Cognitive Graph Engine around one practical
goal: make the graph part of the agent's normal working loop instead of an
optional side tool.

VP1 proved that a repo-local graph can hold shared memory. VP2 made that memory
healthier through stats and hygiene. VP3 should make the graph genuinely useful
for day-to-day agent craft by embedding it into task startup, task closeout, and
cross-repo adoption flows.

The graph should become something an agent naturally reaches for when entering a
task, delegating work, and handing off progress.

This VP should optimize **this repository's own agent workflow first** and treat
cross-repo reuse as the second step once the local pattern is proven.

The implementation should stay intentionally narrow and prove **one golden path**
before broadening scope.

## Product Intent

Today the graph is useful, but too manual:

- agents must remember to initialize it
- agents must decide ad hoc when to query it
- agents must decide ad hoc what to write back
- other repos do not yet have a clean, repeatable adoption path

That means the graph is available, but not yet embedded.

VP3 exists to turn CGE into a **workflow substrate**:

- easy to adopt in this repo first, then package for other repositories
- explicit at task start
- explicit at task finish
- composable across prompts, skills, and sub-agents
- still local, inspectable, and machine-readable

## Core Hypothesis

The narrow VP3 hypothesis is:

> **Graph-backed kickoff and handoff for non-trivial delegated subtasks reduce
> recovery tokens and improve delegation quality without adding more ceremony
> than they save.**

If VP3 cannot prove that hypothesis in this repo, it should not broaden into a
larger workflow platform yet.

## Primary Outcome

Turn CGE from a useful CLI into a **repeatable agent workflow layer**.

At the end of VP3, an agent should be able to:

1. wire this repo's prompts/skills/instructions so graph-backed delegation is a
   natural default for non-trivial subtasks
2. start a delegated subtask with a structured graph-backed kickoff brief
3. finish a delegated subtask with a structured graph-backed writeback and handoff
4. measure whether that workflow beats the non-graph baseline

The first success bar is not "generic installability everywhere." It is:

- this repo's agents and sub-agents use the graph naturally during work
- token-heavy restarts become less necessary because compact graph context is
  available at the right moments
- CGE adoption in this repo becomes easy enough that later packaging for other
  repos is mostly extraction, not reinvention

## Primary Users

The users remain AI agents first.

VP3 is specifically for:

- agents entering a repo and needing fast trustworthy orientation
- agents delegating work to sub-agents
- agents closing a task and preserving what changed
- maintainers who want to embed CGE into other repos without hand-assembling the
  integration

Humans may review the outputs, but the product remains optimized for machine
workflows.

## Core Jobs To Be Done

1. Let this repo's prompts, skills, and instructions actually route non-trivial
   delegated work through graph-backed kickoff and handoff.
2. Let an agent initialize and seed the graph enough that delegated subtasks have
   useful structured context to inherit.
3. Let an agent start a delegated subtask with a compact, machine-readable
   kickoff brief.
4. Let an agent finish a delegated subtask by writing back structured decisions,
   changed artifacts, and follow-up work.
5. Let the product measure whether graph-backed delegation actually reduces token
   usage and recovery cost compared with non-graph delegation.

## Product Principles

- **Workflow first**: if graph usage is not part of the normal task loop, it will
  be skipped.
- **Adoption before abstraction**: prove the workflow in this repo before
  generalizing it into a reusable pack for others.
- **Composable, not monolithic**: agents, prompts, and skills should be able to
  reuse the same start/finish workflow pieces.
- **Reproducible across repos**: adoption should not depend on one-off manual
  curation.
- **Composable snippets over heavy templates**: prefer reusable prompt, skill,
  and instruction snippets over a rigid repo takeover.
- **Lightweight automation over hidden automation**: guide agents toward natural
  graph usage with cheap nudges and convenience defaults, without opaque
  background behavior.
- **Default graph support for meaningful delegation**: most non-trivial sub-agent
  tasks should receive graph-backed kickoff and handoff support unless explicitly
  disabled.
- **Evidence over intuition**: VP3 should create a credible way to benchmark
  graph-backed work against non-graph baselines instead of only claiming token savings.
- **Explicit writeback over accidental memory**: VP3 should structure memory
  capture, not scrape everything implicitly.
- **Machine contracts over folklore**: startup briefs, writeback payloads, and
  repo bootstrap outputs should all be stable enough for automation.
- **Stay local and inspectable**: embedding the graph must not require hosted
  orchestration or opaque background services.

## VP3 Scope

### Included

- just enough repo-first bootstrap/seeding to support the golden path
- a `graph workflow start` flow focused on delegated subtask kickoff
- a `graph workflow finish` flow focused on delegated subtask handoff/writeback
- composable prompt, skill, and instruction snippets that make this delegated
  workflow the normal path
- lightweight wrappers/hooks that support the delegated-task golden path
- a benchmark approach for comparing non-trivial delegated subtasks with and
  without graph-backed workflow support

### Excluded

- always-on background daemons
- silent scraping of shell history or arbitrary agent thoughts
- hosted multi-repo orchestration services
- human GUI dashboards
- cross-repo shared graph federation in VP3
- solving every possible agent workflow in one phase

## VP3 Command Surface

VP3 should add a workflow-oriented surface:

- `graph workflow init`
- `graph workflow start`
- `graph workflow finish`

These commands should reuse VP1/VP2 primitives instead of replacing them.

The command surface should be accompanied by composable metadata snippets rather
than a rigid all-or-nothing workflow template.

VP3 may also need lightweight wrapper or hook support around this command surface
so prompts, skills, and sub-agent launches naturally invoke the right graph steps
at the right time.

## Command Intent

### `graph workflow init`

Install or refresh only the workflow support needed to make delegated graph-backed
kickoff and handoff work reliably in the current repository, initialize the
repo-local graph when needed, and seed baseline knowledge from standard repo
artifacts.

It should solve the current adoption pain in this repo first, while leaving a
clear path to later reuse in other repos.

### `graph workflow start`

Produce a delegated-subtask kickoff envelope that helps an agent answer:

- is the graph present and usable?
- should I bootstrap, query, clean up, or proceed?
- what context matters for this task right now?

This flow should combine graph state, health, and retrieval into a startup brief
that is easy to pass to sub-agents for non-trivial delegated work.

It should be cheap and convenient enough that agents use it naturally instead of
skipping straight to large prompt reconstruction.

For most non-trivial tasks, this kickoff should become the default path for
sub-agent delegation rather than a rare opt-in.

### `graph workflow finish`

Accept a structured delegated-task outcome payload and turn it into durable graph
memory.

It should preserve:

- what changed
- what was decided
- what follow-up remains
- what the next agent should know

It should make closeout cheap enough that agents and sub-agents routinely leave
behind durable graph memory rather than only conversation residue.

For most non-trivial delegated tasks, handoff should be graph-backed by default.

## Example End-to-End Workflow

1. The repo installs the minimum workflow support needed for graph-backed
   delegation and seeds baseline graph knowledge.
2. A parent agent chooses a non-trivial subtask and runs `graph workflow start`.
3. The CLI returns a compact kickoff brief for the delegated sub-agent.
4. The sub-agent performs work, using normal graph primitives as needed.
5. Before ending, the sub-agent runs `graph workflow finish`.
6. The CLI persists the handoff knowledge and returns revision anchors plus a
   next-agent brief.
7. The parent agent or a later agent resumes faster because the delegated work
   left behind compact, structured graph memory.

## Benchmarking and Scientific Validation

VP3 should include a credible way to test whether graph-backed delegated workflow
is actually useful.

The product should support benchmarking tasks in at least two modes:

1. **Without graph-backed workflow support**
   - the agent works from ordinary repo inspection and prompt history only
2. **With graph-backed workflow support**
   - the agent uses the agreed kickoff, retrieval, and handoff workflow

Key measurements should include:

- prompt or token volume consumed to reach useful task orientation
- prompt or token volume consumed across task execution and handoff
- time or step count to reach the first meaningful action
- quality or completeness of the resulting task outcome
- resumability for a later agent

The benchmark should avoid biased comparisons by:

- using the same task set in both modes
- keeping acceptance criteria constant
- recording the exact workflow used
- comparing both token usage and output quality, not token usage alone

## Success Criteria

### Priority 1

Non-trivial delegated subtasks in this repo use graph-backed kickoff and handoff
consistently because the workflow is clear, cheap, and composable.

### Priority 2

Wiring prompts, skills, and instructions to use graph-backed delegation becomes
substantially easier and more natural than it is today.

### Priority 3

Sub-agent delegation becomes cleaner because kickoff and handoff payloads are
stable enough to embed in prompts and skills without large bespoke summaries.

### Priority 4

The repo can produce defensible measurements showing whether graph-backed
delegation reduces token usage and recovery cost on representative subtasks.

## Non-Goals For VP3

- replacing existing `graph query`, `graph context`, or `graph write`
- building a hosted agent platform
- capturing all memory automatically without explicit workflow intent
- turning CGE into a generic project-template generator

## Guidance For The Architect

The architect should optimize VP3 for:

- a minimal workflow-oriented command surface centered on delegated subtasks
- reuse of existing graph, retrieval, stats, hygiene, and revision primitives
- repo-local composability of prompts, skills, instructions, and workflow metadata
- lightweight convenience automation that improves graph usage frequency without
  hiding workflow intent
- wrappers or hooks that help prompts and sub-agent launches invoke the right
  graph-backed steps naturally
- idempotent integration flows for repeated runs
- stable machine-readable contracts for kickoff and handoff
- a repo-first delegated-workflow adoption flow whose reusable parts can later be
  extracted for other repositories
- a benchmark design that can compare with-graph and without-graph task runs

The architect should avoid turning VP3 into a daemon, a hosted control plane, or
an oversized configuration system.

## Resolved Vision Direction

- Optimize this repo's own workflow first; package for other repos second.
- Prioritize wiring prompts/skills/instructions to actually use the graph, then
  make install/adoption and natural in-task usage cheap.
- Prefer composable snippets over rigid repo-wide templates.
- Add lightweight convenience automation, likely including small wrappers/hooks,
  so agents and sub-agents use the graph naturally without losing explicit control.
- Treat graph-backed kickoff and handoff as the default for most non-trivial
  delegated tasks.
- Include a scientific benchmark direction that compares token usage and task
  recovery cost with graph-backed workflow versus without it.
- Implement the benchmark in two surfaces: a repo-local evaluation harness and a
  CGE-facing report/command surface.
- Focus the first benchmark family on delegating non-trivial subtasks.

## Skeptical Review

The main risk in VP3 is overbuilding workflow machinery before proving that it
actually saves tokens and improves delegation quality.

To stay disciplined, VP3 should reject:

- broad workflow coverage beyond the delegated-subtask golden path
- wrappers/hooks that add more ceremony than saved context
- writeback rituals that produce low-value graph noise
- benchmarks that count tokens but ignore task quality

If the delegated-subtask golden path does not show clear value, the product
should pause and simplify rather than expand.

## Questions for User Discussion

At this point, the main product-direction questions are resolved enough to move
to architecture once you want to.
