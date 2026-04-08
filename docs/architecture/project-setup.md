# Project Setup — Cognitive Graph Engine VP1 + VP2 + VP3 + VP4 + VP8

## Repository Structure

Recommended Go project layout:

```text
cmd/
  graph/
    main.go
internal/
  app/
    initcmd/
    writecmd/
    querycmd/
    contextcmd/
    explaincmd/
    diffcmd/
    statscmd/
    hygienecmd/
    workflowcmd/
    benchmarkcmd/
    labcmd/
    workflow/
  domain/
    entity/
    retrieval/
    payload/
    revision/
    hygiene/
    stats/
    evaluator/
    decision/
    attribution/
    lab/
      orchestrator/
      ledger/
      evaluation/
      report/
  infra/
    repo/
    kuzu/
    index/
    workflowassets/
    benchmarks/
    labstore/
docs/
  architecture/
  ADRs/
  vision_of_product/
testdata/
  payloads/
  fixtures/
```

## Local Graph Workspace

Recommended repo-local workspace:

```text
.graph/
  config.json
  kuzu/
  index/
  workflow/
    manifest.json
  benchmarks/
  attribution/
  lab/
    suite.json
    conditions.json
    runs/
    evaluations/
    reports/
  tmp/
```

Notes:

- `.graph/` should be repo-scoped
- `.graph/` should be added to `.gitignore` so local graph state is not committed
- `kuzu/` stores the graph DB files
- `index/` stores the derived text index
- `workflow/manifest.json` stores workflow asset installation and refresh state
- `benchmarks/` stores local benchmark scenarios and machine-readable reports
- `attribution/` stores evaluator-loop attribution records for lab analysis
- `lab/` stores the experiment workspace: suite manifests, conditions, run
  records, evaluations, and generated reports
- `config.json` stores schema version and repo identity metadata

## Build and Test Commands

Recommended baseline commands:

```bash
go build ./...
go test ./...
```

Optional local run pattern:

```bash
go run ./cmd/graph --help
```

## Dependency Management

- Use Go modules
- Commit `go.mod` and `go.sum`
- Keep direct dependencies narrow and explicit
- Avoid adding framework-heavy abstractions unless a later ADR justifies them

## Testing Strategy

Minimum test layers:

- command tests for CLI behavior
- payload validation tests
- repository initialization tests
- Kuzu integration tests for write/query behavior
- retrieval ranking tests with fixtures
- context projection golden tests
- diff/explain output tests
- graph stats tests for indicator correctness and edge cases
- hygiene suggestion/apply tests for duplicates, orphans, and contradictions
- workflow kickoff/handoff contract tests
- workflow asset refresh/idempotency tests
- wrapper/hook transparency tests
- benchmark harness/report determinism tests
- lab init/run/report command tests
- run ledger write/read/immutability tests
- evaluation record creation and blinded presentation tests
- report generation determinism and statistical correctness tests
- condition assignment and randomization tests
- context evaluator scoring tests for relevance, consistency, and usefulness
- decision engine threshold and outcome selection tests
- attribution record generation, persistence, and loading tests
- evaluator loop integration tests through `graph context` and `workflow start`
- harness-aware lab condition assignment tests

## Setup Principles

- one binary
- no external services required
- reproducible local development
- easy shell-based experimentation with stdin/stdout
- explicit local workflows for stats and hygiene without background daemons
- explicit delegated-task workflow support without a local service process
- explicit experiment lab lifecycle through `graph lab` commands
- evaluation-before-trust discipline through the evaluator loop for context
  and workflow paths

## No Deployment Document For MVP

The MVP is a local CLI without a service deployment target, so
`docs/architecture/deployment.md` is intentionally omitted.

This remains true for VP2, VP3, VP4, and VP8.
