# CGE v2 – Minimal Viable Architecture

## Overview
A lightweight agent system combining:
- Iterative control loop
- Evaluation feedback
- Graph-based memory (supporting role)

---

## Core Loop

```python
state = init_state(task)

for step in range(MAX_STEPS):

    context = graph.retrieve(state)

    output = generator.run(task, context)

    score = evaluator.score(task, output)

    decision = decide(score, state)

    graph.update(output, score)

    if decision == "done":
        break

    state = update_state(state, output, score)
```

---

## Components

### 1. Controller Loop
- Max iterations
- Stop on threshold or stagnation

### 2. Generator
Produces structured output:
```json
{
  "answer": "...",
  "entities": [],
  "relations": [],
  "confidence": 0.0
}
```

### 3. Evaluator
Scoring:
- correctness
- relevance
- consistency

```python
final_score = weighted_sum(score)
```

### 4. Decision Engine
```python
if score > 0.85:
    return "done"
elif score < previous_score:
    return "backtrack"
else:
    return "continue"
```

### 5. Graph Memory
Schema:

Node:
- id
- type
- value
- score

Edge:
- source
- target
- relation
- score

---

## Data Flow

Graph → Generator → Evaluator → Decision → Graph

---

## Minimal Stack

- Python
- networkx
- SQLite (optional)
- LLM API

---

## Key Principles

- Loop drives intelligence
- Evaluation provides truth
- Graph supports memory

---

## Evolution Path

1. Improve evaluation
2. Add embeddings
3. Add roles (generator/critic)
4. Persist graph

---

## Anti-Patterns

- Over-engineered graph schemas
- No evaluation loop
- Too many agents early
