# Campaign Experiments and Reports

This document collects the major CGE experiment campaigns that were run through the
repo-local lab, what each campaign was trying to prove, and what the recorded
reports actually said.

It is the shortest path from "does CGE really help?" to the concrete artifact
trail that informed VP5 and VP6.

## Why these campaigns exist

VP4 turned CGE into a local experiment lab. VP5 tightened the telemetry contract
so token claims would be measured instead of guessed. Together, those phases
made it possible to run real graph-vs-baseline comparisons and then decide what
to improve next from evidence instead of intuition.

Reference product docs:

- `docs/vision_of_product/VP4-experimental-evidence-lab/README.md`
- `docs/vision_of_product/VP5-token-instrumented-lab/README.md`
- `docs/plan/RELEASE-v0.3.0.md`

## Campaign ladder

### 1. Repo dogfooding baseline

Purpose:

- prove the repo-local `graph lab` lifecycle was reproducible
- validate report generation on a tiny, inspectable task set

Scope:

- two repo-first delegated-workflow tasks
- graph-backed vs baseline conditions
- illustrative scoring inputs

Key references:

- `docs/themes/TH4-experimental-evidence-lab/epics/E3-evaluation-reporting-and-dogfooding/repo-dogfooding-example.md`
- repo-local artifacts under `.graph/lab/dogfooding/`

What it established:

- the harness could run end-to-end (`init -> run -> evaluate -> report`)
- CGE could preserve the artifact trail needed for later controlled studies
- the sample was intentionally too small for broad product claims

### 2. VP5 instrumentation shakedown

Purpose:

- verify that token telemetry was being captured honestly before scaling volume
- check that the report pipeline behaved correctly with measured usage

Recorded summary:

- campaign: `vp5-scaled-campaign`
- run count: `12`
- scored paired tasks: `2`
- report artifact: `.graph/lab/reports/report-20260327t094527z.json`

Observed paired results:

| Task | Scored pairs | Mean token delta (graph - baseline) | Quality delta | Resumability delta |
| --- | ---: | ---: | ---: | ---: |
| `repo-lab-reporting` | 3 | `-1,066.67` | `+0.03` | `+0.0467` |
| `repo-workflow-kickoff-handoff` | 3 | `-1,193.33` | `+0.0333` | `+0.0733` |

Interpretation:

- the telemetry pipeline was healthy enough to trust the numbers
- graph-backed kickoff showed small early wins on both sampled tasks
- no warnings or missing-data caveats remained in this shakedown run

### 3. Live Copilot CLI complete-telemetry rerun

Purpose:

- confirm that live delegated runs could carry complete measured usage
- inspect whether the graph advantage survived contact with the real runtime

Recorded summary:

- campaign: `vp5-live-copilot-cli-complete-rerun-20260327t113431z`
- canonical runs: `4`
- superseded overlapping runs: `1`
- report artifact: `.graph/lab/reports/report-20260327t114825z.json`

Headline metrics:

- grouped total-token delta: `-114,950`
- workflow kickoff/handoff pair delta: `-265,564`
- reporting pair delta: `+35,664`
- grouped quality delta: `+0.03`
- grouped resumability delta: `+0.055`

Interpretation:

- graph-backed kickoff was already showing a large win on workflow-heavy tasks
- reporting/synthesis was the first clear warning area
- this rerun removed the earlier partial-telemetry warning and made the signal decision-grade enough to justify a larger campaign

### 4. VP5 decision-grade dry run

Purpose:

- validate the frozen campaign matrix before spending the full run budget

Recorded summary:

- campaign: `vp5-decision-grade-campaign`
- dry-run volume: `24` runs
- successes: `24`
- failures: `0`

Interpretation:

- the matrix and harness were stable enough to scale
- no further instrumentation blocker remained before the full batch

### 5. VP5 decision-grade full campaign

Purpose:

- measure graph-backed vs baseline behavior across a broader, blocked paired matrix
- cover task families instead of relying on one or two showcase tasks

Matrix design:

- study type: `blocked_paired_live_experiment`
- tasks: `12`
- conditions: `with-graph`, `without-graph`
- model: `gpt-5.4`
- topology: `delegated-parallel`
- runtime control: same seed per pair, same prompt template, kickoff brief as the only condition difference

