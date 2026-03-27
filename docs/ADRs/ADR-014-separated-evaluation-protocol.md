# ADR-014: Separated evaluation protocol for quality scoring

## Status
Proposed

## Context

VP4's product principles explicitly require separating execution from judgment:

> Separate execution from judgment: benchmark runs and quality evaluation
> should be distinct steps so scoring can be blinded where possible.

If quality scoring happens inside the run execution path, several problems
follow:

- evaluators cannot be blinded to experimental condition
- scoring criteria become entangled with run machinery
- it becomes harder to re-evaluate the same runs under improved rubrics
- token-reduction wins can mask quality regressions

VP3's benchmark harness bundled quality proxies into the benchmark report
(ADR-011). VP4 needs a cleaner separation so that evaluation is an independent
step that reads run artifacts and produces evaluation records without knowledge
of which condition produced them.

## Decision

Architecturally separate evaluation from run execution:

1. **Run execution** (`graph lab run`) captures telemetry and outcome artifacts
   but does **not** score quality, success, or resumability
2. **Evaluation** is a distinct step that reads run artifacts and produces
   evaluation records stored separately in `.graph/lab/evaluations/`
3. **Evaluation records** are linked to run IDs but stored outside the run
   record to preserve the execution/judgment boundary
4. **Blinding support** — the evaluation step should be able to present run
   outcomes without revealing the experimental condition, so human or automated
   evaluators judge outputs on merit

The evaluation record should capture at minimum:

- run ID reference
- success/failure against acceptance criteria
- quality score or rubric result
- resumability score
- human intervention count
- evaluator identity (human or automated)
- evaluation timestamp

VP4 should support both:

- **automated evaluation** through programmatic rubrics applied to run artifacts
- **human evaluation** through structured scoring forms that can be
  condition-blind

## Consequences

### Positive
- Enables blinded evaluation, reducing confirmation bias
- Allows re-scoring runs under improved rubrics without re-execution
- Makes quality and efficiency separable in reports
- Supports both human and automated evaluation paths

### Negative
- Adds a two-phase workflow (run then evaluate) where a single phase would be
  simpler
- Blinding requires careful artifact presentation to strip condition metadata
- Automated rubrics may still embed implicit biases

### Risks
- Risk: evaluation step is skipped in practice, leaving runs unscored
  - Mitigation: `graph lab report` should warn when runs lack evaluation records
    and clearly distinguish scored from unscored comparisons
- Risk: blinding is impractical for some metrics (e.g. token usage reveals
  condition)
  - Mitigation: separate quality/success evaluation (blindable) from efficiency
    metrics (inherently condition-aware); reports should combine both

## Alternatives Considered

### Inline scoring during run execution
- Pros: simpler single-step workflow, immediate results
- Cons: prevents blinding, couples scoring to execution, cannot re-evaluate
- Rejected because: VP4 principles explicitly require execution/judgment
  separation

### Fully automated evaluation only
- Pros: no human scoring workflow to design
- Cons: quality and resumability judgments may require human nuance, especially
  for non-trivial tasks
- Rejected because: VP4 should support human evaluation where automated rubrics
  are insufficient, while still making automated scoring the default path

### External evaluation service
- Pros: richer evaluation tooling, collaborative scoring
- Cons: hosted complexity, breaks local-first principle
- Rejected because: VP4 evaluation must work from local artifacts with no
  external dependencies
