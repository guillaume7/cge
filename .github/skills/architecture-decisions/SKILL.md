---
name: architecture-decisions
description: 'ADR format, tech stack analysis methodology, component boundary definition, system design patterns. Use when: creating ADRs, choosing tech stack, defining architecture, designing components, making system design decisions.'
---

# Architecture Decisions Skill

Architecture Decision Records live in
`docs/architecture/adrs/ADR-<NNN>-<slug>.md`.

### Template

```markdown
# ADR-<NNN>: <Title>

## Status
Proposed | Accepted | Deprecated | Superseded by ADR-<NNN>

## Context
<What is the issue that motivates this decision? What forces are at play?>

## Decision
<What is the change that we're proposing and/or doing?>

## Consequences

### Positive
- <benefit 1>
- <benefit 2>

### Negative
- <trade-off 1>
- <trade-off 2>

### Risks
- <risk and mitigation>

## Alternatives Considered

### <Alternative 1>
- Pros: <advantages>
- Cons: <disadvantages>
- Rejected because: <reason>

### <Alternative 2>
- Pros: <advantages>
- Cons: <disadvantages>
- Rejected because: <reason>
```

### ADR Lifecycle

1. **Proposed** → Under discussion, not yet committed
2. **Accepted** → Decision made, implementation proceeds
3. **Deprecated** → No longer relevant (explain why)
4. **Superseded** → Replaced by a newer ADR (link to it)

### ADR Immutability

Once an ADR's status is `Accepted` **and** its associated theme is `locked` (see skill: `the-copilot-build-method` — Immutability Policy), the ADR document is **frozen**:

- **Do not edit** the body, decision, or consequences of a locked ADR
- To change or revise a decision, **create a new ADR** with the next sequential number
- In the new ADR, reference the old one in its context (e.g., "Supersedes ADR-001")
- Update the old ADR's `Status` line to `Superseded by ADR-<NNN>` — this is the **only permitted edit** to a locked ADR

## Tech stack analysis

When choosing technologies, evaluate along these dimensions:

| Dimension | Questions |
|:---|:---|
| Fitness | Does it solve the actual problem from the vision? |
| Maturity | Is it production-ready? Community size? |
| Simplicity | Is this the simplest tool that works? |
| Team fit | (Language-agnostic template — defer to project context) |
| Ecosystem | Libraries, tooling, CI/CD support? |
| Scalability | Does it meet the NFRs from the vision? |
| Security | CVE history? Active maintenance? |
| Cost | Licensing? Infrastructure requirements? |

### Simplest Viable Architecture

Always start with the simplest architecture that satisfies the vision's requirements. Complexity is added only when justified by concrete, documented NFRs — not hypothetical future needs.

## Component boundaries

### Defining components

Each component in `docs/architecture/components.md` should specify:
- **Responsibility**: What it does (single sentence)
- **Interface**: How other components interact with it
- **Data ownership**: What data it owns and persists
- **Dependencies**: What it depends on (other components, external services)

### Boundary Rules
- Components communicate through defined interfaces, not internal details
- Data ownership is exclusive — one component owns each data entity
- Cross-cutting concerns (logging, auth, config) are separate shared components
- New dependencies require architectural review (check against ADRs)

## Architecture document structure

```
docs/architecture/
├── README.md            # System context + high-level design
├── adrs/                # Architecture Decision Records
├── tech-stack.md        # Chosen technologies with rationale
├── components.md        # Component breakdown and boundaries
├── data-model.md        # Data entities, storage, relationships
├── project-setup.md     # Repo structure, build system, dev environment
└── deployment.md        # (optional) CI/CD, infra, health checks, rollback
```

### deployment.md (Optional)

Include when the product has a deployment target beyond local development. Covers:
- CI/CD pipeline design (build → test → stage → prod)
- Infrastructure requirements (compute, storage, networking)
- Health check endpoints and monitoring
- Rollback strategy and blue/green or canary deployment
- Environment configuration management

Each significant decision is cross-referenced with its ADR in
`docs/architecture/adrs/`.

## Git Workflow

### Commit Convention

One commit per story using conventional commit format with the qualified story ID:

```
feat(TH1.E1.US1): implement user login form
fix(TH1.E2.US3): correct session timeout handling
docs(TH1.E3.US1): slim copilot-instructions.md
```

### Branching Strategy (Optional)

For teams that prefer branch isolation:
- **Branch per epic**: `epic/TH1-E1-core-schema` — merge to main when epic is done
- **Main-only**: All commits go directly to the default branch — simpler for solo or small teams
- Choose the model that fits the team size and risk tolerance

### Orchestrator Integration

The orchestrator should create a commit after each story transitions to `done`, following the commit convention above.

## Dependency Management

### Lockfiles
- Always commit lockfiles (`package-lock.json`, `yarn.lock`, `Pipfile.lock`, `go.sum`, etc.)
- Lockfiles ensure reproducible builds across environments

### Version Pinning
- Pin direct dependencies to exact versions or narrow ranges
- Let lockfiles handle transitive dependency resolution
- Document version constraints in `docs/architecture/tech-stack.md`

### Update Strategy
- Review dependency updates at epic boundaries (not per-story)
- Use automated tools (Dependabot, Renovate) for security patches
- Major version bumps require an ADR documenting the upgrade rationale
