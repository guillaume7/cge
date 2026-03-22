# Components — Cognitive Graph Engine VP1 + VP2

## Component Map

### 1. CLI Surface

- **Responsibility**: Parse commands, flags, stdin/stdout modes, and exit codes.
- **Interface**: `graph init|write|query|context|explain|diff|stats|hygiene`
- **Data ownership**: None
- **Dependencies**: Payload Validator, Graph Repository Manager, Retrieval
  Engine, Context Projector, Explain/Diff Service

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
  retrieval into ranked task candidates.
- **Interface**: query(task), context(task, tokenBudget), explain(task)
- **Data ownership**: ranking logic and retrieval traces
- **Dependencies**: Kuzu Store, Text Index

### 7. Context Projector

- **Responsibility**: Compress ranked results into a prompt-ready context
  envelope that respects token budgets.
- **Interface**: project(results, tokenBudget)
- **Data ownership**: projection rules, truncation policy
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

## Boundary Rules

- The CLI Surface never talks directly to Kuzu or Bleve internals.
- Kuzu remains the system of record for graph knowledge.
- The Text Index is derived data and can be rebuilt.
- Retrieval logic is centralized in the Retrieval Engine.
- Context shaping is separated from retrieval so token policies remain explicit.
- Stats are derived on demand from the graph system of record.
- Hygiene suggestions are advisory until an explicit apply workflow is invoked.

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
```

## Why These Boundaries

These boundaries keep the MVP simple while making three concerns explicit:

- storage
- retrieval
- projection/explanation
- graph-health analysis and cleanup

That separation is enough to keep implementation clean without inventing a
premature service architecture.
