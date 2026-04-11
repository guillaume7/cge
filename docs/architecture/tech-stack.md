# Tech Stack — Cognitive Graph Engine VP1 + VP2 + VP3 + VP4 + VP8

## Overview

The MVP stack prioritizes:

- local/offline operation
- simple binary distribution
- embedded storage
- strong shell composability
- low operational overhead

All key choices are documented in ADRs and are currently **Proposed** pending
user review.

## Chosen Stack

| Concern | Technology | Rationale | ADR |
|---|---|---|---|
| CLI language | Go 1.22+ | Simple binaries, strong stdlib, good CLI ergonomics | ADR-001 |
| CLI framework | Cobra | Proven command/subcommand model, help generation, good fit for `graph` UX | ADR-001 |
| Config / flags | Cobra + stdlib | Avoid extra config abstraction for the current product stage | ADR-001 |
| Graph database | Kuzu embedded DB | Native graph storage, local embedding, no service deployment | ADR-002 |
| Text relevance index | Bleve | Embedded full-text ranking, offline, easy Go integration | ADR-006 |
| Hygiene analysis | In-process Go analyzers over graph snapshots | Reuse current store and avoid adding services or a second data platform | ADR-007 |
| Stats computation | On-demand snapshot metrics in-process | Avoid a metrics backend and keep stats cheap, local, and explicit | ADR-008 |
| Workflow orchestration | Thin in-process Go workflow service | Compose existing graph primitives for delegated subtasks without a daemon or control plane | ADR-009 |
| Workflow metadata | Composable markdown/YAML/snippet assets + small shell hooks | Make graph-backed delegation natural without rigid repo takeover | ADR-010 |
| Benchmark reports | Local JSON reports + CLI-facing summary surface | Keep benchmark evidence reproducible and local while still machine-readable | ADR-011 |
| Experiment orchestration | Thin in-process Go lab orchestrator | Compose existing workflow and benchmark primitives into controlled experiment batches without a daemon or external platform | ADR-012 |
| Run ledger | Local filesystem JSON records under `.graph/lab/` | Keep run artifacts immutable, inspectable, and traceable without an external database | ADR-013 |
| Evaluation / scoring | Separated in-process evaluation step with blinding support | Enable condition-blind quality scoring and independent re-evaluation | ADR-014 |
| Report generation | In-process statistical aggregation to local JSON reports | Derive scientific-style reports from run and evaluation artifacts without external analytics tooling | ADR-013 |
| Context evaluation | In-process Go heuristic evaluator | Score candidate context for relevance, consistency, and usefulness locally without LLM calls | ADR-018 |
| Decision engine | In-process Go threshold-driven outcome selector | Select continue/minimal/abstain/backtrack/write outcomes from evaluator scores | ADR-019 |
| Attribution recording | Local filesystem JSON records under `.graph/attribution/` | Persist decision evidence for lab analysis and agent inspection | ADR-021 |
| Payload format | Versioned JSON | Native machine-readable protocol for stdin/stdout/files | ADR-005 |
| Testing | Go `testing` package | Keep setup simple and standard | ADR-001 |
| Build / dependency management | Go modules | Reproducible builds with `go.mod` and `go.sum` | ADR-001 |

## Versioning Guidance

- Pin direct dependencies in `go.mod`
- Commit `go.sum`
- Revisit dependency upgrades at epic boundaries

## Technology Notes

### Go

Why it fits:

- convenient for CLI development
- easy local binary distribution
- straightforward integration with Unix pipes
- good ecosystem maturity for embedded tools

Trade-off:

- local ML/embedding integration is weaker than Python/Rust ecosystems, so MVP
  retrieval should stay lean

### Cobra

Why it fits:

- fast path to a clean command tree
- standard flag parsing and help output
- easy to test command behavior

Trade-off:

- adds a framework layer for a small CLI, but the ergonomics justify it

### Kuzu

Why it fits:

- embedded graph database aligned with the product identity
- supports local graph queries without external infrastructure
- better fit than stretching a document store into a graph product

Trade-off:

- schema changes need deliberate migration design
- Go integration needs care around native bindings

### Bleve

Why it fits:

- embedded and offline
- practical for MVP text relevance
- complements Kuzu without replacing it

