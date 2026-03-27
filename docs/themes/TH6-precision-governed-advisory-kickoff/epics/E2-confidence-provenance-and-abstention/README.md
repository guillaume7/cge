# TH6.E2 — Confidence, Provenance, and Abstention

## Epic Goal

Make kickoff outcomes explainable and confidence-aware so the system can abstain
honestly when policy says graph injection is unsafe or weakly supported.

## Stories

- `TH6.E2.US1` — Compute family-aware kickoff confidence and thresholding
- `TH6.E2.US2` — Return one-line inclusion reasons for kickoff entities
- `TH6.E2.US3` — Recommend pull-on-demand when kickoff confidence is low

## Done When

- workflow start computes a confidence signal tied to the selected family policy
- included kickoff entities explain why they were selected
- low-confidence results can abstain and recommend pull-on-demand instead of forcing context injection

