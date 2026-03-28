# Components — Cognitive Graph Engine VP1 + VP2 + VP3 + VP4 + VP5 + VP6 + VP7

## Component Map

### 1. CLI Surface

- **Responsibility**: Parse commands, flags, stdin/stdout modes, and exit codes.
- **Interface**: `graph init|write|query|context|explain|diff|stats|hygiene|workflow|lab`
- **Data ownership**: None
- **Dependencies**: Payload Validator, Graph Repository Manager, Retrieval
  Engine, Context Projector, Explain/Diff Service, Workflow Orchestration,
  Experiment Lab Orchestrator

### 2. Payload Validator

- **Responsibility**: Validate native JSON payloads and command inputs before
  execution.
- **Interface**: internal validation methods used by command handlers
- **Data ownership**: payload schema version definitions
- **Dependencies**: CLI Surface

### 3. Graph Repository Manager

- **Responsibility**: Discover repo root, manage on-disk graph workspace, and
  initialize local storage.
- **Interface**: init/open repository graph, read/write repository metadata
- **Data ownership**: graph workspace layout, schema version metadata
- **Dependencies**: Kuzu Store, Text Index

### 4. Kuzu Store

- **Responsibility**: Persist graph entities, relationships, provenance, and
  graph state revisions.
- **Interface**: upsert/load/query/diff primitives for graph data
- **Data ownership**: graph system of record
- **Dependencies**: Kuzu

### 5. Text Index

- **Responsibility**: Maintain searchable text projections for retrieval.
- **Interface**: index entity text, search task text, return ranked candidates
- **Data ownership**: local search index
- **Dependencies**: Bleve

### 6. Retrieval Engine

- **Responsibility**: Combine structural graph retrieval and text-relevance
  retrieval into ranked task candidates under a task-family retrieval policy.
- **Interface**: query(task, policy), context(task, tokenBudget, policy), explain(task)
- **Data ownership**: ranking logic, retrieval traces, and policy-filtered
  candidate sets, including verification-profile-aware candidate suppression
- **Dependencies**: Kuzu Store, Text Index

### 7. Context Projector

- **Responsibility**: Compress ranked results into a prompt-ready context
  envelope that respects token budgets and can explain why each entity survived
  projection.
- **Interface**: project(results, tokenBudget, policyDecision)
- **Data ownership**: projection rules, truncation policy, and inclusion-reason formatting
- **Dependencies**: Retrieval Engine

### 8. Explain / Diff Service

- **Responsibility**: Produce explainable retrieval output and graph change
  reports.
- **Interface**: explain(queryRun), diff(stateA, stateB)
- **Data ownership**: explanation traces and diff formatting logic
- **Dependencies**: Kuzu Store, Retrieval Engine

### 9. Stats Service

- **Responsibility**: Compute snapshot graph metrics and cognitive health
  indicators on demand.
- **Interface**: `stats()`
- **Data ownership**: metric definitions and computation rules
- **Dependencies**: Kuzu Store

### 10. Hygiene Service

- **Responsibility**: Detect duplicate-near-identical nodes, orphan nodes, and
  contradictory facts, then generate or apply cleanup plans.
- **Interface**: `suggestHygiene()`, `applyHygiene(plan)`
- **Data ownership**: hygiene suggestion logic, action plan schema, and cleanup
  orchestration rules
- **Dependencies**: Kuzu Store, Explain / Diff Service

### 11. Workflow Asset Manager

- **Responsibility**: Install, refresh, and inspect the composable prompts,
  skills, instruction snippets, and wrapper/hook assets that enable graph-backed
  delegated workflow.
- **Interface**: `installAssets()`, `refreshAssets()`, `loadManifest()`
- **Data ownership**: local workflow asset manifest, asset installation metadata,
  and preserved-override records
- **Dependencies**: Graph Repository Manager, local filesystem

### 12. Delegation Workflow Service

- **Responsibility**: Orchestrate the delegated-subtask golden path through
  `workflow init`, `workflow start`, and `workflow finish` without replacing the
  existing graph primitives.
- **Interface**: `workflowInit()`, `workflowStart(task, kickoffMode)`, `workflowFinish(outcome)`
- **Data ownership**: kickoff family classification, policy selection,
  verification sub-profile routing, confidence/abstention rules,
  verification-specific downgrade and token-budget decisions, kickoff and
  handoff envelope assembly rules, workflow recommendations, and delegated-task
  orchestration logic
- **Dependencies**: Workflow Asset Manager, Graph Repository Manager, Retrieval
  Engine, Context Projector, Stats Service, Hygiene Service, Explain / Diff
  Service, Kuzu Store

### 13. Benchmark Evaluation Service

- **Responsibility**: Run and summarize delegated-subtask benchmark scenarios in
  with-graph and without-graph modes.
- **Interface**: `runBenchmark()`, `loadBenchmarkReports()`, `summarizeBenchmark()`
- **Data ownership**: benchmark scenario definitions, benchmark run reports, and
  comparison summaries
- **Dependencies**: Delegation Workflow Service, Graph Repository Manager, local
  filesystem

### 14. Experiment Lab Orchestrator