Trade-off:

- does not provide dense vector semantics by itself
- requires a second local persistence mechanism alongside the graph DB

### In-process hygiene analyzers

Why they fit:

- reuse the current Kuzu-backed graph snapshot instead of introducing another
  persistent subsystem
- keep duplicate/orphan/contradiction analysis local and explainable
- make suggest-first hygiene cheap to invoke from the CLI

Trade-off:

- heuristic analysis quality will depend on careful tuning
- larger graphs may require optimization work inside the process

### On-demand stats computation

Why it fits:

- VP2 requires snapshot health metrics, not trend infrastructure
- avoids shipping a metrics store, scheduler, or daemon
- aligns with explicit agent workflows like `graph stats` before retrieval-heavy
  work

Trade-off:

- expensive metrics must be computed efficiently enough for interactive CLI use
- no historical trends are available unless a later phase adds them

### Thin workflow orchestration

Why it fits:

- VP3 only needs the delegated-subtask golden path, not a full workflow platform
- reuses existing retrieval, stats, hygiene, write, and revision primitives
- keeps orchestration explicit and local

Trade-off:

- adds orchestration glue that must remain disciplined and narrow
- may feel redundant if the workflow envelopes are not materially better than ad
  hoc prompting

### Composable workflow metadata and small hooks

Why they fit:

- the current pain point is wiring prompts, skills, and instructions to use the graph
- snippets are easier to adopt than a rigid repo takeover
- small hooks can reduce friction without hiding the actual workflow commands

Trade-off:

- too many snippets or hooks can become ceremony
- local overrides need clear preservation rules during refresh

### Local benchmark reports

Why they fit:

- benchmark evidence must stay reproducible and inspectable inside the repo
- JSON reports are easy to diff, archive, and summarize from the CLI
- avoids external telemetry services for the first benchmark phase

Trade-off:

- output-quality measurement still needs careful rubric design
- benchmark harness realism depends on good scenario selection

### In-process experiment lab orchestrator

Why it fits:

- VP4 needs controlled multi-factor experiments, not a hosted platform
- composing existing workflow and benchmark primitives keeps the new layer thin
- batch orchestration, condition assignment, and randomization are lightweight
  logic that fit naturally in the CLI process

Trade-off:

- adds a new orchestration layer that must stay disciplined
- statistical reporting adds implementation complexity beyond simple aggregation

### Filesystem-based run ledger

Why it fits:

- run artifacts must be immutable, inspectable, and traceable without tooling
- JSON files under `.graph/lab/runs/` are easy to diff, archive, and audit
- avoids adding a database dependency for experiment metadata
- aligns with the repo-local artifact philosophy from VP1 through VP3

Trade-off:

- querying across many runs requires in-process scanning rather than indexed
  queries
- large experiment batches may be slower to aggregate than a database-backed
  alternative

### Separated evaluation with blinding support

Why it fits:

- VP4 principles explicitly require execution/judgment separation
- storing evaluation records outside run records enables re-scoring and blinding
- supports both automated rubrics and human judgment without changing the run
  path

Trade-off:

- two-phase workflow (run then evaluate) is more complex than inline scoring
- blinding requires careful artifact presentation to strip condition metadata

### In-process context evaluator (VP8)

Why it fits:

- VP8 requires evaluation before trust for every context retrieval path
- heuristic-based scoring (overlap, recency, coherence) is lightweight enough
  for in-process CLI use
- avoids LLM calls on the critical path, preserving local-first operation
- complements (does not replace) the existing retrieval ranking pipeline

Trade-off:

- heuristic scoring is weaker than semantic evaluation from an LLM
- calibrating scoring weights requires empirical evidence from lab experiments
- adds latency to every evaluated retrieval path

### In-process decision engine (VP8)

Why it fits:

- VP8 requires normalized outcomes (continue/minimal/abstain/backtrack/write)
  that go beyond the binary inject/suppress model from VP6
- threshold-driven outcome selection is cheap local logic
- composable with existing family-aware kickoff policies

Trade-off:

- threshold calibration is empirical; bad defaults can cause over-abstention
  or under-filtering
- five outcomes are more complex for consumers than two

