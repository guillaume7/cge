# Release: Graph-Backed Delegated Workflow (TH3)

## Summary

TH3 turns CGE from a graph memory CLI into a graph-backed delegated-workflow toolchain for agents. Repositories can now bootstrap local workflow assets, create compact kickoff briefs, persist revision-aware handoffs, record benchmark comparisons, and dogfood the flow through explicit repo-local hooks.

## Epics Delivered

### E1 — Workflow Bootstrap and Assets
- `graph workflow init` bootstraps `.graph/workflow/` and writes a machine-readable manifest
- baseline repo knowledge is seeded from standard repo artifacts through the normal graph write path
- managed prompt/instruction/skill/hook assets refresh atomically while preserving declared overrides

### E2 — Delegated Kickoff and Handoff
- `graph workflow start` inspects readiness and recommends whether to proceed, bootstrap, inspect hygiene, or gather context
- kickoff envelopes provide bounded graph context and a prompt-ready delegation brief
- `graph workflow finish` persists delegated outcomes, returns before/after revision anchors, and emits a machine-readable handoff brief or explicit no-op result

### E3 — Benchmark and Repo Dogfooding
- benchmark scenarios and runs are stored locally under `.graph/benchmarks/` as machine-readable artifacts
- `graph workflow benchmark` summarizes with-graph vs without-graph runs conservatively and flags incomplete or non-comparable evidence
- this repository's own prompts, agents, and hooks now steer non-trivial delegated subtasks through explicit graph-backed kickoff and handoff helpers with opt-out support

## Breaking Changes

None. TH3 adds a new top-level `graph workflow` command group and repo-local dogfooding helpers without breaking existing command contracts.

## Migration Notes

- Existing repos should run `graph workflow init` to install the delegated-workflow manifest and managed assets.
- The repo-local helper script `bash .github/hooks/scripts/repo-delegated-workflow.sh` is optional and remains easy to opt out of with `--opt-out` or `CGE_REPO_WORKFLOW_OPTOUT=1`.
- Benchmark evidence remains local to `.graph/benchmarks/` and is not written into graph memory.

## Architecture Decisions

- **ADR-009**: Thin delegated workflow orchestration
- **ADR-010**: Composable workflow snippets and hooks
- **ADR-011**: Delegated workflow benchmark surfaces

## Verification

- `go test ./...`
- `go build ./...`
- `PATH="$PWD/.tmp/bin:$PATH" bash .github/hooks/scripts/verify-repo-delegated-workflow.sh --artifacts-dir .tmp/repo-workflow-verify-fixed`

## Files Added/Modified

### New Surfaces
- `internal/app/workflow/start.go`, `finish.go`, `benchmark.go`
- `internal/infra/benchmarks/store.go`
- `.github/hooks/scripts/repo-delegated-workflow.sh`
- `.github/hooks/scripts/announce-delegated-workflow.sh`
- `.github/hooks/scripts/verify-repo-delegated-workflow.sh`
- `.github/hooks/autopilot-lifecycle.json`

### Modified Surfaces
- `internal/app/workflow/service.go`, `seed.go`, `assets.go`
- `internal/app/workflowcmd/command.go`
- `internal/infra/kuzu/store.go`
- `.github/copilot-instructions.md`
- `.github/prompts/run-autopilot.prompt.md`
- `.github/agents/orchestrator.agent.md`, `developer.agent.md`, `reviewer.agent.md`, `troubleshooter.agent.md`
- `README.md`
