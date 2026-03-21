---
# Methodology Improvements Backlog
# Tracks proposed changes to the Copilot Autopilot methodology itself.
# Generated from a systematic review of all skills, agents, prompts, and instructions.
#
# Status values: todo | in-progress | done | rejected
# Priority: P0 (critical) | P1 (high) | P2 (medium) | P3 (low)

backlog:
  project: "copilot-autopilot-methodology"
  last-updated: "2026-03-10"
  themes:
    - id: TH-C
      name: "State Management & Data Model Fixes"
      status: todo
      epics:
        - id: C1
          name: "Fix story ID collisions"
          priority: P0
          status: todo
          description: >
            US1 in E1 and US1 in E2 have the same ID. depends-on: [US1] is ambiguous.
            Cross-epic dependencies are broken by design.
          proposed-change: >
            Use qualified IDs everywhere: TH1.E1.US1 or at minimum E1.US1.
            Update backlog schema, story frontmatter, and dependency resolution logic.
          files-affected:
            - .github/skills/backlog-management/SKILL.md
            - .github/skills/bdd-stories/SKILL.md
            - .github/copilot-instructions.md
            - .github/agents/orchestrator.agent.md
            - .github/agents/product-owner.agent.md
            - docs/plan/backlog.md

        - id: C2
          name: "Single source of truth for status"
          priority: P1
          status: todo
          description: >
            Status lives in both backlog.md AND each story file's frontmatter.
            Writes can fail mid-update, leaving them inconsistent.
          proposed-change: >
            Make backlog.md the sole source of truth. Remove status from story
            frontmatter entirely or make it read-only/informational. The orchestrator
            only reads/writes backlog.md.
          files-affected:
            - .github/skills/backlog-management/SKILL.md
            - .github/skills/bdd-stories/SKILL.md
            - .github/agents/orchestrator.agent.md

        - id: C3
          name: "Pure YAML backlog file"
          priority: P2
          status: todo
          description: >
            The entire backlog is deeply nested YAML inside markdown fences.
            AI agents are notoriously bad at preserving YAML indentation on edits,
            especially for large files.
          proposed-change: >
            Move the backlog to a pure YAML file (docs/plan/backlog.yaml). Drop the
            markdown wrapper. Reduces parsing complexity and makes edits less error-prone.
          files-affected:
            - docs/plan/backlog.md (rename to backlog.yaml)
            - .github/skills/backlog-management/SKILL.md
            - .github/copilot-instructions.md
            - .github/agents/orchestrator.agent.md
            - .github/agents/product-owner.agent.md
            - .github/prompts/run-autopilot.prompt.md

        - id: C4
          name: "Session log cleanup"
          priority: P3
          status: todo
          description: >
            session-log.md is append-only, grows without limit, and duplicates
            information already in backlog.md status fields.
          proposed-change: >
            Either remove the session log, or make it a structured log with rotation
            (keep last N entries). Consider using git history as the natural session log.
          files-affected:
            - .github/skills/backlog-management/SKILL.md
            - .github/agents/orchestrator.agent.md

    - id: TH-B
      name: "Agent Architecture Improvements"
      status: todo
      epics:
        - id: B1
          name: "Merge implementer + tester into developer agent"
          priority: P1
          status: todo
          description: >
            9 agents with strict boundaries means every delegation loses context.
            The implementer can't see tests, the tester can't see implementation
            rationale. Each subagent call is a cold start. This doubles delegation overhead.
          proposed-change: >
            Merge implementer + tester into a single developer agent that implements
            AND writes tests for one story. The reviewer stays separate (independent eyes).
            Consider also folding the documenter's work into the orchestrator's epic/theme
            completion steps.
          files-affected:
            - .github/agents/implementer.agent.md (merge into developer.agent.md)
            - .github/agents/tester.agent.md (merge into developer.agent.md)
            - .github/agents/orchestrator.agent.md
            - .github/agents/README.md
            - .github/copilot-instructions.md
            - .github/skills/the-copilot-build-method/SKILL.md

        - id: B2
          name: "Add explicit review-feedback loop"
          priority: P1
          status: todo
          description: >
            The reviewer reports issues but the orchestrator only has binary pass/fail.
            There's no "address review feedback" step — it falls to the troubleshooter,
            which is semantically wrong (troubleshooter is for failures, not review feedback).
          proposed-change: >
            Add re-work step in orchestrator loop: if reviewer returns REQUEST_CHANGES,
            delegate back to implementer/developer with the review feedback, then re-run
            reviewer. Max 2 iterations, then escalate.
          files-affected:
            - .github/agents/orchestrator.agent.md
            - .github/skills/the-copilot-build-method/SKILL.md

        - id: B3
          name: "Remove separate refactorer agent"
          priority: P2
          status: todo
          description: >
            The methodology creates tech debt on purpose (implementer: "keep implementations
            minimal") then pays to clean it up with a separate refactorer. This is wasteful churn.
          proposed-change: >
            Remove the separate refactorer agent. Instruct the implementer/developer to write
            clean code from the start. Run a lightweight code quality check at epic boundaries
            (part of reviewer) instead of a full refactor pass.
          files-affected:
            - .github/agents/refactorer.agent.md (remove or archive)
            - .github/agents/orchestrator.agent.md
            - .github/agents/README.md
            - .github/copilot-instructions.md
            - .github/skills/the-copilot-build-method/SKILL.md
            - .github/prompts/refactor.prompt.md

        - id: B4
          name: "Fold documenter into orchestrator"
          priority: P3
          status: todo
          description: >
            The documenter agent has the thinnest instructions of all agents.
            Its output is formulaic (changelogs, release notes). A separate agent for
            this adds delegation overhead with minimal value.
          proposed-change: >
            Fold changelog/release-note generation into the orchestrator's epic/theme
            completion steps. The orchestrator can generate these artifacts directly.
          files-affected:
            - .github/agents/documenter.agent.md (remove or archive)
            - .github/agents/orchestrator.agent.md
            - .github/agents/README.md
            - .github/copilot-instructions.md

    - id: TH-E
      name: "Complexity & Redundancy Reduction"
      status: todo
      epics:
        - id: E1
          name: "Deduplicate instructions across files"
          priority: P1
          status: todo
          description: >
            The same content (naming conventions, status values, story format, Definition
            of Done, agent table) is repeated across 6+ files. Any change requires updating
            all of them — copilot-instructions.md, README.md, SKILL.md files, agent files.
          proposed-change: >
            Apply DRY principle. Make copilot-instructions.md a concise entry point that
            references skills. Each skill is the canonical source for its topic. Agent files
            reference skills, not repeat them. Remove duplication from README (README is for
            humans, not agents).
          files-affected:
            - .github/copilot-instructions.md
            - .github/skills/the-copilot-build-method/SKILL.md
            - .github/skills/backlog-management/SKILL.md
            - .github/skills/bdd-stories/SKILL.md
            - README.md

        - id: E2
          name: "Resolve skill vs agent instruction overlap"
          priority: P2
          status: todo
          description: >
            Agent .agent.md files contain full process instructions. Skills contain
            the same information in a different format. The <!-- Skills: ... --> comments
            in agent files are not functional — they are just notes.
          proposed-change: >
            Make agents thin: they declare which skills to load and contain only agent-specific
            constraints and output format. Skills contain all reusable process instructions.
          files-affected:
            - .github/agents/*.agent.md (all agent files)
            - .github/skills/*.md (all skill files)

        - id: E3
          name: "Consolidate pass-through prompts"
          priority: P3
          status: todo
          description: >
            run-autopilot.prompt.md just says "invoke @orchestrator." review.prompt.md says
            "invoke @reviewer." These add minimal value as separate files.
          proposed-change: >
            Keep only prompts that provide real value (e.g., plan-product which sequences
            two agents). Remove pure pass-through prompts or consolidate into fewer files.
          files-affected:
            - .github/prompts/run-autopilot.prompt.md
            - .github/prompts/review.prompt.md
            - .github/prompts/troubleshoot.prompt.md

    - id: TH-A
      name: "SDLC Blind Spots"
      status: todo
      epics:
        - id: A1
          name: "Add user validation checkpoints"
          priority: P1
          status: todo
          description: >
            Vision is frozen during Phase 4. The system goes fully autonomous with no
            mechanism for user acceptance, demos, or feedback between themes. Real products
            learn from delivery.
          proposed-change: >
            Add a Phase 4.5 — User Checkpoint at theme boundaries. After theme completion,
            the orchestrator pauses and presents a demo summary. User can accept, reject,
            or amend the vision for the next VP. Vision is frozen per-theme, not globally.
          files-affected:
            - .github/agents/orchestrator.agent.md
            - .github/skills/the-copilot-build-method/SKILL.md
            - .github/copilot-instructions.md

        - id: A2
          name: "Add spike/investigation story type"
          priority: P1
          status: todo
          description: >
            The architect makes technology decisions with no way to validate risky
            technical assumptions before committing the entire backlog.
          proposed-change: >
            Add a story type 'spike' to the backlog schema. Spikes produce ADR updates
            and feasibility reports, not production code. The architect or product-owner
            can create them.
          files-affected:
            - .github/skills/bdd-stories/SKILL.md
            - .github/skills/backlog-management/SKILL.md
            - .github/agents/product-owner.agent.md
            - .github/agents/implementer.agent.md

        - id: A3
          name: "Add deployment/operational readiness"
          priority: P2
          status: todo
          description: >
            The lifecycle ends at "working software" but never addresses deploying it.
            No CI/CD, no infra-as-code, no health checks, no monitoring.
          proposed-change: >
            Add optional docs/architecture/deployment.md in Phase 2. Add a deploy
            verification step in theme completion ceremony.
          files-affected:
            - .github/skills/architecture-decisions/SKILL.md
            - .github/agents/architect.agent.md
            - .github/skills/the-copilot-build-method/SKILL.md

        - id: A4
          name: "Add non-functional requirement testing"
          priority: P2
          status: todo
          description: >
            The tester only does BDD/functional tests. NFRs (latency, throughput, memory)
            from the vision are never formally verified.
          proposed-change: >
            Extend the tester agent with an nfr-testing mode triggered at theme boundaries
            when NFRs are documented in the vision. Alternatively, add NFR acceptance criteria
            directly to stories.
          files-affected:
            - .github/agents/tester.agent.md
            - .github/skills/bdd-stories/SKILL.md

        - id: A5
          name: "Add UX/accessibility review step"
          priority: P3
          status: todo
          description: >
            For products with a UI, there is no step to validate usability or accessibility.
          proposed-change: >
            Add an optional ux-review step in the reviewer checklist when the project has
            a UI component. Not a new agent — just an extension of the reviewer's checklist.
          files-affected:
            - .github/agents/reviewer.agent.md
            - .github/skills/code-quality/SKILL.md

    - id: TH-D
      name: "Planning & Story Model Improvements"
      status: todo
      epics:
        - id: D1
          name: "Relax VP-to-TH mapping to 1:N"
          priority: P2
          status: todo
          description: >
            VP↔TH 1:1 mapping is too rigid. Cross-cutting concerns (observability, auth)
            span multiple vision phases. A vision phase might produce multiple themes.
          proposed-change: >
            Make the mapping VP:TH = 1:N. One VP can map to multiple themes.
            Update vision-ref to accept multiple VP references.
          files-affected:
            - .github/skills/the-copilot-build-method/SKILL.md
            - .github/skills/backlog-management/SKILL.md
            - .github/copilot-instructions.md

        - id: D2
          name: "Add priority field to stories"
          priority: P2
          status: todo
          description: >
            When multiple stories are eligible, the orchestrator has no way to pick the
            highest-value one. It just follows US numbering.
          proposed-change: >
            Add optional priority: high|medium|low field to stories. The orchestrator
            prefers higher-priority eligible stories. Default is medium.
          files-affected:
            - .github/skills/bdd-stories/SKILL.md
            - .github/skills/backlog-management/SKILL.md
            - .github/agents/orchestrator.agent.md

        - id: D3
          name: "Add size/complexity estimate to stories"
          priority: P2
          status: todo
          description: >
            No estimated complexity marker on stories. Large stories cause long agent
            sessions that can fail or timeout.
          proposed-change: >
            Add optional size: S|M|L field to story frontmatter. Product-owner estimates
            during planning. Orchestrator can use it for session management.
          files-affected:
            - .github/skills/bdd-stories/SKILL.md
            - .github/agents/product-owner.agent.md

        - id: D4
          name: "Fix or remove blocked status"
          priority: P3
          status: todo
          description: >
            The blocked status exists in the status table but there is no mechanism to set
            or auto-resolve it. It is never used in agent logic.
          proposed-change: >
            Either implement blocked properly (orchestrator marks ineligible stories as
            blocked and auto-transitions to todo when deps resolve) or remove it from
            the schema entirely.
          files-affected:
            - .github/skills/backlog-management/SKILL.md
            - .github/copilot-instructions.md

    - id: TH-F
      name: "Process Flexibility"
      status: todo
      epics:
        - id: F1
          name: "Proportional ceremony overhead"
          priority: P2
          status: todo
          description: >
            Even a 2-story epic must go through full ceremony: integration tests + refactor
            + reviewer approval + documenter changelog. Disproportionate for small epics.
          proposed-change: >
            Make ceremonies proportional to epic size. Epics with 3 or fewer stories skip
            the refactor and produce a minimal changelog. Full ceremony for 4+ stories.
          files-affected:
            - .github/agents/orchestrator.agent.md
            - .github/skills/the-copilot-build-method/SKILL.md

        - id: F2
          name: "Fast-track for trivial stories"
          priority: P2
          status: todo
          description: >
            A 1-AC config change or documentation fix still goes through the full
            implement → test → review pipeline.
          proposed-change: >
            Add type: standard|trivial field to stories. Trivial stories skip the reviewer
            or run a lightweight review. Product-owner assigns the type.
          files-affected:
            - .github/skills/bdd-stories/SKILL.md
            - .github/agents/orchestrator.agent.md
            - .github/agents/product-owner.agent.md

        - id: F3
          name: "Simplify theme regression testing"
          priority: P3
          status: todo
          description: >
            Theme-level regression testing adds no value beyond running the full test suite,
            which should already happen. The tester's regression mode does nothing unique.
          proposed-change: >
            Rename to "full test suite verification" and simplify the tester's modes.
            Do not pretend it is a special testing mode.
          files-affected:
            - .github/agents/tester.agent.md
            - .github/skills/the-copilot-build-method/SKILL.md

    - id: TH-G
      name: "Practical Concerns"
      status: todo
      epics:
        - id: G1
          name: "Add git workflow guidance"
          priority: P2
          status: todo
          description: >
            The methodology never mentions branches, commits, or version control workflow.
            Stories produce code but no git hygiene guidance exists.
          proposed-change: >
            Add guidance in project-setup.md for git workflow. At minimum: one commit per
            story with a conventional commit message referencing the story ID. Optionally:
            branch-per-story.
          files-affected:
            - .github/skills/architecture-decisions/SKILL.md
            - .github/agents/architect.agent.md
            - .github/agents/orchestrator.agent.md

        - id: G2
          name: "Add crash recovery protocol"
          priority: P2
          status: todo
          description: >
            If the orchestrator session dies mid-story, the story is in-progress with
            partial code. There is no rollback or recovery guidance.
          proposed-change: >
            Add recovery protocol: when orchestrator starts and finds a story in-progress,
            assess the state (partial changes?), then continue, reset, or escalate to user.
          files-affected:
            - .github/agents/orchestrator.agent.md
            - .github/skills/backlog-management/SKILL.md

        - id: G3
          name: "Add dependency management guidance"
          priority: P3
          status: todo
          description: >
            The architect picks a tech stack but there is no guidance on lockfiles,
            version pinning, or dependency updates.
          proposed-change: >
            Add a brief section in the architecture-decisions skill about dependency
            management conventions.
          files-affected:
            - .github/skills/architecture-decisions/SKILL.md
---

# Methodology Improvements Backlog

This backlog tracks proposed improvements to the Copilot Autopilot methodology itself, generated from a systematic review of all skills, agents, prompts, and instructions.

## Priority Legend

| Priority | Meaning |
|----------|---------|
| P0 | Critical — correctness bug, must fix |
| P1 | High — significant friction or design flaw |
| P2 | Medium — structural improvement |
| P3 | Low — nice to have |

## Recommended Implementation Order

### Wave 1 — Foundations (fix correctness & reduce friction)
| ID | Name | Priority |
|----|------|----------|
| C1 | Fix story ID collisions | P0 |
| B1 | Merge implementer + tester into developer agent | P1 |
| E1 | Deduplicate instructions across files | P1 |
| B2 | Add explicit review-feedback loop | P1 |
| C2 | Single source of truth for status | P1 |

### Wave 2 — Structural (improve the SDLC)
| ID | Name | Priority |
|----|------|----------|
| A1 | Add user validation checkpoints | P1 |
| A2 | Add spike/investigation story type | P1 |
| C3 | Pure YAML backlog file | P2 |
| D1 | Relax VP-to-TH mapping to 1:N | P2 |
| B3 | Remove separate refactorer agent | P2 |

### Wave 3 — Polish (flexibility & practical concerns)
| ID | Name | Priority |
|----|------|----------|
| F1 | Proportional ceremony overhead | P2 |
| F2 | Fast-track for trivial stories | P2 |
| G1 | Add git workflow guidance | P2 |
| G2 | Add crash recovery protocol | P2 |
| A3 | Add deployment/operational readiness | P2 |
| A4 | Add NFR testing | P2 |
| D2 | Add priority field to stories | P2 |
| D3 | Add size/complexity estimate | P2 |
| E2 | Resolve skill vs agent overlap | P2 |

### Wave 4 — Cleanup
| ID | Name | Priority |
|----|------|----------|
| B4 | Fold documenter into orchestrator | P3 |
| E3 | Consolidate pass-through prompts | P3 |
| C4 | Session log cleanup | P3 |
| D4 | Fix or remove blocked status | P3 |
| F3 | Simplify theme regression testing | P3 |
| A5 | Add UX/accessibility review step | P3 |
| G3 | Add dependency management guidance | P3 |
