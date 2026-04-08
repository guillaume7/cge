# Evaluation Loop and Lab Signals — VP8

## Intent

VP8 should correct the product's biggest functional gap: the lack of an explicit
evaluator loop.

Without that loop, the product cannot reliably tell whether:

- retrieved context is relevant
- stored graph memory is still trustworthy
- generated guidance should be used, revised, minimized, or rejected

## Evaluation Loop Shape

The seed architecture sketch points in the right direction:

```text
retrieve -> generate -> evaluate -> decide -> update
```

VP8 should keep that shape, but reinterpret it as product behavior instead of
just component structure.

## Minimum Evaluation Questions

Every meaningful loop should be able to ask:

1. Is the retrieved context relevant to the task?
2. Is the retrieved context consistent with other known signals?
3. Is the current output likely to help task completion?
4. Is confidence high enough to continue, or should the system minimize,
   backtrack, or abstain?
5. Should this result update memory, or would that amplify drift?

## Decision Outcomes

VP8 should normalize a small set of honest outcomes:

- **continue** when evidence is good enough
- **backtrack** when the current path is degrading quality
- **minimal** when only narrow, low-risk guidance should be injected
- **abstain** when the harness cannot justify strong guidance
- **write** only when the new state is strong enough to improve future continuity

These outcomes matter because the current failure mode is not only missing
answers. It is confidently carrying forward weak or stale state.

## Attribution Requirements

The evaluator loop should preserve enough evidence to explain:

- why a context bundle was selected
- why a context bundle was trimmed or rejected
- why an output was accepted or sent back for revision
- why memory was updated, skipped, or rewritten

Without this attribution, lab results will not explain whether token reductions
came from real improvement or from suppressed behavior that harmed quality.

## Lab Signals

The primary VP8 success signal is:

> **Lab experiments should show a clear decline in token consumption when using
> the CGE harness.**

That signal is necessary but not sufficient. The lab should also preserve enough
evidence to inspect:

- relevance of injected context
- rate of minimized or abstained runs
- cases where backtracking improved or failed to improve results
- whether memory updates reduced or amplified drift
- whether lower token use still produced useful task outcomes

## Risks To Watch

- lower token use caused by over-abstaining rather than better guidance
- evaluator scores that become arbitrary and hard to trust
- graph rewrite policies that erase useful continuity in the name of hygiene
- excessive loop complexity that costs more than it saves

## Open Questions

- What minimum scoring dimensions are enough for VP8 to be useful in practice?
- Which lab task families best expose whether the new loop beats the old graph-led
  behavior?
- When should memory be rewritten versus merely down-ranked or ignored?
