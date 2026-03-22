# Changelog

## Epic E1 — Core Schema & State

### Stories Completed
- TH1.E1.US1: Introduce qualified story IDs
- TH1.E1.US2: Single source of truth for status
- TH1.E1.US3: Convert backlog to pure YAML

### Key Changes
- All entity IDs use fully-qualified dot-notation (TH1.E1.US1)
- Status lives only in backlog.yaml, not in story files
- Backlog converted from markdown+YAML frontmatter to pure YAML

### Files Modified
- docs/plan/backlog.yaml (created), docs/plan/backlog.md (deleted)
- .github/skills/backlog-management/SKILL.md, .github/skills/bdd-stories/SKILL.md

## Epic E2 — Agent Consolidation

### Stories Completed
- TH1.E2.US1: Create developer agent (merged implementer + tester)
- TH1.E2.US2: Add review feedback loop to orchestrator
- TH1.E2.US3: Remove refactorer agent
- TH1.E2.US4: Fold documenter into orchestrator

### Key Changes
- Agent count reduced from 9 to 6
- Developer agent handles both implementation and testing
- Orchestrator handles changelog/release notes (was documenter)
- Review rework loop: max 2 iterations before escalation

### Files Modified
- .github/agents/developer.agent.md (created)
- .github/agents/orchestrator.agent.md, .github/agents/reviewer.agent.md
- .github/agents/archive/ (implementer, tester, refactorer, documenter archived)

## Epic E3 — Instruction Deduplication

### Stories Completed
- TH1.E3.US1: DRY copilot-instructions.md
- TH1.E3.US2: Make agents thin, skills canonical
- TH1.E3.US3: Consolidate pass-through prompts

### Key Changes
- copilot-instructions.md reduced from ~190 to ~50 lines
- All 6 agents slimmed to ≤50 body lines (process details in skills)
- 4 pass-through prompts removed (review, review-docs, refactor, troubleshoot)

### Files Modified
- .github/copilot-instructions.md
- All 6 agent files (.github/agents/*.agent.md)
- .github/prompts/ (4 files removed, 3 kept)

## Epic E4 — SDLC Enhancements

### Stories Completed
- TH1.E4.US1: User validation checkpoints at theme boundaries
- TH1.E4.US2: Spike story type
- TH1.E4.US3: Deployment readiness guidance
- TH1.E4.US4: NFR testing guidance
- TH1.E4.US5: UX/accessibility review checklist

### Key Changes
- User checkpoints at theme completion (accept/reject/amend)
- Story types: standard, trivial, spike
- deployment.md as optional architecture output
- NFR acceptance criteria guidance with examples
- UX/accessibility checklist for UI projects

### Files Modified
- .github/skills/the-copilot-build-method/SKILL.md
- .github/skills/bdd-stories/SKILL.md
- .github/skills/architecture-decisions/SKILL.md
- .github/skills/code-quality/SKILL.md
- .github/agents/orchestrator.agent.md, architect.agent.md, product-owner.agent.md

## Epic E5 — Planning Model & Flexibility

### Stories Completed
- TH1.E5.US1: VP:TH mapping relaxed to 1:N
- TH1.E5.US2: Priority field (high/medium/low)
- TH1.E5.US3: Size field (S/M/L)
- TH1.E5.US4: Removed blocked status
- TH1.E5.US5: Proportional ceremony overhead
- TH1.E5.US6: Fast-track trivial stories
- TH1.E5.US7: Simplified regression testing

### Key Changes
- One VP can produce multiple themes (1:N mapping)
- Stories support optional priority and size fields
- Blocked status removed (was dead code)
- Small epics (≤3 stories) get lightweight ceremony
- Trivial stories skip full reviewer
- Theme testing renamed to "full test suite verification"

### Files Modified
- .github/skills/the-copilot-build-method/SKILL.md
- .github/skills/bdd-stories/SKILL.md
- .github/skills/backlog-management/SKILL.md
- .github/agents/orchestrator.agent.md, developer.agent.md

## Epic E6 — Operational Concerns

### Stories Completed
- TH1.E6.US1: Git workflow guidance
- TH1.E6.US2: Crash recovery protocol
- TH1.E6.US3: Dependency management guidance
- TH1.E6.US4: Session log cleanup

### Key Changes
- Conventional commits: `feat(TH1.E1.US1): description`
- Optional branch-per-epic strategy documented
- Crash recovery: detect stale in-progress, assess, decide (continue/reset/escalate)
- Dependency management: lockfiles, version pinning, update strategy
- Session log bounded to 50 entries; git history as primary audit trail

### Files Modified
- .github/skills/architecture-decisions/SKILL.md
- .github/skills/backlog-management/SKILL.md
- .github/agents/orchestrator.agent.md

## TH1.E1 — Workspace and CLI Foundation

- bootstrapped the Go-based `graph` CLI and repo-local `.graph/` workspace
- added the chainable MVP command surface with repo discovery and stdin support
- defined the versioned native payload envelope and structured validation errors

## TH1.E2 — Graph Persistence and Provenance

- replaced stub persistence with real Kuzu-backed entity and relationship storage
- enforced provenance for reasoning units and agent sessions within the entity-centric model
- added rewrite-safe revision anchors for future graph diff support

## TH1.E3 — Hybrid Retrieval and Context

- implemented offline hybrid retrieval by combining graph structure with a rebuildable local text-relevance index
- added compact token-budgeted context projection with prioritization that preserves critical relationships and provenance
- introduced structured `graph explain` output with ranking reasons, graph paths, and provenance traces for trust/debugging

## TH1.E4 — Diff, Trust, and Agent Interoperability

- standardized the machine-readable JSON contract across write, query, context, explain, and diff
- implemented revision-anchor graph diffs with added, updated, removed, and retagged entity/relationship reporting
- verified end-to-end stdin/stdout chaining for payload writes, task queries, context consumption, and structured error flows
