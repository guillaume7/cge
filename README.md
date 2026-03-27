# Cognitive Graph Engine

> A local, chainable graph memory CLI for AI agents.

**CGE** gives agents a shared, repo-scoped memory they can write to, query, compress into prompt-ready context, explain, and diff over time.

It is built for one job: help agents **resume faster, hand off better, and spend fewer tokens recovering context**.

```bash
copilot "design auth service" | graph write
copilot "what depends on auth?" | graph query
graph context --task "continue auth work" --max-tokens 1200 | copilot
```

## What CGE is

CGE is a local CLI that turns working knowledge into an explicit graph instead of leaving it trapped inside transient prompt history.

That graph can hold:

- plans, ADRs, prompts, instructions, and skills
- themes, epics, stories, and backlog knowledge
- reasoning units and session summaries
- repository structure and code entities such as files and functions

The result is a practical cognitive substrate for agents: durable enough to compound on, compact enough to retrieve efficiently, and structured enough to inspect and trust.

## Why it matters

Without a shared memory layer, agents keep paying the same costs:

- they re-scan the repo
- they replay prompt history
- they lose continuity across sessions
- they struggle to justify why a retrieved context slice should be trusted

CGE reduces that waste by making memory:

- **local** — no hosted dependency for the core workflow
- **repo-scoped** — one `.graph/` workspace per repository
- **explicit** — agents choose what deserves persistence
- **chainable** — stdin/stdout friendly by design
- **inspectable** — provenance, explanation, and revision diff are built in

## Current command surface

The current branch exposes ten top-level commands:

| Command | Purpose |
| --- | --- |
| `graph init` | Create the repo-local graph workspace |
| `graph write` | Persist native graph payloads |
| `graph query` | Retrieve task-relevant structured graph results |
| `graph context` | Project compact prompt-ready context |
| `graph explain` | Show why retrieval returned specific results |
| `graph diff` | Compare two graph revisions |
| `graph stats` | Measure graph size and hygiene indicators |
| `graph hygiene` | Suggest and optionally apply graph cleanup actions |
| `graph workflow` | Bootstrap, hand off, benchmark, and dogfood delegated workflow state |
| `graph lab` | Run controlled workflow experiments and derive repo-local reports |

## Quick setup

### Install the released binary

The latest tagged release is `v0.3.0`. It currently ships a Linux AMD64 archive.

The release archive now includes the Kuzu runtime library (`libkuzu.so`) alongside the executable wrapper, so users do **not** need to install Kuzu separately for the packaged Linux release.

```bash
VERSION=v0.3.0
curl -L -o cge.tar.gz \
  "https://github.com/guillaume7/cge/releases/download/${VERSION}/cge_${VERSION}_linux_amd64.tar.gz"
tar -xzf cge.tar.gz

# keep the bundled lib/ directory next to the launcher
sudo mkdir -p /opt/cge
sudo cp -R cge_${VERSION}_linux_amd64/. /opt/cge/
sudo ln -sf /opt/cge/graph /usr/local/bin/graph

graph --help
```

### Or build from source

Build from source if you want the latest local branch state or you are iterating ahead of the packaged Linux release artifact.

Requirements:

- Go `1.22+`
- a machine able to build the embedded Kuzu-backed CLI
- access to the Go module-managed Kuzu shared library at runtime during local development builds

```bash
git clone https://github.com/guillaume7/cge.git
cd cge
go build -o graph ./cmd/graph
./graph --help
```

## Five-minute walkthrough

### 1. Initialize a repo-local graph

Run this from the repository you want to augment:

```bash
graph init
```

That creates the local workspace under `.graph/`.

### 2. Write a first payload

Create `seed.json`:

```json
{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-001",
    "timestamp": "2026-03-22T10:00:00Z",
    "revision": {
      "reason": "Seed initial repo knowledge"
    }
  },
  "nodes": [
    {
      "id": "repo:demo",
      "kind": "Repository",
      "title": "Demo Repo",
      "summary": "Example project memory root",
      "tags": ["repository"]
    },
    {
      "id": "adr:001",
      "kind": "ADR",
      "title": "Use CGE",
      "summary": "Persist agent memory in a local graph",
      "tags": ["adr"]
    }
  ],
  "edges": [
    {
      "from": "repo:demo",
      "kind": "HAS_ADR",
      "to": "adr:001"
    }
  ]
}
```

Persist it:

```bash
graph write --file seed.json
```

### 3. Query the graph

```bash
graph query --task "what ADRs are in this repo?"
```

### 4. Project compact context

```bash
graph context --task "continue work on repository architecture" --max-tokens 300
```

### 5. Explain the retrieval

```bash
graph explain --task "what ADRs are in this repo?"
```

