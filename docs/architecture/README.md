# Architecture Overview — Methodology Improvements

## System Context

The "product" is the Copilot Autopilot methodology itself — a set of interconnected markdown files that define agents, skills, prompts, and conventions. The improvement project modifies these files to fix correctness bugs, reduce friction, and improve the SDLC.

## Component Map

```
.github/
├── copilot-instructions.md        ← entry point for all agents (references skills)
├── agents/
│   ├── orchestrator.agent.md      ← squad leader, autopilot loop
│   ├── developer.agent.md         ← implements + tests one story (NEW: replaces implementer + tester)
│   ├── reviewer.agent.md          ← code review, security audit
│   ├── troubleshooter.agent.md    ← diagnoses failed stories
│   ├── architect.agent.md         ← Phase 2: system design
│   ├── product-owner.agent.md     ← Phase 3: planning
│   └── archive/                   ← retired agents (implementer, tester, refactorer, documenter)
├── skills/
│   ├── the-copilot-build-method/  ← canonical lifecycle, conventions, DoD
│   ├── backlog-management/        ← canonical backlog schema, state machine
│   ├── bdd-stories/               ← canonical story format, BDD patterns
│   ├── code-quality/              ← canonical review checklist, refactoring
│   └── architecture-decisions/    ← canonical ADR format, tech stack analysis
├── prompts/
│   ├── kickstart-vision.prompt.md ← interactive vision brainstorm (keep)
│   ├── plan-product.prompt.md     ← architect → product-owner pipeline (keep)
│   └── run-autopilot.prompt.md    ← launch orchestrator (evaluate: keep or drop)
docs/
├── plan/
│   └── backlog.yaml               ← pure YAML, sole source of truth (replaces backlog.md)
└── themes/                        ← story files have no status field (status lives only in backlog.yaml)
```

## Key Architectural Decisions

### AD1 — Qualified Story IDs
All story, epic, and theme references use fully-qualified dot-notation: `TH1.E1.US1`. This eliminates ambiguity in `depends-on` references across epics.

### AD2 — Single Source of Truth
`docs/plan/backlog.yaml` is the only file where status is read or written. Story files contain the story definition (As-a/I-want, ACs, BDD scenarios) but NOT status.

### AD3 — Developer Agent (Merged)
The implementer and tester are merged into a single `developer` agent. One agent session = one story's implementation + tests. The reviewer stays independent.

### AD4 — Thin Agents, Rich Skills
Agent `.agent.md` files contain only: identity, constraints, output format, and which skills to load. All reusable process knowledge lives in skill files.

### AD5 — DRY Instruction Hierarchy
```
copilot-instructions.md (concise entry point, references skills)
  └── skills/ (canonical source for each topic)
       └── agents/ (thin, load skills, add constraints + output format)
```
No content is duplicated between these layers.

## Change Strategy

Modifications are **in-place edits to existing markdown files**. No programming language, no build system, no dependencies. Verification = read the file back and confirm it matches the specification.

## Dependency Flow Between Epics

```
E1 (Core Schema)
 ├──→ E2 (Agent Consolidation)
 │     ├──→ E3 (Deduplication)
 │     ├──→ E4 (SDLC Enhancements)
 │     └──→ E6 (Operational Concerns)
 └──→ E5 (Planning Model)
```

E1 is foundational — qualified IDs and backlog format changes affect everything downstream.
