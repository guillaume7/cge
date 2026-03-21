---
id: TH1.E6.US3
title: "Add dependency management guidance"
agents: [developer, reviewer]
skills: [architecture-decisions]
acceptance-criteria:
  - AC1: "architecture-decisions skill includes a dependency management section"
  - AC2: "Covers lockfiles, version pinning, and update strategy"
depends-on: []
---

# TH1.E6.US3 — Add Dependency Management Guidance

**As a** product builder, **I want** dependency management guidance in the methodology, **so that** the architect's tech stack choices include practical package management conventions.

## Acceptance Criteria

- [ ] AC1: `architecture-decisions` skill includes a dependency management section
- [ ] AC2: Covers lockfiles, version pinning, and update strategy

## BDD Scenarios

### Scenario: Dependency section exists
- **Given** the `architecture-decisions` skill
- **When** I read the sections
- **Then** there's a "Dependency Management" section

### Scenario: Key topics covered
- **Given** the dependency management section
- **When** I read the guidance
- **Then** it mentions lockfiles, version pinning, and a strategy for updates

### Scenario: Architect considers dependencies
- **Given** the architect's tech stack analysis checklist
- **When** I read the evaluation dimensions
- **Then** dependency management is included
