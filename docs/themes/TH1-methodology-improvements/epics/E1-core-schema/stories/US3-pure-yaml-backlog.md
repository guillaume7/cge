---
id: TH1.E1.US3
title: "Convert backlog to pure YAML"
agents: [developer, reviewer]
skills: [backlog-management]
acceptance-criteria:
  - AC1: "Backlog template is a .yaml file, not .md with YAML frontmatter"
  - AC2: "All agent/skill/prompt references updated from backlog.md to backlog.yaml"
  - AC3: "backlog-management skill documents the pure YAML format"
  - AC4: "Human-readable summary table moves to a separate file or is dropped"
depends-on: [TH1.E1.US2]
---

# TH1.E1.US3 — Convert Backlog to Pure YAML

**As a** methodology user, **I want** the backlog to be a pure YAML file, **so that** AI agents can edit it without YAML-inside-markdown parsing errors and indentation issues.

## Acceptance Criteria

- [ ] AC1: Backlog template is a `.yaml` file, not `.md` with YAML frontmatter
- [ ] AC2: All agent/skill/prompt references updated from `backlog.md` to `backlog.yaml`
- [ ] AC3: `backlog-management` skill documents the pure YAML format
- [ ] AC4: Human-readable summary table moves to a separate README or is dropped

## BDD Scenarios

### Scenario: Backlog file is pure YAML
- **Given** the file at `docs/plan/backlog.yaml`
- **When** I parse it with a YAML parser
- **Then** it parses successfully without needing to strip markdown fences

### Scenario: Orchestrator references backlog.yaml
- **Given** the orchestrator agent instructions
- **When** I search for backlog file references
- **Then** all references point to `docs/plan/backlog.yaml` (not `backlog.md`)

### Scenario: Product-owner references backlog.yaml
- **Given** the product-owner agent instructions
- **When** I search for backlog file references
- **Then** all references point to `docs/plan/backlog.yaml`

### Scenario: No markdown wrapper in backlog
- **Given** the backlog file template in `backlog-management/SKILL.md`
- **When** I read the schema example
- **Then** it shows a pure YAML file with no `---` markdown frontmatter delimiters or markdown body
