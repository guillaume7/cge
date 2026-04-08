# ADR-006: Use a hybrid retrieval pipeline with local text relevance, graph traversal, and context projection

## Status
Proposed

## Context

The product must support both graph-aware and semantic retrieval while staying
local and offline. The MVP also prioritizes task continuity first and token
reduction second, which means retrieval should favor trustworthy useful context
over exhaustive dumps.

## Decision

Use a hybrid retrieval pipeline composed of:

1. graph-structured candidate retrieval from Kuzu
2. local text-relevance retrieval over indexed entity text
3. merged ranking and neighborhood expansion
4. token-budget-aware context projection
5. explanation traces for why results were returned

For MVP, implement the local semantic side using embedded BM25/FTS-style text
ranking rather than dense local embeddings.

## Consequences

### Positive
- Satisfies offline retrieval needs with manageable complexity
- Balances graph structure with text relevance
- Supports `graph explain` naturally through explicit ranking traces
- Keeps token budgeting explicit in the projection phase

### Negative
- BM25/FTS is weaker than dense embeddings for semantic similarity
- Maintaining a second local index adds synchronization work
- Retrieval quality will depend on good titles, summaries, and tags

### Risks
- Risk: users may expect stronger semantic recall than BM25 provides
  - Mitigation: design retrieval interfaces so a future local embedding scorer
    can be added without rewriting command contracts

## Alternatives Considered

### Graph traversal only
- Pros: simplest architecture, one store only
- Cons: poor recall for natural-language task phrasing
- Rejected because: it does not satisfy the vision's semantic retrieval needs

### Local dense embeddings in MVP
- Pros: stronger semantic matching
- Cons: much higher binary, model, and runtime complexity
- Rejected because: it is not the simplest viable offline MVP architecture