Execution summary:

- planned full volume: `192` runs (`12 tasks x 8 seeds x 2 conditions`)
- executed in the recorded batch: `168` attempted runs (`7` seeds)
- completed runs included in summary: `156`
- failures excluded from scoring: `12`
- debrief artifact: `.graph/lab/dogfooding/generated/vp5-decision-grade-campaign/full-volume-20260327t133931z/campaign-debrief.json`
- report artifact: `.graph/lab/reports/report-20260327t193838z.json`

Global result:

- mean token delta across tasks: `-8,315`
- graph token wins: `7`
- baseline token wins: `5`
- graph win rate by task: `58.3%`
- quality signal: flat under the automated rubric
- resumability signal: flat under the automated rubric

Family-level result:

| Task family | Tasks | Scored pairs | Mean token delta | Verdict |
| --- | ---: | ---: | ---: | --- |
| `write_producing` | 2 | 11 | `-40,576` | graph preferred |
| `troubleshooting_diagnosis` | 3 | 18 | `-7,748` | graph marginal |
| `verification_audit` | 4 | 27 | `-2,089` | mixed |
| `reporting_synthesis` | 3 | 19 | `+4,325` | baseline marginal |

Representative task-level results:

- strongest graph win: `diagnose-workflow-start-low-context` at `-163,560.67`
- strongest verification win: `audit-query-retrieval-provenance` at `-146,814.57`
- strongest graph regression: `diagnose-contradiction-resolution-path` at `+203,796.17`
- reporting was split: `report-context-projection-under-budget` won for graph, while `repo-lab-reporting` and `report-diff-revision-provenance` favored baseline

Interpretation:

- CGE was clearly worth keeping for write-producing work
- troubleshooting looked promising but needed better retrieval discipline
- reporting and synthesis were the main regression bucket
- a single bad retrieval could erase many smaller wins, so variance control became the real product problem

## What the campaign changed in product direction

The full VP5 campaign did not produce the simplistic answer "graph good" or
"graph bad." It produced a more useful answer:

1. graph-backed kickoff is already a strong fit for write-producing work
2. false-positive retrieval is the main way CGE hurts
3. reporting/synthesis should not receive the same kickoff policy as implementation work
4. confidence, provenance, and explicit operator control matter as much as raw retrieval

That directly became TH6:

- task-family classification
- family-specific retrieval policies
- inclusion reasons per kickoff entity
- confidence-gated advisory kickoff
- explicit `--kickoff-mode auto|minimal|none`

See:

- `docs/vision_of_product/VP6-precision-governed-advisory-kickoff/README.md`
- `docs/ADRs/ADR-016-precision-governed-advisory-kickoff.md`
- `docs/plan/RELEASE-TH6.md`

## Cross-model survey on the campaign evidence

After the decision-grade batch, the recorded evidence was also pushed through a
consultative model survey covering five high-end coding models.

Consensus trend:

- unanimous yes on continued investment in CGE
- unanimous agreement that retrieval precision is the key lever
- strong support for advisory, suppressible kickoff instead of mandatory context injection
- consistent warning that reporting/synthesis tasks are most exposed to anchoring from irrelevant retrieved context

The strongest shared recommendation was to treat precision control as the
product, not as a secondary tuning detail.

That survey is the reason VP6 prioritized precision-governed advisory kickoff
over broader evaluation infrastructure work.

## Artifact map

Most raw campaign artifacts are intentionally repo-local rather than committed.
The important paths are:

- `.graph/lab/dogfooding/generated/vp5-scaled-campaign/`
- `.graph/lab/dogfooding/generated/vp5-live-copilot-cli-complete-rerun-20260327t113431z/`
- `.graph/lab/dogfooding/generated/vp5-decision-grade-campaign/`
- `.graph/lab/reports/report-20260327t094527z.json`
- `.graph/lab/reports/report-20260327t114825z.json`
- `.graph/lab/reports/report-20260327t193838z.json`

These artifacts are the machine-readable audit trail behind the summaries above.
