# ADR-005: Use a versioned native JSON payload contract across stdin, stdout, and files

## Status
Proposed

## Context

The CLI must be chainable. Custom agents are expected to be trained to speak
and consume the graph tool's native structured format directly. That requires a
stable machine-readable contract that works the same way in files, stdin, and
stdout.

## Decision

Define a versioned native JSON payload envelope as the canonical interchange
format for:

- `graph write` input
- structured `graph query` output
- structured `graph context` output
- structured `graph explain` output
- file-based payload exchange

## Consequences

### Positive
- Gives agents a stable protocol to emit and consume
- Makes shell chaining natural and lossless
- Simplifies validation and contract testing
- Avoids ad hoc text parsing between tools

### Negative
- JSON can be verbose for very large payloads
- Versioning discipline is required
- Human readability is secondary to machine reliability

### Risks
- Risk: contract drift between commands can break chaining
  - Mitigation: keep a shared schema version and common envelope conventions

## Alternatives Considered

### Free-form text output
- Pros: easy for humans to read
- Cons: brittle for tool interoperability
- Rejected because: machine-to-machine reliability is a core requirement

### YAML as the primary native protocol
- Pros: readable and expressive
- Cons: less robust for strict machine contracts in pipelines
- Rejected because: JSON is the safer default for agent/tool interoperability
