# Repo Dogfooding Example for `graph lab`

This repository now includes a small, reproducible dogfooding harness for the
experiment lab under `.graph/lab/`.

## What is committed

- `.graph/lab/suite.json` — two delegated-workflow tasks drawn from this repo's
  own implementation history
- `.graph/lab/conditions.json` — the graph-backed and baseline conditions used
  for the comparison
- `.graph/lab/dogfooding/baseline-v1.json` — the repo-local experiment manifest
  describing the selected tasks, run plan, illustrative scoring inputs, and
  sample-artifact paths
- `.graph/lab/dogfooding/run-baseline.py` — a minimal helper that runs the full
  lab lifecycle (`init → run → evaluate → report`) from the committed manifest
- `.graph/lab/runs/`, `.graph/lab/evaluations/`, `.graph/lab/reports/` — a
  tiny baseline artifact set that future operators can inspect before running a
  fresh experiment

## Selected tasks

The suite intentionally stays small and repo-first:

1. `repo-workflow-kickoff-handoff`
   - source history: TH3 delegated-workflow repo dogfooding
   - acceptance reference:
     `docs/themes/TH3-graph-backed-delegated-workflow/epics/E3-benchmark-and-repo-dogfooding/stories/US3-repo-workflow-snippets-and-hook-verification.md`
2. `repo-lab-reporting`
   - source history: TH4 report generation work
   - acceptance reference:
     `docs/themes/TH4-experimental-evidence-lab/epics/E3-evaluation-reporting-and-dogfooding/stories/US2-lab-report-command.md`

This is enough to exercise the harness on realistic delegated-workflow tasks
without pretending that two tasks are a benchmark campaign.

## Reproduce the flow

From the repo root:

```bash
python3 .graph/lab/dogfooding/run-baseline.py
```

The helper performs:

1. `graph lab init`
2. `graph lab run` over the committed task and condition selection
3. `graph lab evaluate` for each produced run using the committed illustrative
   score inputs in `.graph/lab/dogfooding/baseline-v1.json`
4. `graph lab report` over the generated run IDs

It writes machine-readable command responses under
`.graph/lab/dogfooding/generated/` and prints a final JSON summary containing the
new run IDs and report artifact path.

## Baseline example artifacts

The committed baseline manifest points at one inspected sample report:

- `.graph/lab/reports/report-20260326t232306z.json`

That report shows the kind of evidence we want from the harness:

- paired task comparisons
- token deltas
- quality and resumability deltas
- explicit negative results where the graph-backed path underperformed
- explicit limitations when the sample is too small for strong conclusions

## Limitations and honesty notes

This example is intentionally conservative.

- It uses **two tasks, one model, one topology, and one scored pair per task**.
- The committed evaluation scores are **illustrative baseline inputs for harness
  verification**, not strong empirical claims about CGE effectiveness.
- The sample report is useful as a baseline example for future experiments, but
  it should not be presented as proof that graph-backed workflow is generally
  better.
- Any stronger recommendation should come from a larger suite, repeated seeds,
  and preferably blinded human or stronger automated evaluation.

In short: this setup proves the repo-local lab workflow is reproducible and
inspectable here, while being explicit that the current sample is far too small
for sweeping claims.
