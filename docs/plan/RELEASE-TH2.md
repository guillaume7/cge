# Release: Graph Hygiene and Stats (TH2)

## Summary

TH2 transforms the Cognitive Graph Engine from a memory primitive into a **maintainable memory system**. Agents can now measure graph quality, detect structural disorder, and explicitly apply approved cleanup — all through the existing local, repo-scoped, machine-readable CLI.

Two new commands were added:
- **`graph stats`** — snapshot metrics and cognitive health indicators
- **`graph hygiene`** — suggest-first cleanup with explicit apply

## Epics Delivered

### E1 — Graph Stats Snapshot
- `graph stats` returns node count, relationship count, and five cognitive health indicators
- Indicators: duplication_rate, orphan_rate, contradictory_facts, density_score, clustering_score
- All computed on-demand from current graph snapshot — no persisted metrics backend
- Read-only: never mutates graph state

### E2 — Hygiene Suggestions
- `graph hygiene` (default: suggest mode) detects three categories of graph disorder:
  - **Orphan nodes** — entities with zero relationships
  - **Near-identical duplicates** — same kind + normalized title/body fingerprint
  - **Contradictory facts** — same subject with conflicting values
- Returns structured plan with snapshot_anchor, suggestions, and typed actions
- Each action includes action_id, type, target_ids, canonical_node_id, and explanation
- Suggest mode is read-only and non-mutating

### E3 — Hygiene Apply and Revision Safety
- `graph hygiene --apply --file <plan>` executes only explicitly selected actions
- Returns revision anchor enabling before/after diff inspection via `graph diff`
- Stale plans (snapshot changed since suggest) rejected with structured error
- Unsupported actions and missing targets rejected with structured error
- All rejected plans leave graph completely unchanged

## Breaking Changes

None. TH2 adds new commands without modifying existing command contracts.

## Migration Notes

No migration needed. The `graph stats` and `graph hygiene` commands are additive and work with any existing graph workspace created by TH1.

## Architecture Decisions

- **ADR-007**: Suggest-first graph hygiene workflow with explicit apply
- **ADR-008**: On-demand graph stats computed from current snapshot

## Test Coverage

- 6 stats command tests (snapshot counts, health indicators, empty graph, error cases, file output)
- 21 hygiene command tests (orphan/duplicate/contradiction detection, plan envelope, apply, diff compatibility, stale/unsafe rejection, graph preservation)
- Full regression suite: all 12 test packages pass

## Files Added/Modified

### New Files
- `internal/app/statscmd/command.go` — graph stats command
- `internal/app/statscmd/command_test.go` — stats tests
- `internal/infra/kuzu/stats.go` — stats computation from Kuzu store
- `internal/app/hygienecmd/command.go` — graph hygiene command
- `internal/app/hygienecmd/command_test.go` — hygiene tests
- `internal/app/graphhealth/graphhealth.go` — shared analysis, detection, and apply engine
- `internal/infra/kuzu/sync.go` — graph replacement and revision operations

### Modified Files
- `internal/app/graphcmd/root.go` — registered stats and hygiene subcommands
