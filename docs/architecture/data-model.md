# Data Model — Cognitive Graph Engine VP1 + VP2 + VP3 + VP4

## Modeling Strategy

The MVP uses a **flexible entity-centric property graph model**.

Instead of creating a dedicated table for every domain concept, the graph keeps
a small stable core schema and represents domain diversity through entity kinds,
relation kinds, and structured properties.

This is the simplest way to satisfy the vision's wide entity surface while
remaining compatible with Kuzu's typed storage.

VP2 keeps this same model and adds graph-health analysis on top of it rather
than introducing a separate hygiene schema.

VP3 keeps the same graph system of record and adds delegated-workflow contracts
plus local benchmark report artifacts around it rather than inventing a second
memory system.

VP4 keeps the same graph system of record and adds a local experiment lab
alongside it. Experiment runs, evaluation records, and reports are stored as
local filesystem artifacts under `.graph/lab/`, not as graph entities.

## Core Node Shapes

### `Entity`

Represents most first-class graph objects.

Suggested fields:

- `id` — stable unique identifier
- `kind` — e.g. `Skill`, `Prompt`, `Function`, `ADR`, `UserStory`
- `title` — human/machine-readable display label
- `summary` — short compact description
- `content` — optional fuller body or excerpt
- `repo_path` — optional file path when applicable
- `language` — optional source language
- `tags` — normalized tags
- `created_at`
- `updated_at`
- `props_json` — extensible structured properties

### `ReasoningUnit`

Represents an atomic persisted reasoning boundary.

Suggested fields:

- `id`
- `task`
- `summary`
- `agent_id`
- `session_id`
- `timestamp`
- `props_json`

### `AgentSession`

Represents a session-level summary and provenance anchor.

Suggested fields:

- `id`
- `agent_id`
- `started_at`
- `ended_at`
- `summary`
- `repo_root`
- `props_json`

### `GraphRevision`

Represents comparable graph states for diffing.

Suggested fields:

- `id`
- `created_at`
- `created_by`
- `reason`
- `props_json`

## Core Relationship Shapes

Suggested fixed relation kinds:

- `RELATES_TO`
- `PART_OF`
- `DEPENDS_ON`
- `DERIVED_FROM`
- `GENERATED_IN`
- `CITES`
- `ABOUT`
- `SUPERSEDES`
- `CONTRADICTS`
- `CONSOLIDATED_INTO`

Each relationship should support:

- `kind`
- `created_at`
- `created_by`
- `props_json`

## VP2 Hygiene Model

VP2 should not create a second persistence model for graph cleanliness. Instead,
it should derive hygiene suggestions from the existing entity/relationship
snapshot and apply approved changes back into the same graph.

### Duplicate-near-identical analysis

Suggested comparison inputs:

- normalized `kind`
- normalized `title`
- aliases or equivalent keys in `props_json`
- `summary`
- relationship neighborhood overlap where useful

Illustrative suggestion shape:

```json
{
  "action": "consolidate_duplicate_nodes",
  "canonical_id": "adr:ADR-002",
  "duplicates": ["adr:ADR-002-copy"],
  "reasons": ["title_similarity", "matching_alias"],
  "confidence": 0.93
}
```

### Orphan-node analysis

An orphan candidate is a node with insufficient meaningful connectivity for its
kind and role in the graph.

The model should allow:

- structural orphan detection
- exclusions for intentionally isolated nodes if needed
- explicit pruning plans rather than immediate deletion

### Contradiction model

Contradictions are not a separate store. They are detected by comparing graph
facts that appear mutually incompatible.

A contradiction suggestion should capture:

- the conflicting entities or relationships
- the reason they are considered in tension
- the proposed resolution path

Illustrative suggestion shape:

```json
{
  "action": "resolve_contradiction",
  "facts": ["node:fact-a", "node:fact-b"],
  "reason": "same subject with incompatible property values",
  "proposed_resolution": "supersede fact-a with fact-b"
}
```

Applied resolutions may be represented through:

- property updates
- node/edge removal where safe
- explicit `SUPERSEDES`, `CONTRADICTS`, or `CONSOLIDATED_INTO` relationships
- revision metadata that explains the hygiene action

## Provenance Requirements

Every persisted write should be traceable through:

- `agent_id`
- `session_id`
- `timestamp`
- source entity or write reason when available

This provenance is necessary for:

- trusted context retrieval
- `graph explain`
- graph cleanup decisions
- handoff debugging

## Native Payload Contract

The canonical write envelope should be versioned JSON, shared across file I/O
and stdin/stdout.

Illustrative shape:

```json
{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-42",
    "timestamp": "2026-03-21T14:00:00Z"
  },
  "nodes": [],
  "edges": []
}
```

The exact field names can be refined during implementation, but the contract
must remain:

- structured
- stable
- machine-oriented
- versioned
- consistent across stdin, stdout, and files

## Retrieval Model

### Structural Retrieval

Uses graph relations, kinds, tags, and repo-local metadata to identify relevant
starting nodes and neighborhoods.

