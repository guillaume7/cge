# Release: Methodology Improvements

## Summary
Theme TH1 delivers a comprehensive overhaul of the Copilot Autopilot methodology — consolidating agents from 9 to 6, deduplicating instructions, adding planning flexibility (priority, size, story types), and introducing operational safeguards (crash recovery, git workflow, bounded session log).

## Epics Delivered
- E1: Core Schema & State — qualified IDs, single source of truth for status, pure YAML backlog
- E2: Agent Consolidation — developer agent (impl+test), review rework loop, removed refactorer, folded documenter
- E3: Instruction Deduplication — DRY instructions, thin agents (≤50 lines), removed pass-through prompts
- E4: SDLC Enhancements — user checkpoints, spike stories, deployment readiness, NFR testing, UX/a11y review
- E5: Planning Model & Flexibility — 1:N VP mapping, priority/size fields, removed blocked status, proportional ceremony
- E6: Operational Concerns — git workflow, crash recovery, dependency management, bounded session log

## Breaking Changes
- `backlog.md` removed — use `backlog.yaml` exclusively
- `blocked` status removed — stories are simply `todo` until eligible
- Agent names changed: `implementer`/`tester` → `developer`; `refactorer`/`documenter` removed
- 4 prompt files removed: `review.prompt.md`, `review-docs.prompt.md`, `refactor.prompt.md`, `troubleshoot.prompt.md`
- Story frontmatter gains optional `type`, `priority`, and `size` fields

## Migration Notes
- Replace any `backlog.md` references with `backlog.yaml`
- Replace `@implementer`/`@tester` agent references with `@developer`
- Remove `@refactorer`/`@documenter` references (responsibilities absorbed by reviewer and orchestrator)
- Remove `status` field from story file frontmatter (status only in backlog.yaml)
- Remove `blocked` from any status value lists