### 6. Inspect graph health

```bash
graph stats
graph hygiene
```

### 7. Apply a reviewed hygiene plan

```bash
graph hygiene > hygiene-plan.json
# review and select action IDs in selected_action_ids first
graph hygiene --apply --file hygiene-plan.json
```

### 8. Run a delegated workflow kickoff and handoff

```bash
graph workflow init
graph workflow start --task "delegate a non-trivial repo task"
graph workflow finish --file task-outcome.json
```

### 9. Summarize a benchmark comparison

```bash
graph workflow benchmark --scenario delegated-subtask-001
```

### 10. Reproduce the repo-local experiment-lab dogfooding example

```bash
python3 .graph/lab/dogfooding/run-baseline.py
```

See
`docs/themes/TH4-experimental-evidence-lab/epics/E3-evaluation-reporting-and-dogfooding/repo-dogfooding-example.md`
for the committed harness, example artifacts, and the current limitations of the
tiny sample.

## Designed for chaining

CGE is meant to sit inside agent workflows, not beside them.

### Write from a pipeline

```bash
copilot "summarize the current auth design as a graph payload" | graph write
```

### Query from stdin

```bash
printf 'what depends on auth?\n' | graph query
```

### Feed compact context into another tool

```bash
graph context --task "continue diff implementation" --max-tokens 160 | copilot
```

## Command examples

### `graph init`

Create the repo-local workspace:

```bash
graph init
```

### `graph write`

Write a graph payload from a file, inline JSON, or stdin:

```bash
graph write --file seed.json
graph write --payload '{"schema_version":"v1","metadata":{"agent_id":"dev","session_id":"s1","timestamp":"2026-03-22T10:00:00Z"},"nodes":[],"edges":[]}'
cat seed.json | graph write
```

### `graph query`

Retrieve structured graph results for a task:

```bash
graph query --task "what implements graph diff?"
printf 'what implements graph diff?\n' | graph query
```

### `graph context`

Project prompt-ready context under an approximate token budget:

```bash
graph context --task "continue work on diff and machine contracts" --max-tokens 160
```

### `graph explain`

Return ranking reasons, graph paths, and provenance:

```bash
graph explain --task "what implements graph diff?"
```

### `graph diff`

Compare two graph revision anchors:

```bash
graph diff --from <older-anchor> --to <newer-anchor>
```

### `graph stats`

Inspect graph size and hygiene indicators without mutating the workspace:

```bash
graph stats
```

### `graph hygiene`

Generate a cleanup plan, then optionally apply a reviewed subset of actions:

```bash
graph hygiene
graph hygiene > hygiene-plan.json
graph hygiene --apply --file hygiene-plan.json
```

### `graph workflow`

Bootstrap delegated workflow assets, create kickoff briefs, persist handoff outcomes, and summarize local benchmark artifacts:

```bash
graph workflow init
graph workflow start --task "review the repo dogfooding flow" --max-tokens 1200
graph workflow finish --file task-outcome.json
graph workflow benchmark --scenario delegated-subtask-001
```

## Stable machine-readable contract

Operational commands return structured JSON envelopes.

Success shape:

```json
{
  "schema_version": "v1",
  "command": "query",
  "status": "ok",
  "result": {}
}
```

Error shape:

```json
{
  "schema_version": "v1",
  "command": "query",
  "status": "error",
  "error": {
    "category": "validation_error",
    "type": "input_error",
    "message": "task text is required"
  }
}
```

This contract makes CGE predictable inside shell pipelines and custom agent tooling.

## What belongs in the graph?

The MVP is designed to represent both project-operating knowledge and code knowledge, including:

- repositories, directories, files, functions, methods, types, classes, and variables
- ADRs, plans, prompts, instructions, and skills
- themes, epics, stories, and backlog artifacts
- reasoning units and agent sessions

## Documentation

- Vision: [`docs/vision_of_product/VP1-MVP/`](docs/vision_of_product/VP1-MVP/)
- Architecture: [`docs/architecture/`](docs/architecture/)
- ADRs: [`docs/ADRs/`](docs/ADRs/)
- Campaign experiments and reports: [`docs/experiments/`](docs/experiments/)

## Releases

- Latest release: [`v0.3.0`](https://github.com/guillaume7/cge/releases/tag/v0.3.0)
- `v0.3.0` ships the delegated workflow, experimental lab, token instrumentation, and precision-governed advisory kickoff surfaces from TH3 through TH6.
- Repository: [`guillaume7/cge`](https://github.com/guillaume7/cge)

## Status

CGE is intentionally still primitive.

That is a feature, not an apology.

This first line of releases is about delivering a **small, dependable, local graph substrate** that agents can immediately build on top of.

## License

MIT
