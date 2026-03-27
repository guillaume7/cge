# Architecture Overview — Cognitive Graph Engine VP1 + VP2 + VP3 + VP4 + VP5 + VP6

## System Context

The product is a local, repo-scoped CLI named `graph` that provides a shared
graph memory for AI agents and sub-agents working inside the same repository.

Its core purpose is to improve:

- task continuity across sessions and agent handoffs
- token efficiency by returning compact context instead of large prompt reloads
- trust in retrieved context through provenance and explanation
- long-term graph maintainability through hygiene and graph-health visibility
- delegated-subtask kickoff and handoff quality through graph-backed workflow
- measurable reduction in recovery-token cost for non-trivial delegated work
- credible, repeatable experimental evidence about when graph-backed workflow
  helps and when it does not

The product remains local/offline, chainable in shell pipelines, and optimized
for machine use rather than human graph exploration.

## Key Requirements

### Functional

- initialize a repository-scoped graph store
- accept explicit structured writes from agents
- store reasoning, planning, project, and codebase entities
- support shell-native chaining through stdin/stdout
- retrieve relevant graph data for a task
- project a compact context slice within a token budget
- explain why context or query results were returned
- diff graph states over time
- allow graph cleanup and rewrite workflows
- compute graph-health snapshots
- suggest graph hygiene actions for duplicates, orphans, and contradictions
- apply explicit hygiene plans and record them as graph revisions
- install or refresh repo-local workflow assets that make graph-backed delegation
  the normal path
- seed enough baseline graph knowledge for delegated subtasks to inherit useful
  context
- produce graph-backed kickoff envelopes for non-trivial delegated subtasks
- persist graph-backed delegated-task handoff/writeback envelopes
- benchmark delegated subtasks with and without graph-backed workflow support
- expose benchmark summaries through a local evaluation harness and a CLI-facing
  report surface
- define controlled experiment suites with explicit conditions, blocking
  factors, and reproducible seed tracking
- execute repeated controlled benchmark runs across models, topologies, and
  graph conditions
- capture immutable per-run artifacts including telemetry, session structure,
  and outcomes
- evaluate run outcomes for quality, success, and resumability separately from
  run execution
- generate scientific-style reports with paired comparisons, effect sizes, and
  uncertainty intervals
- classify delegated workflow-start tasks into kickoff families
- apply family-specific entity-type allowlists and suppressions before context
  injection
- emit advisory kickoff confidence and abstention recommendations
- explain why each kickoff entity was included in the delegation brief
- allow explicit no-kickoff and minimal-kickoff operation without breaking the
  existing workflow contract

### Non-Functional

- offline/local operation
- simple single-machine setup
- stable machine-readable input/output contract
- trustworthy provenance on writes and retrieval
- proportional MVP complexity
- safe-by-default graph mutation for hygiene workflows
- low-ceremony workflow embedding that saves more tokens than it costs
- transparent automation that remains inspectable and easy to opt out of
- precision-biased kickoff behavior where false positives are treated as more
  harmful than missing context
- reproducible benchmark runs with comparable task quality checks
- artifact-first experiment evidence traceable to concrete run records
- separated evaluation that supports blinding and re-scoring
- proportional experiment tooling with no hosted telemetry or external services

## Architectural Style

The product uses a **single-process CLI with embedded storage and a small
internal service layer**.

This is intentionally simple:

- one local binary
- one graph database as the system of record
- one local text-relevance index for retrieval ranking
- one in-process graph analysis layer for stats and hygiene
- one thin workflow orchestration layer for delegated-subtask kickoff/handoff
- one local benchmark harness/report path for evaluating workflow usefulness
- one local experiment lab for controlled multi-factor benchmarking
- one task-family policy layer that can abstain from kickoff when precision is
  low or the task family is known to be contamination-prone
- no remote services
- no daemon requirement

## High-Level Design

```text
stdin / args / files
        │
        v
  +-------------+
  | graph CLI   |
  | commands    |
  +------+------+ 
         |
         v
  +-------------+      +------------------+
  | payload &   |----->| graph repository |
  | command     |      | manager          |
  | validation  |      +--------+---------+
  +------+------+               |
         |                      v
         |              +---------------+
         |              | Kuzu graph DB |
         |              +---------------+
         |
         v
  +-------------+      +------------------+
  | retrieval   |<---->| local text index |
  | engine      |      | (BM25/FTS)       |
  +------+------+      +------------------+
          |
          +-------------------+--------------------+----------------------+
          |                   |                    |                      |
          v                   v                    v                      v
   +-------------+      +-------------+     +--------------+     +---------------+
   | context     |      | explain/diff|     | stats &      |     | workflow      |
   | projector   |      | services    |     | hygiene      |     | orchestration |
   +------+------+      +-------------+     | analyzers    |     +-------+-------+
          |                                 +------+-------+             |
          |                                        |                     v
          +----------------------------------------+             +---------------+
          |                                                      | benchmark     |
          |                                                      | harness       |
          |                                                      +-------+-------+
          |                                                              |
          |                                                              v
          |                                                      +---------------+
          |                                                      | experiment    |
          |                                                      | lab           |
          |                                                      | orchestrator  |
          |                                                      +--+--------+---+
          |                                                         |        |
          |                                                    +----+---+ +--+--------+
          |                                                    |run     | |evaluation |
          |                                                    |ledger  | |& report   |
          |                                                    +----+---+ +--+--------+
          |                                                         |        |
          v                                                         v        v
  stdout / files                                              stdout / files
```