### Text-Relevance Retrieval

Uses a local text index over:

- titles
- summaries
- content excerpts
- tags
- aliases

### Hybrid Ranking

The retrieval engine should merge:

- structural matches
- text-relevance matches
- provenance-aware confidence signals

The ranked set is then passed into context projection.

## Graph Stats Model

VP2 stats are snapshot-oriented and computed on demand from the current graph.

### Required snapshot counts

- total node count
- total relationship count

### Required health indicators

- duplication rate
- orphan rate
- contradictory fact count
- density / clustering indicators

These indicators are derived values, not durable system-of-record entities.

Illustrative result shape:

```json
{
  "snapshot": {
    "nodes": 540,
    "relationships": 1320
  },
  "indicators": {
    "duplication_rate": 0.08,
    "orphan_rate": 0.03,
    "contradictory_facts": 4,
    "density_score": 0.71,
    "clustering_score": 0.64
  }
}
```

## Context Projection Model

The context envelope should aim to preserve:

- the minimum useful nodes
- critical relationships
- provenance
- a compact summary field for each result

It should avoid:

- dumping raw graph state
- including unrelated neighborhoods
- exhausting the token budget on low-value detail

## State Diff Model

`graph diff` should compare revision anchors and report:

- entities added
- entities updated
- entities removed
- relations added/removed/updated
- kind or tag changes

Because MVP allows rewrite history, revisions are comparison anchors rather than
immutable audit events.

VP2 should continue using revisions for hygiene apply operations so cleanup
remains inspectable through `graph diff`.

## VP3 Delegated-Workflow Model

VP3 should not create a separate graph persistence schema for workflow memory.
Instead, it should:

- keep durable knowledge inside the existing entity-centric graph model
- add structured kickoff/handoff contracts as machine-readable workflow envelopes
- keep benchmark runs as local evaluation artifacts

### Workflow Asset Manifest

Workflow installation state should live in a local manifest under the repo graph
workspace, for example:

```json
{
  "schema_version": "v1",
  "installed_at": "2026-03-22T17:00:00Z",
  "assets": [
    {
      "path": ".github/prompts/...",
      "kind": "prompt-snippet",
      "status": "installed"
    }
  ],
  "preserved_overrides": []
}
```

This manifest is workflow metadata, not the graph system of record.

### Delegated Subtask Kickoff Envelope

Illustrative shape:

```json
{
  "schema_version": "v1",
  "command": "workflow.start",
  "status": "ok",
  "result": {
    "task": "implement retrieval ranking adjustment",
    "graph_state": {
      "workspace_present": true,
      "current_revision": "rev:123",
      "health_summary": {
        "duplication_rate": 0.02,
        "orphan_rate": 0.00
      }
    },
    "recommended_action": "proceed",
    "context": {},
    "delegation_brief": {}
  }
}
```

Required contents:

- graph readiness
- current revision anchor when available
- health summary sufficient for go/no-go workflow advice
- task-relevant compact context
- a prompt-ready delegation brief

### Delegated Task Outcome Payload

Illustrative finish input:

```json
{
  "schema_version": "v1",
  "task": "implement retrieval ranking adjustment",
  "summary": "Adjusted ranking weights for prompt relevancy",
  "decisions": [],
  "changed_artifacts": [],
  "follow_up": []
}
```

This payload should be transformed into normal graph writes plus revision-aware
write summaries.

### Delegated Task Handoff Envelope

Illustrative finish output:

```json
{
  "schema_version": "v1",
  "command": "workflow.finish",
  "status": "ok",
  "result": {
    "before_revision": "rev:123",
    "after_revision": "rev:124",
    "write_summary": {},
    "handoff_brief": {}
  }
}
```

Required contents:

- before/after revision anchors
- structured write summary
- next-agent handoff brief
- explicit no-op signaling when no graph changes occurred

### Benchmark Scenario Model

Benchmark scenarios are evaluation artifacts, not graph entities.

Illustrative scenario shape:

```json
{
  "scenario_id": "delegated-subtask-001",
  "task_family": "delegated-non-trivial-subtask",
  "mode": "with_graph",
  "task_prompt": "implement retrieval ranking adjustment",
  "acceptance_criteria_ref": "scenario-local"
}
```

### Benchmark Report Model

Illustrative report shape:

```json
{
  "scenario_id": "delegated-subtask-001",
  "mode": "with_graph",
  "metrics": {
    "token_volume": 1200,
    "orientation_steps": 3,
    "quality_score": "pass",
    "handoff_quality": "strong"
  }
}
```

Benchmark reports should be stored locally, for example under
`.graph/benchmarks/`, and surfaced through a CLI-facing report flow.

## VP3 Modeling Rule

VP3 should prefer:

- graph persistence for durable shared knowledge
- workflow envelopes for orchestration contracts
- local report artifacts for benchmark evidence

It should avoid inventing a second long-lived memory store just for workflow
machinery.

## VP4 Experiment Lab Model

