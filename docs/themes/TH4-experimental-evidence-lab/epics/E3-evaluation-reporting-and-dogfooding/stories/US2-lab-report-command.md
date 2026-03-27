---
id: TH4.E3.US2
title: "Add `graph lab report` with paired comparisons and uncertainty"
type: standard
priority: high
size: L
agents: [developer]
skills: [bdd-stories]
acceptance-criteria:
  - AC1: "`graph lab report` aggregates run records and evaluation scores into a machine-readable report with paired within-task comparisons and grouped comparisons by model or topology."
  - AC2: "Reports include effect-size summaries, uncertainty intervals, and explicit null-result or negative-result sections."
  - AC3: "Reports warn when runs lack evaluation records and clearly distinguish scored from unscored comparisons."
depends-on: [TH4.E3.US1]
---
# TH4.E3.US2 — Add `graph lab report` with paired comparisons and uncertainty

**As a** maintainer reviewing experiment results, **I want** a scientific-style
report that surfaces paired comparisons, effect sizes, and uncertainty, **so
that** I can make evidence-based decisions about graph-backed workflow value.

## Acceptance Criteria

- [ ] AC1: `graph lab report` aggregates run records and evaluation scores into a machine-readable report with paired within-task comparisons and grouped comparisons by model or topology.
- [ ] AC2: Reports include effect-size summaries, uncertainty intervals, and explicit null-result or negative-result sections.
- [ ] AC3: Reports warn when runs lack evaluation records and clearly distinguish scored from unscored comparisons.

## BDD Scenarios

### Scenario: Generate a paired within-task comparison report
- **Given** the run ledger contains completed runs for the same task under both graph-backed and baseline conditions with evaluation scores
- **When** a maintainer runs `graph lab report`
- **Then** the report includes a paired comparison for that task showing token delta, quality effect size, resumability effect size, and uncertainty intervals

### Scenario: Generate a grouped comparison by model
- **Given** the run ledger contains runs for the same task and conditions across two different models
- **When** a maintainer runs `graph lab report`
- **Then** the report includes a grouped comparison that shows model-specific effect sizes and identifies whether the graph benefit varies by model

### Scenario: Report null and negative results explicitly
- **Given** the run ledger contains runs where the graph-backed condition showed no improvement or worse performance than baseline
- **When** a maintainer runs `graph lab report`
- **Then** the report surfaces the null or negative results in a dedicated section rather than omitting them

### Scenario: Warn about unscored runs in the report
- **Given** some completed runs in the ledger have no corresponding evaluation records
- **When** a maintainer runs `graph lab report`
- **Then** the report includes a warning listing the unscored run IDs and marks those comparisons as incomplete