## Request Flows

### `graph init`

1. Detect repo root.
2. Create graph workspace under the repo.
3. Initialize Kuzu storage, metadata, and text index.
4. Record schema version and repository identity.

### `graph write`

1. Read native graph payload from file or stdin.
2. Validate required metadata and payload schema.
3. Upsert entities and relationships into Kuzu.
4. Refresh text index entries for relevant nodes.
5. Emit machine-readable write summary to stdout.

### `graph query`

1. Parse task text from flags or stdin.
2. Retrieve graph candidates through structural traversal.
3. Retrieve text-relevance candidates from the local index.
4. Rank and merge candidates.
5. Return a structured result set with provenance.

### `graph context`

1. Run the same hybrid retrieval pipeline as `graph query`.
2. Expand only the neighborhood needed for continuity.
3. Project results into a token-budgeted context envelope.
4. Return compact machine-readable context for downstream agents.

### `graph explain`

1. Reconstruct retrieval decisions.
2. Show matched terms, graph paths, ranking reasons, and provenance.
3. Return a structured explanation payload.

### `graph diff`

1. Compare two graph states or revisions.
2. Report added, updated, removed, and retagged entities/relations.
3. Return machine-readable change sets.

### `graph stats`

1. Open the current repo-local graph snapshot.
2. Compute structural counts and health indicators on demand.
3. Return machine-readable snapshot metrics for agent decisions.

### `graph hygiene`

1. Analyze the current graph for duplicate-near-identical nodes, orphan nodes,
   and contradictory facts.
2. In suggest mode, return structured candidate actions and explanations.
3. In apply mode, execute only explicit requested actions.
4. Persist the resulting graph changes through the normal revision flow.

### `graph workflow init`

1. Detect repo root and current graph/workflow state.
2. Install or refresh the minimum workflow assets needed for delegated-subtask
   kickoff and handoff.
3. Initialize the repo-local graph if missing.
4. Seed baseline graph knowledge from standard repo artifacts when needed.
5. Emit a machine-readable summary of assets installed, preserved, skipped, and
   seeded.

### `graph workflow start`

1. Accept a delegated-subtask request.
2. Inspect graph availability, current revision state, and graph health.
3. Retrieve compact task-relevant context from the graph.
4. Produce a machine-readable kickoff envelope suitable for a sub-agent prompt.
5. Recommend whether to proceed, bootstrap, inspect hygiene, or gather more
   context.

### `graph workflow finish`

1. Accept a structured delegated-task outcome payload.
2. Validate the payload and write durable graph memory through the existing
   revision-aware write path.
3. Return revision anchors, write summaries, and a machine-readable handoff
   envelope for the next agent.

### `graph workflow benchmark`

1. Run or summarize delegated-subtask benchmark scenarios in both with-graph and
   without-graph modes.
2. Capture token/prompt usage, orientation effort, output-quality proxies, and
   handoff quality metrics.
3. Emit machine-readable benchmark reports for both local evaluation harness use
   and CLI consumption.

### `graph lab init`

1. Detect repo root and current graph/lab state.
2. Create or refresh the local experiment workspace under `.graph/lab/`.
3. Install or update the benchmark suite manifest, condition definitions,
   artifact directories, and evaluation scaffolding.
4. Emit a machine-readable summary of lab assets installed, refreshed, or
   preserved.

### `graph lab run`

1. Accept a run request declaring task ID, experimental condition, model,
   session topology, prompt variant, and seed.
2. Validate the declared condition against the suite manifest.
3. Assign or verify randomized/counterbalanced condition ordering when running
   a batch.
4. Execute the task through existing workflow primitives (`workflow start`,
   `workflow finish`) for graph-backed conditions or through a baseline path
   for no-graph conditions.
5. Capture truthful run telemetry: kickoff inputs, delegated session structure,
   writeback outputs, measured token/usage data when available, explicit
   completeness state when not, plus timing and retry information.
