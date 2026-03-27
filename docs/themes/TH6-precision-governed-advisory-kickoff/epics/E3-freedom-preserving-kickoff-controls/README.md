# TH6.E3 — Freedom-Preserving Kickoff Controls

## Epic Goal

Preserve agent freedom by making kickoff explicitly optional, minimally invasive
when requested, and compatible with the current machine-readable workflow path.

## Stories

- `TH6.E3.US1` — Add explicit no-kickoff and minimal-kickoff controls
- `TH6.E3.US2` — Expose advisory kickoff state without breaking workflow compatibility
- `TH6.E3.US3` — Gracefully degrade kickoff on sparse repos and ambiguous tasks

## Done When

- agents can request no kickoff or minimal kickoff explicitly
- workflow start exposes advisory state in a stable machine-readable form
- sparse repos and ambiguous tasks degrade gracefully instead of producing noisy context