VP4 should not store experiment data in the graph database. Experiment runs,
evaluations, and reports are transient evidence artifacts, not durable domain
knowledge. They live in the local filesystem under `.graph/lab/`.

### Benchmark Suite Manifest

Defines the corpus of tasks available for controlled experiments.

Illustrative shape:

```json
{
  "schema_version": "v1",
  "suite_id": "delegated-workflow-evidence-v1",
  "created_at": "2026-04-01T10:00:00Z",
  "tasks": [
    {
      "task_id": "task-001",
      "family": "delegated-non-trivial-subtask",
      "description": "implement retrieval ranking adjustment",
      "acceptance_criteria_ref": "tasks/task-001/criteria.md"
    }
  ]
}
```

### Condition Manifest

Defines the experimental conditions that can be assigned to runs.

Illustrative shape:

```json
{
  "schema_version": "v1",
  "conditions": [
    {
      "condition_id": "with-graph",
      "workflow_mode": "graph_backed",
      "description": "full graph-backed kickoff and handoff"
    },
    {
      "condition_id": "without-graph",
      "workflow_mode": "baseline",
      "description": "no graph context; standard delegation only"
    }
  ],
  "blocking_factors": ["task_family", "model", "session_topology"]
}
```

### Run Record

Each controlled run produces a self-contained, immutable record.

Illustrative shape:

```json
{
  "schema_version": "v1",
  "run_id": "run-20260401-001",
  "task_id": "task-001",
  "condition_id": "with-graph",
  "model": "claude-sonnet-4-20250514",
  "session_topology": "delegated-parallel",
  "seed": 42,
  "prompt_variant": "default",
  "started_at": "2026-04-01T10:30:00Z",
  "finished_at": "2026-04-01T10:45:00Z",
  "telemetry": {
    "measurement_status": "complete",
    "source": "workflow_finish_payload",
    "provider": "copilot-cli",
    "total_tokens": 14200,
    "input_tokens": 8100,
    "output_tokens": 6100,
    "wall_clock_seconds": 900,
    "retry_count": 1,
    "delegated_sessions": 2
  },
  "kickoff_inputs_ref": "artifacts/kickoff.json",
  "session_structure_ref": "artifacts/sessions/",
  "writeback_outputs_ref": "artifacts/writeback.json",
  "outcome_artifacts_ref": "artifacts/output/"
}
```

Required contents:

- task and condition identity
- model and topology declarations
- seed for reproducibility
- measured token/usage telemetry or an explicit partial/unavailable state with reasons
- timing and retry information
- references to preserved outcome artifacts

Run records are written once and never modified after completion.

### Evaluation Record

Scores a run's outcomes separately from execution. Linked to a run ID but
stored outside the run record.

Illustrative shape:

```json
{
  "schema_version": "v1",
  "run_id": "run-20260401-001",
  "evaluator": "automated:rubric-v1",
  "evaluated_at": "2026-04-01T11:00:00Z",
  "scores": {
    "success": true,
    "quality_score": 0.85,
    "resumability_score": 0.90,
    "human_intervention_count": 0
  },
  "notes": "acceptance criteria fully met; handoff brief complete"
}
```

Required contents:

- run ID reference
- evaluator identity (human or automated)
- success/failure against acceptance criteria
- quality and resumability scores
- evaluation timestamp

### Report Model

Reports are generated by aggregating run records and evaluation records. They
are derived artifacts and can be regenerated.

Illustrative shape:

```json
{
  "schema_version": "v1",
  "report_id": "report-20260401",
  "generated_at": "2026-04-01T12:00:00Z",
  "suite_id": "delegated-workflow-evidence-v1",
  "runs_included": 24,
  "comparisons": [
    {
      "comparison_type": "paired_task",
      "task_id": "task-001",
      "conditions": ["with-graph", "without-graph"],
      "metrics": {
        "token_delta": -2300,
        "token_delta_pct": -16.2,
        "success_rate_with_graph": 1.0,
        "success_rate_without_graph": 0.83,
        "quality_effect_size": 0.42,
        "resumability_effect_size": 0.71
      },
      "uncertainty": {
        "token_delta_ci_95": [-3100, -1500],
        "quality_effect_ci_95": [0.10, 0.74]
      }
    }
  ],
  "summary": {
    "recommendation": "graph-backed workflow reduces tokens and improves resumability for this task family",
    "null_results": [],
    "negative_results": []
  }
}
```

Reports should support:

- paired within-task comparisons
- grouped comparisons by model, topology, or task family
- success/failure rates
- token and step distributions
- effect-size summaries with uncertainty intervals
- explicit null-result and negative-result reporting

## VP4 Modeling Rule

VP4 should prefer:

- graph persistence for durable shared knowledge (unchanged from VP3)
- workflow envelopes for orchestration contracts (unchanged from VP3)
- local filesystem artifacts for experiment runs, evaluations, and reports

It should avoid:

- storing experiment data in the graph database
- inventing a hosted telemetry backend
- conflating evaluation scores with run execution records
