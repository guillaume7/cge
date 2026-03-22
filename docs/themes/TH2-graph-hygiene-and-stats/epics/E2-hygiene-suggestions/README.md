# TH2.E2 — Hygiene Suggestions

## Epic Goal

Help agents detect graph disorder safely by returning explainable, structured
cleanup suggestions without mutating the graph by default.

## Stories

- `TH2.E2.US1` — Detect orphan and duplicate-near-identical hygiene candidates
- `TH2.E2.US2` — Detect contradictory facts and propose resolution candidates
- `TH2.E2.US3` — Return machine-readable hygiene suggestion plans

## Done When

- `graph hygiene` suggest mode identifies the agreed classes of graph disorder
- suggestions explain why a candidate was produced
- agents can consume suggestion plans programmatically
