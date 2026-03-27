# VP4 — Experimental Evidence Lab

> Status: accepted. Architecture artefacts derived from this VP.

## Vision Summary

Build the fourth product phase of the Cognitive Graph Engine around one practical
goal: move from plausible claims about graph-backed agent workflow to credible,
repeatable experimental evidence.

VP1 gave CGE a repo-local graph memory. VP2 made that memory healthier. VP3
embedded the graph into delegated-task kickoff and handoff. VP4 should turn that
workflow into something we can test scientifically, compare across models and
conditions, and improve based on evidence rather than anecdotes.

The result should be a local, inspectable experiment lab for agent workflow
benchmarking.

## Product Intent

Today we can say the graph-backed workflow is useful, and we now have the
golden-path tooling to exercise it. But we still lack a rigorous answer to
questions like:

- does the graph actually reduce total token consumption?
- does it reduce restart and handoff cost, or only move tokens around?
- does it help equally across different models, or only some?
- does it improve quality, or only efficiency?
- when does the graph help, and when is it just extra ceremony?

Without an experimental harness, these remain intuitive judgments.

VP4 exists to turn CGE into an **evidence-producing workflow lab**:

- define comparable task runs
- execute controlled experiments across conditions
- capture machine-readable telemetry
- evaluate outcome quality and resumability
- produce reports with effect sizes, uncertainty, and actionable insight

## Core Hypothesis

The narrow VP4 hypothesis is:

> **For non-trivial delegated software tasks, graph-backed kickoff and handoff
> reduce total recovery cost and improve resumability without reducing task
> success, and this effect is measurable across repeated controlled runs.**

The stronger version of the hypothesis is not assumed. VP4 should be able to
show where the graph helps, where it is neutral, and where it may impose too much
ceremony.

## Primary Outcome

Turn CGE from a workflow substrate into a **measurable workflow science tool**.

At the end of VP4, a maintainer should be able to:

1. define a benchmark suite of realistic repo tasks
2. run the same tasks under graph-backed and non-graph conditions
3. vary models, session topology, and task families in a controlled way
4. collect local machine-readable metrics for cost, quality, and resumability
5. generate a report that supports scientific comparison rather than a demo-only narrative

## Primary Users

Primary users are still AI-agent workflows, but VP4 now explicitly serves the
human experiment designer too.

VP4 is for:

- maintainers testing whether CGE actually improves agent craft
- agents or orchestrators launching controlled benchmark runs
- reviewers comparing workflow conditions across models
- future repo adopters who want evidence before embedding CGE into their workflow

## Core Jobs To Be Done

1. Let a maintainer define a benchmark task corpus with fixed acceptance criteria.
2. Let the system assign controlled experimental conditions for each task run.
3. Let a run execute with a declared model, workflow mode, and session topology.
4. Let the system capture run telemetry across parent and delegated sessions.
5. Let the system preserve enough artifacts to audit and replay what happened.
6. Let the system score success, quality, and resumability separately from raw token cost.
7. Let the system produce scientifically legible reports: paired comparisons,
   variance, uncertainty, and practical recommendations.

## Product Principles

- **Evidence over folklore**: claims about token savings or quality improvement
  should be backed by repeatable measurements.
- **Control before scale**: prefer a smaller, better-controlled task suite over a
  large noisy benchmark zoo.
- **Paired comparisons first**: compare the same task under multiple conditions
  before making pooled claims.
- **Measure quality and cost together**: token reduction without acceptable task
  quality is not success.
- **Auditability matters**: every benchmark result should be reproducible from
  local artifacts, prompts, settings, and outputs.
- **Randomization over anecdotal ordering**: condition order should not quietly
  bias outcomes.
- **Separate execution from judgment**: benchmark runs and quality evaluation
  should be distinct steps so scoring can be blinded where possible.
- **Stay local and inspectable**: the lab should work from repo-local artifacts,
  not hosted black-box telemetry.
- **Model plurality matters**: the protocol should support multiple LLMs without
  assuming one provider's notion of usage or success.
- **Scientific humility**: VP4 should be able to report null or mixed results, not
  only positive stories for the graph.

## VP4 Scope

### Included

- a local benchmark-suite definition format for repo tasks
- condition manifests covering:
  - with graph-backed workflow
  - without graph-backed workflow
  - model identity
  - session topology (single-session vs delegated/parallel)
- machine-readable run telemetry and artifact capture
- local orchestration support for repeated benchmark execution
- report generation for paired and grouped comparisons
- quality and resumability scoring support
- benchmark summaries that surface uncertainty, not just means

### Excluded

- global public leaderboards
- hosted benchmark telemetry backends
- autonomous prompt optimization loops in VP4
- arbitrary internet-scale benchmark task ingestion
- broad multi-repo federation of benchmark data
- proving all causal claims beyond what the measured task suite supports

## VP4 Command Surface

VP4 should add an experiment-oriented surface, likely under one command group:

