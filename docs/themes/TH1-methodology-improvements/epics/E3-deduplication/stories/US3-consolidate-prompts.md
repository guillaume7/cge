---
id: TH1.E3.US3
title: "Consolidate pass-through prompts"
agents: [developer, reviewer]
skills: [the-copilot-build-method]
acceptance-criteria:
  - AC1: "Pass-through prompts that only invoke one agent are removed or consolidated"
  - AC2: "Prompts that sequence multiple agents or add real logic are kept"
  - AC3: "At minimum kickstart-vision and plan-product prompts are preserved"
depends-on: [TH1.E3.US2]
---

# TH1.E3.US3 — Consolidate Pass-Through Prompts

**As a** template maintainer, **I want** to remove prompts that just invoke a single agent with no added logic, **so that** the prompt directory contains only meaningful orchestration files.

## Acceptance Criteria

- [ ] AC1: Pass-through prompts that only invoke one agent are removed or consolidated
- [ ] AC2: Prompts that sequence multiple agents or add real logic are kept
- [ ] AC3: At minimum `kickstart-vision` and `plan-product` prompts are preserved

## BDD Scenarios

### Scenario: Pass-through prompts removed
- **Given** the `.github/prompts/` directory
- **When** I list the prompt files
- **Then** pure pass-through prompts (e.g., `review.prompt.md`, `troubleshoot.prompt.md`) are removed

### Scenario: Valuable prompts preserved
- **Given** the `.github/prompts/` directory
- **When** I list the prompt files
- **Then** `kickstart-vision.prompt.md` and `plan-product.prompt.md` are present

### Scenario: Multi-agent prompts kept
- **Given** `plan-product.prompt.md`
- **When** I read its content
- **Then** it sequences @architect then @product-owner (real orchestration logic)
