# TH1.E2 — Graph Persistence and Provenance

## Epic Goal

Persist native graph payloads into Kuzu using the MVP's entity-centric model
while enforcing provenance and revision-aware graph hygiene.

## Stories

- `TH1.E2.US1` — Persist entities and relationships from native writes
- `TH1.E2.US2` — Store reasoning units and agent sessions with provenance
- `TH1.E2.US3` — Support graph updates and revision anchors

## Done When

- `graph write` stores valid payloads in Kuzu
- persisted data includes required provenance
- graph cleanup workflows can update data while preserving revision anchors for diffing
