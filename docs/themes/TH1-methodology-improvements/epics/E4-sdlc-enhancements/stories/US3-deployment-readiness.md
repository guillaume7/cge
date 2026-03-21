---
id: TH1.E4.US3
title: "Add deployment readiness guidance"
agents: [developer, reviewer]
skills: [architecture-decisions, the-copilot-build-method]
acceptance-criteria:
  - AC1: "architecture-decisions skill includes deployment.md as an optional output"
  - AC2: "Architect agent instructions mention deployment.md in the process"
  - AC3: "Theme completion ceremony includes a deploy verification step"
depends-on: []
---

# TH1.E4.US3 — Add Deployment Readiness Guidance

**As a** product builder, **I want** deployment guidance included in the architecture phase, **so that** the product can actually be deployed, not just built.

## Acceptance Criteria

- [ ] AC1: `architecture-decisions` skill includes `deployment.md` as an optional output
- [ ] AC2: Architect agent instructions mention `deployment.md` in the process
- [ ] AC3: Theme completion ceremony includes a deploy verification step

## BDD Scenarios

### Scenario: Architecture doc structure includes deployment
- **Given** the architecture document structure in `architecture-decisions` skill
- **When** I read the file listing
- **Then** `deployment.md` is listed as an optional file

### Scenario: Architect considers deployment
- **Given** the architect agent's process steps
- **When** it produces architecture docs
- **Then** it optionally creates `docs/architecture/deployment.md` covering CI/CD, infra, and health checks

### Scenario: Theme ceremony checks deployment
- **Given** the orchestrator's theme completion steps
- **When** I read the release readiness checks
- **Then** there's a step to verify deployment readiness (if deployment.md exists)
