# Themes

This directory contains the implementation plan organized as themes, epics, and user stories.

## Structure

```
themes/
├── TH1-<theme-name>/
│   ├── README.md                       # Theme overview
│   └── epics/
│       ├── E1-<epic-name>/
│       │   ├── README.md               # Epic overview
│       │   └── stories/
│       │       ├── US1-<story>.md       # User story (hybrid BDD)
│       │       └── US2-<story>.md
│       └── E2-<epic-name>/
│           ├── README.md
│           └── stories/
│               └── US1-<story>.md
├── TH2-<theme-name>/
│   └── ...
```

## Conventions

| Entity | Pattern | Maps to |
|--------|---------|---------|
| Theme | `TH<n>-<slug>/` | One or more `VP<n>` in vision_of_product (1:N mapping) |
| Epic | `E<m>-<slug>/` | A coherent unit of deliverable functionality |
| Story | `US<l>-<slug>.md` | One implementable unit of work |

## Story File Format

Each user story file contains YAML frontmatter with metadata (id, title, type, priority, size, acceptance criteria, dependencies), followed by the story body with acceptance criteria and BDD scenarios. Status lives only in `docs/plan/backlog.yaml`. See the `bdd-stories` skill for the canonical template.

## Lifecycle

1. **product-owner** agent creates these directories and files during planning
2. **orchestrator** agent sequences stories and tracks status in `docs/plan/backlog.yaml`
3. **orchestrator** produces changelogs and release notes at epic/theme completion
