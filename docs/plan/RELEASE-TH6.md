# Release: Precision-Governed Advisory Kickoff (TH6)

## Summary

TH6 makes `graph workflow start` more precise, more inspectable, and easier to override without taking control away from the operator. Kickoff is now family-aware, confidence-aware, provenance-aware, and explicitly advisory.

## Epics Delivered

### E1 — Task-Family Retrieval Policies
- `graph workflow start` now classifies requests into kickoff families such as write-producing, troubleshooting, verification, reporting, workflow-context, and ambiguous tasks
- kickoff retrieval applies family-specific entity allowlists and suppressions so the envelope favors the right context for the task at hand
- reporting and synthesis tasks abstain from kickoff by default instead of forcing unnecessary context

### E2 — Confidence, Provenance, and Abstention
- kickoff confidence is derived from graph availability, result quality, ambiguity, and graph sparsity
- each returned kickoff entity includes a compact machine-readable inclusion reason
- low-confidence situations now recommend minimal kickoff or pull-on-demand guidance instead of pretending confidence that is not there

### E3 — Freedom-Preserving Kickoff Controls
- operators can explicitly request `auto`, `minimal`, or `none` kickoff modes through the CLI
- kickoff now emits an advisory state that explains the effective mode, confidence, and recommended next step without breaking existing workflow compatibility
- sparse repositories and ambiguous tasks degrade gracefully instead of over-injecting low-signal context

## Breaking Changes

None. TH6 extends `graph workflow start` conservatively and keeps compatibility with existing workflow envelopes while adding new machine-readable advisory fields.

## Migration Notes

- Existing workflow integrations can continue consuming the old readiness surfaces; new `kickoff.*` advisory fields are additive.
- Consumers that want explicit operator control can start passing `--kickoff-mode auto|minimal|none`.
- Reporting flows should expect abstention by default and request graph context only when needed.

## Architecture Decisions

- **ADR-016**: Precision-governed advisory kickoff

## Verification

- `go test ./internal/app/workflow ./internal/app/workflowcmd`
- `go test ./...`

## Files Added/Modified

### New Surfaces
- `docs/vision_of_product/VP6-precision-governed-advisory-kickoff/README.md`
- `docs/ADRs/ADR-016-precision-governed-advisory-kickoff.md`
- `docs/themes/TH6-precision-governed-advisory-kickoff/...`
- `docs/plan/RELEASE-TH6.md`

### Modified Surfaces
- `internal/app/workflow/start.go`
- `internal/app/workflowcmd/command.go`
- `internal/app/contextprojector/projector.go`
- `internal/app/workflow/start_test.go`
- `internal/app/workflowcmd/command_test.go`
- `docs/plan/backlog.yaml`
- `README.md`
