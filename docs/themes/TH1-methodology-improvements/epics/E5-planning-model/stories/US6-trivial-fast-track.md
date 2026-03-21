---
id: TH1.E5.US6
title: "Fast-track trivial stories"
agents: [developer, reviewer]
skills: [bdd-stories, the-copilot-build-method]
acceptance-criteria:
  - AC1: "Trivial stories (type: trivial) skip full review or run lightweight review"
  - AC2: "Orchestrator checks story type and adjusts pipeline accordingly"
  - AC3: "Product-owner assigns type during planning"
depends-on: [TH1.E5.US2]
---

# TH1.E5.US6 — Fast-Track Trivial Stories

**As a** product builder, **I want** trivial stories (config changes, doc fixes) to skip the full review pipeline, **so that** simple work doesn't incur disproportionate overhead.

## Acceptance Criteria

- [ ] AC1: Trivial stories (`type: trivial`) skip full review or run lightweight review
- [ ] AC2: Orchestrator checks story type and adjusts pipeline accordingly
- [ ] AC3: Product-owner assigns type during planning

## BDD Scenarios

### Scenario: Trivial story skips reviewer
- **Given** a story with `type: trivial`
- **When** the orchestrator processes it
- **Then** it delegates to developer but skips the full reviewer step (or runs lightweight review)

### Scenario: Standard story gets full pipeline
- **Given** a story with `type: standard` (or no type field)
- **When** the orchestrator processes it
- **Then** it runs the full developer → reviewer pipeline

### Scenario: Product-owner assigns type
- **Given** the product-owner creating stories during planning
- **When** it creates a config-change or doc-only story
- **Then** it can set `type: trivial` in the story frontmatter
