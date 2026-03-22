# ADR-001: Use Go and Cobra for the CLI implementation

## Status
Proposed

## Context

The product is a local, repo-scoped CLI intended for AI agents. It must be easy
to distribute, fast to start, simple to chain in shell pipelines, and light on
operational overhead.

The user explicitly allowed either Go or Rust for the CLI binary.

## Decision

Build the MVP as a Go CLI using Cobra for command structure and the Go standard
library for the rest of the application where practical.

## Consequences

### Positive
- Produces a simple local binary distribution story
- Good fit for stdin/stdout-heavy CLI workflows
- Strong standard library reduces framework load
- Faster MVP iteration than a lower-level systems stack

### Negative
- Less comfortable ecosystem for local semantic inference than Python
- Native database bindings require care
- Cobra adds some framework overhead to a small CLI

### Risks
- Risk: future local ML requirements may exceed Go ecosystem convenience
  - Mitigation: keep retrieval contracts isolated behind internal interfaces

## Alternatives Considered

### Rust
- Pros: excellent performance, strong binaries, strong type safety
- Cons: slower MVP iteration, higher implementation friction
- Rejected because: Go provides a better simplicity-to-capability ratio for
  this MVP

### Python
- Pros: strongest local ML ecosystem
- Cons: weaker single-binary distribution, heavier local environment management
- Rejected because: local CLI distribution and operational simplicity matter
  more than ML convenience in MVP
