# TH8.E4 — Evaluated Graph Memory Discipline

Apply evaluation discipline to graph memory (ADR-022) by gating
workflow-mediated writes on evaluator confidence, down-ranking stale or
low-confidence graph state during retrieval, and preserving backward
compatibility for raw `graph write` commands.
