---
id: TH1.E4.US5
title: "Add UX/accessibility review checklist"
agents: [developer, reviewer]
skills: [code-quality]
acceptance-criteria:
  - AC1: "code-quality skill includes an optional UX/accessibility section in the review checklist"
  - AC2: "Checklist covers WCAG basics, keyboard navigation, color contrast"
  - AC3: "Section is clearly marked as applicable only to UI projects"
depends-on: []
---

# TH1.E4.US5 — Add UX/Accessibility Review Checklist

**As a** product builder with a UI, **I want** the reviewer's checklist to include accessibility checks, **so that** UI products meet basic usability and accessibility standards.

## Acceptance Criteria

- [ ] AC1: `code-quality` skill includes an optional UX/accessibility section in the review checklist
- [ ] AC2: Checklist covers WCAG basics, keyboard navigation, color contrast
- [ ] AC3: Section is clearly marked as applicable only to UI projects

## BDD Scenarios

### Scenario: Accessibility checklist present
- **Given** the review checklist in `code-quality/SKILL.md`
- **When** I read the checklist sections
- **Then** there's a section for UX/accessibility review

### Scenario: Checklist covers key areas
- **Given** the UX/accessibility section
- **When** I read the items
- **Then** it includes WCAG compliance, keyboard navigation, and color contrast

### Scenario: Section marked as optional
- **Given** the UX/accessibility section
- **When** I read the header
- **Then** it indicates this applies only when the project has UI components
