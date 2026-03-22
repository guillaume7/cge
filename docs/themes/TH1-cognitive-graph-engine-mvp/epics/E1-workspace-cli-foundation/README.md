# TH1.E1 — Workspace and CLI Foundation

## Epic Goal

Create the repo-local graph workspace and a chainable CLI foundation that all
later graph capabilities can build on safely.

## Stories

- `TH1.E1.US1` — Initialize a repo-scoped graph workspace
- `TH1.E1.US2` — Add a chainable command surface and repo discovery
- `TH1.E1.US3` — Define and validate the native graph payload contract

## Done When

- `graph init` can bootstrap the repo-local workspace
- command handlers can read from flags, stdin, and files as appropriate
- native JSON payloads are validated consistently before persistence or query work
