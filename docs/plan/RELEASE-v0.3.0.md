# Release v0.3.0

## Summary

`v0.3.0` turns CGE into a graph-backed workflow and experimentation substrate, not just a graph memory CLI. This release ships the delegated workflow surfaces from TH3, the experimental evidence lab from TH4, the token-instrumented lab line from TH5, and the precision-governed advisory kickoff improvements from TH6.

## Highlights

- `graph workflow init`, `start`, `finish`, and `benchmark` for graph-backed delegated work
- repo-local workflow assets, hooks, and dogfooding helpers
- `graph lab` experiment initialization, execution, and reporting surfaces
- token-aware experiment artifacts and telemetry-oriented lab support
- family-aware, confidence-aware, provenance-aware workflow kickoff with explicit operator controls

## Themes Delivered

### TH3 — Graph-Backed Delegated Workflow
- bootstraps repo-local workflow assets and manifest state
- generates readiness-aware kickoff envelopes and revision-aware handoff briefs
- adds benchmark summaries and repo dogfooding helpers

### TH4 — Experimental Evidence Lab
- introduces `graph lab` experiment scaffolding and schemas
- records controlled runs and evaluation artifacts locally
- emits paired-comparison reports with uncertainty-aware reporting

### TH5 — Token-Instrumented Lab
- extends the lab line with token-oriented telemetry and measurement artifacts
- establishes the measured execution baseline used to evaluate graph-assisted workflows

### TH6 — Precision-Governed Advisory Kickoff
- classifies kickoff requests into task families
- computes family-aware kickoff confidence and inclusion reasons
- adds explicit `--kickoff-mode auto|minimal|none` controls with graceful degradation

## Breaking Changes

None. This release expands the command surface and workflow contracts additively.

## Verification

- `go test ./...`

## Notes

- Linux AMD64 release archives include the bundled `libkuzu.so` runtime next to the launcher.
- Existing repos can adopt the new workflow/lab surfaces incrementally; the graph workspace remains local and repo-scoped.