- `graph lab init`
- `graph lab run`
- `graph lab report`

The exact naming can still be refined in architecture, but the intent is:

- define experiment assets
- execute controlled runs
- analyze and report results

These commands should build on VP3 workflow primitives instead of replacing them.

## Command Intent

### `graph lab init`

Create or refresh the local experiment assets needed to run a benchmark suite in
the current repo.

It should install:

- a benchmark-suite manifest
- condition definitions
- run artifact directories
- evaluation scaffolding
- stable schema contracts for telemetry and report outputs

### `graph lab run`

Execute a controlled set of benchmark runs.

Each run should declare at minimum:

- task ID
- experimental condition
- model
- session topology
- prompt/workflow variant
- seed or run identifier

The run artifact should preserve:

- kickoff inputs
- delegated session structure
- writeback outputs
- token/usage measurements
- timing and retry information
- outcome artifacts

### `graph lab report`

Aggregate completed runs into a scientific report.

The report should support:

- paired task comparison
- grouped comparison by model or topology
- success and failure rates
- token and step distributions
- resumability and handoff quality comparisons
- effect-size oriented summaries
- uncertainty intervals or equivalent confidence summaries

## Experimental Protocol Expectations

VP4 should make the following workflow natural:

1. Define a benchmark corpus of realistic non-trivial repo tasks.
2. Freeze the repo state and acceptance criteria for a benchmark batch.
3. Assign conditions across the same tasks:
   - graph-backed workflow
   - no graph-backed workflow
   - multiple models
   - multiple repetitions
4. Randomize or counterbalance condition ordering.
5. Run the tasks while capturing parent-session and delegated-session telemetry.
6. Score outputs for success and quality, ideally blind to condition.
7. Produce a report that distinguishes:
   - efficiency
   - quality
   - resumability
   - variance and uncertainty
8. Derive recommendations about where graph-backed workflow is worth the ceremony.

## Experimental Design Expectations

VP4 should support scientifically serious comparisons, including:

### Paired task design

The same benchmark task should be executed under multiple conditions so within-task
comparisons dominate over loose between-task anecdotes.

### Blocking factors

The system should treat the following as explicit factors, not informal notes:

- task family
- model
- graph condition
- session topology
- run repetition

### Randomization and counterbalancing

Order effects matter. The protocol should support randomized or counterbalanced
condition ordering so later runs do not unfairly benefit from operator familiarity
or warmed intuition.

### Blind or separated evaluation

Where feasible, quality scoring should be separated from run execution and blinded
to condition, so evaluators judge outcomes rather than the hypothesis.

### Null-result support

The reporting surface should make it easy to say:

- the graph helped
- the graph did not matter
- the graph hurt under these conditions

without forcing all results into a positive narrative.

## Candidate Metrics

VP4 should treat metrics as a bundle, not a single score.

### Primary metrics

- total token usage across all sessions in a run
- task success or failure against fixed acceptance criteria
- human intervention count
- resumability score for the next agent
- wall-clock duration
- number of retries or repair loops

### Secondary metrics

- token usage until first meaningful action
- delegated handoff completeness
- number of files changed
- benchmark condition compliance
- writeback completeness
- variance across repetitions

### Derived metrics

- token delta between graph and no-graph conditions
- success-adjusted token efficiency
- handoff efficiency
- restart penalty reduction
- model-specific effect sizes

## Example End-to-End Workflow

1. A maintainer defines a benchmark suite of delegated engineering tasks.
2. `graph lab init` installs the local manifests, schemas, and evaluation assets.
3. The maintainer launches a batch with multiple models and both graph conditions.
4. The system executes repeated runs while preserving machine-readable artifacts.
5. Evaluators score outcomes for correctness and resumability.
6. `graph lab report` generates a report showing where graph-backed workflow helps,
   where it is neutral, and where it adds avoidable cost.
7. The team uses that evidence to refine prompts, hooks, and workflow policy.

## Success Criteria

### Priority 1

- The repo can run controlled benchmark batches locally with stable artifacts.
- Reports compare graph-backed and non-graph conditions for the same task set.
- The output makes quality, cost, and resumability trade-offs visible together.

### Priority 2

- The harness supports multiple models and delegated-session topologies.
- The reporting layer surfaces effect sizes and uncertainty, not just raw totals.
- The benchmark workflow is reproducible enough that later repos can adopt it.

### Priority 3

- The lab produces evidence strong enough to change workflow policy in this repo.
- The team can identify where graph-backed workflow should be default, optional, or
  avoided.

## Risks and Failure Modes

- token metrics may be inconsistent across model providers
- task difficulty variance may swamp workflow effects
- benchmark scoring may be biased if evaluation is not blinded
- operator involvement may contaminate otherwise clean comparisons
- overly broad benchmark suites may produce noisy conclusions

VP4 should reduce these risks through explicit schemas, paired design, artifact
capture, and disciplined evaluation rather than by pretending the risks do not exist.