- **Responsibility**: Manage the experiment lifecycle: initialize lab assets,
  execute controlled benchmark batches with condition assignment and
  randomization, and delegate to the report generator for aggregate analysis.
- **Interface**: `labInit()`, `labRun(runRequest)`, `labReport(reportRequest)`
- **Data ownership**: suite manifest, condition definitions, batch lifecycle
  state, randomization/counterbalancing logic, and verification-rerun
  attribution planning
- **Dependencies**: Delegation Workflow Service, Benchmark Evaluation Service,
  Run Ledger, Evaluation Service, Report Generator, Graph Repository Manager

### 15. Run Ledger

- **Responsibility**: Persist immutable per-run records and outcome artifacts
  for every controlled experiment run.
- **Interface**: `writeRun(record)`, `loadRun(runId)`, `listRuns(filter)`
- **Data ownership**: run records under `.graph/lab/runs/`, run record schema
  version, preserved raw kickoff responses, and baseline prompt-surface
  metadata
- **Dependencies**: local filesystem

### 16. Evaluation Service

- **Responsibility**: Score run outcomes for quality, success, and resumability
  independently from run execution. Support blinded presentation of run
  artifacts for condition-blind evaluation.
- **Interface**: `evaluate(runId, rubric)`, `loadEvaluation(runId)`,
  `presentBlinded(runId)`
- **Data ownership**: evaluation records under `.graph/lab/evaluations/`,
  evaluation rubric definitions
- **Dependencies**: Run Ledger, local filesystem

### 17. Report Generator

- **Responsibility**: Aggregate run ledger entries and evaluation records into
  scientific-style reports with paired comparisons, grouped analysis, effect
  sizes, and uncertainty intervals.
- **Interface**: `generateReport(reportRequest)`, `loadReport(reportId)`
- **Data ownership**: aggregate report artifacts under `.graph/lab/reports/`,
  statistical computation logic
- **Dependencies**: Run Ledger, Evaluation Service

## Boundary Rules

- The CLI Surface never talks directly to Kuzu or Bleve internals.
- Kuzu remains the system of record for graph knowledge.
- The Text Index is derived data and can be rebuilt.
- Retrieval logic is centralized in the Retrieval Engine.
- Context shaping is separated from retrieval so token policies remain explicit.
- Workflow kickoff policy is separated from raw retrieval ranking so the system
  can abstain when precision is low or the task family is known to regress.
- Verification and audit tasks may be split into narrower sub-profiles with
  stricter token budgets and downgrade rules than other families.
- Stats are derived on demand from the graph system of record.
- Hygiene suggestions are advisory until an explicit apply workflow is invoked.
- The Delegation Workflow Service composes existing primitives; it does not
  replace or bypass their ownership boundaries.
- Wrapper/hook automation must remain thin and transparent around explicit
  workflow commands.
- Reporting and synthesis tasks may intentionally receive no kickoff context by
  default when the selected policy says abstention is safer than injection.
- Verification-focused experiment runs must preserve enough kickoff and baseline
  prompt metadata to diagnose why a pair injected, minimized, abstained, won, or
  regressed.
- Benchmark reports are evaluation artifacts, not the graph system of record.
- Experiment run records are immutable once written; evaluation is stored
  separately to preserve the execution/judgment boundary.
- The Experiment Lab Orchestrator composes the Delegation Workflow Service and
  Benchmark Evaluation Service; it does not bypass their ownership boundaries.
- Evaluation scoring is separated from run execution so scoring can be blinded
  and re-run independently.
- Report generation reads the run ledger and evaluation records; reports are
  derived artifacts, not the system of record.

## Dependency Diagram

```text
CLI Surface
  ├── Payload Validator
  ├── Graph Repository Manager
  │     ├── Kuzu Store
  │     └── Text Index
  ├── Retrieval Engine
  │     ├── Kuzu Store
  │     └── Text Index
  ├── Context Projector
  ├── Explain / Diff Service
  ├── Stats Service
  │     └── Kuzu Store
  └── Hygiene Service
        ├── Kuzu Store
        └── Explain / Diff Service
  └── Workflow Asset Manager
        └── Graph Repository Manager
  └── Delegation Workflow Service
        ├── Workflow Asset Manager
        ├── Retrieval Engine
        ├── Context Projector
        ├── Stats Service
        ├── Hygiene Service
        ├── Explain / Diff Service
        └── Kuzu Store
  └── Benchmark Evaluation Service
        ├── Delegation Workflow Service
        └── Graph Repository Manager
  └── Experiment Lab Orchestrator
        ├── Delegation Workflow Service
        ├── Benchmark Evaluation Service
        ├── Run Ledger
        ├── Evaluation Service
        │     └── Run Ledger
        └── Report Generator
              ├── Run Ledger
              └── Evaluation Service
```

## Why These Boundaries

These boundaries keep the MVP simple while making three concerns explicit:

- storage
- retrieval
- projection/explanation
- graph-health analysis and cleanup
- delegated-subtask workflow orchestration
- benchmark evidence collection
- controlled experiment orchestration with separated evaluation

That separation is enough to keep implementation clean without inventing a
premature service architecture.