6. Write an immutable run record and outcome artifacts to the run ledger under
   `.graph/lab/runs/<run-id>/`.
7. Emit a machine-readable run summary to stdout.

### `graph lab report`

1. Read the run ledger and evaluation records from `.graph/lab/`.
2. Aggregate completed runs into a scientific-style report.
3. Support paired within-task comparisons, grouped comparisons by model or
   topology, success/failure rates, token and step distributions,
   resumability and handoff quality comparisons, effect-size summaries, and
   uncertainty intervals.
4. Write the report to `.graph/lab/reports/` and emit a machine-readable
   summary to stdout.

## Data Ownership

### Graph Repository Manager

Owns:

- repository graph location
- schema version metadata
- initialization state

### Graph Store

Owns:

- entities
- relationships
- provenance metadata
- state revision markers

### Text Index

Owns:

- indexed text projections for retrieval
- searchable aliases, titles, summaries, and content excerpts

### Context Projector

Owns:

- token-budgeted output shaping rules
- result compaction policies

### Stats and Hygiene Analyzers

Own:

- graph-health metrics
- duplicate/orphan/contradiction detection logic
- suggested hygiene action plans

### Workflow Orchestration Layer

Owns:

- delegated-subtask kickoff envelope assembly
- delegated-task finish/handoff envelope assembly
- workflow init/install decisions over local assets

### Benchmark Harness / Report Layer

Owns:

- benchmark scenario definitions for delegated subtasks
- benchmark report generation
- comparison summaries for with-graph versus without-graph runs

### Experiment Lab Orchestrator

Owns:

- experiment lifecycle management (init, run, report)
- suite manifest and condition definition management
- condition assignment, randomization, and blocking-factor awareness
- batch execution orchestration over workflow primitives

### Run Ledger

Owns:

- immutable per-run records and outcome artifacts
- run ledger directory layout under `.graph/lab/runs/`
- run record schema versioning

### Evaluation Service

Owns:

- quality, success, and resumability scoring logic
- evaluation records stored under `.graph/lab/evaluations/`
- blinded artifact presentation for condition-blind evaluation
- automated rubric execution and human scoring scaffolding

### Report Generator

Owns:

- aggregate report generation from run ledger and evaluation records
- paired and grouped comparison logic
- effect-size computation and uncertainty interval estimation
- report artifacts stored under `.graph/lab/reports/`

## Key Architectural Decisions

- **ADR-001** — Use Go and Cobra for the CLI implementation
- **ADR-002** — Use Kuzu as the embedded graph system of record
- **ADR-003** — Use repo-local graph storage with a deterministic on-disk layout
- **ADR-004** — Use a flexible entity-centric graph schema with provenance-first
  metadata
- **ADR-005** — Define a versioned native JSON payload contract for stdin,
  stdout, and files
- **ADR-006** — Use a hybrid retrieval pipeline with graph traversal, local text
  relevance ranking, context projection, and explanation
- **ADR-007** — Use a suggest-first graph hygiene workflow with explicit apply
- **ADR-008** — Compute graph stats and health indicators on demand from the
  current graph snapshot
- **ADR-009** — Add a thin delegated-workflow orchestration layer over existing
  graph primitives
- **ADR-010** — Use composable workflow snippets with transparent wrapper/hooks
- **ADR-011** — Benchmark delegated workflow through a local harness and CLI
  report surface
- **ADR-012** — Local experiment-lab orchestration and `graph lab` command
  surface
- **ADR-013** — Artifact-first run ledger and scientific reporting model
- **ADR-014** — Separated evaluation protocol for quality scoring

## Why This Architecture Is Proportional

This architecture is deliberately minimal for MVP:

- no networked services
- no distributed components
- no background process
- no separate API layer
- no human visualization subsystem
- no dedicated metrics backend

The only complexity retained is complexity that directly serves the product
value:

- graph persistence
- retrieval quality
- provenance
- chainability
- graph maintainability
- delegated-subtask briefing and handoff quality
- evidence that the workflow actually saves recovery cost
- controlled experiment evidence with scientific-style reporting

## Open Review Focus

Before planning themes and stories, the most important review points are:

1. the boundary between thin workflow orchestration and the existing graph
   primitives it composes
2. how transparent wrappers/hooks should remain while still making graph-backed
   delegation feel natural
3. whether benchmark evidence is strong enough to justify later workflow
   expansion beyond the delegated-subtask golden path
4. the boundary between the experiment lab orchestrator and the existing
   benchmark harness — where does controlled-experiment logic live vs
   single-scenario benchmarking
5. whether the run ledger schema is lean enough for local filesystem storage
   while rich enough for meaningful statistical analysis
6. how blinded evaluation should work in practice — which metrics can be
   blinded and which inherently reveal condition
