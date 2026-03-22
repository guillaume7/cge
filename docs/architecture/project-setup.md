# Project Setup — VP1 Cognitive Graph Engine MVP

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
  domain/
    entity/
    retrieval/
    payload/
    revision/
  infra/
    repo/
    kuzu/
    index/
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
  tmp/
```

Notes:

- `.graph/` should be repo-scoped
- `.graph/` should be added to `.gitignore` so local graph state is not committed
- `kuzu/` stores the graph DB files
- `index/` stores the derived text index
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

## Setup Principles

- one binary
- no external services required
- reproducible local development
- easy shell-based experimentation with stdin/stdout

## No Deployment Document For MVP

The MVP is a local CLI without a service deployment target, so
`docs/architecture/deployment.md` is intentionally omitted.