### Local attribution records (VP8)

Why they fit:

- VP8 makes attribution load-bearing for lab analysis
- JSON records under `.graph/attribution/` follow the same local-filesystem
  pattern as run ledger and evaluation records
- keeps decision evidence inspectable and diffable

Trade-off:

- attribution volume grows with every evaluated retrieval; may need pruning
- adds a second write to every evaluator-loop pass (inline envelope + persisted
  record)

## Rejected Simpler/Heavier Options

### Python

- Better ML ecosystem
- Worse single-binary distribution story
- Rejected because the CLI-first local distribution experience matters more in
  MVP

### Rust

- Strong performance and binary story
- Higher implementation friction for fast MVP iteration
- Rejected because Go is the more pragmatic choice for this product stage

### Neo4j or remote graph service

- Mature ecosystem
- Violates local/offline simplicity
- Rejected because the MVP must remain embedded and repo-scoped

### Dense local embedding stack in MVP

- Stronger semantic retrieval potential
- Higher complexity in binaries, model packaging, and inference runtime
- Rejected for MVP in favor of local BM25/FTS plus graph-aware ranking

### Separate metrics backend or observability store in VP2
- Pros: could support trend analysis and larger-scale dashboards
- Cons: adds deployment, persistence, and synchronization complexity
- Rejected because: VP2 only requires snapshot stats and must remain a simple
  local CLI

### Background autonomous hygiene daemon in VP2
- Pros: could keep the graph continuously tidy
- Cons: violates the explicit/safe-by-default cleanup principle and adds runtime
  complexity
- Rejected because: VP2 should suggest first and apply only on explicit request

### Full workflow automation daemon in VP3
- Pros: could make graph usage automatic everywhere
- Cons: high hidden complexity, weak transparency, difficult to trust during dogfooding
- Rejected because: VP3 must first prove the delegated-subtask golden path with
  explicit, inspectable workflow steps

### Rigid repo-wide metadata takeover in VP3
- Pros: stronger standardization, fewer local choices
- Cons: too invasive for current dogfooding, harder to preserve repo conventions
- Rejected because: composable snippets are the lower-risk adoption path

### External observability platform for workflow benchmarking
- Pros: richer analytics and long-term dashboards
- Cons: hosted complexity, privacy concerns, operational overhead
- Rejected because: VP3 needs local evidence first, not a telemetry platform

### Hosted experiment platform or online dashboard for VP4
- Pros: collaborative experiment design, richer visualization, persistent history
- Cons: violates local-first principle, adds deployment and privacy overhead
- Rejected because: VP4 must stay local and inspectable; the experiment lab must
  work from repo-local artifacts with zero external dependencies

### SQLite or embedded database for the VP4 run ledger
- Pros: richer querying, better aggregation primitives, indexed access
- Cons: adds a dependency, complicates the single-binary story, overkill for
  expected experiment scale
- Rejected because: filesystem-based JSON records are sufficient for VP4 scope
  and can be reconsidered if experiment scale demands it

### Inline quality scoring during VP4 run execution
- Pros: simpler single-step workflow, immediate results
- Cons: prevents blinding, couples scoring to execution, cannot re-evaluate
- Rejected because: VP4 principles explicitly require execution/judgment
  separation to reduce confirmation bias

### LLM-based context evaluation on the critical path (VP8)
- Pros: richer semantic scoring, better relevance detection
- Cons: adds latency, cost, and external dependency; breaks local-first
  principle
- Rejected because: VP8 must stay local-first; heuristic scoring should be
  tried first. LLM-based scoring can be added in a future VP if heuristics
  prove insufficient

### Skip evaluation and use retrieval ranking as the only quality gate (VP8)
- Pros: no new component, minimal pipeline change
- Cons: ranking cannot detect task-level relevance drift or consistency problems
- Rejected because: VP8 identifies the missing evaluator loop as the core
  product failure to correct

### Build CGE as a standalone agent runtime instead of a Copilot CLI augmentation (VP8)
- Pros: full control over agent lifecycle
- Cons: duplicates Copilot CLI capabilities, adds platform complexity
- Rejected because: VP8 should prove value in the existing Copilot CLI workflow
  before building a separate runtime
